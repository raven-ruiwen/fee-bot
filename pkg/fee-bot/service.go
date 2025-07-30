package fee_bot

import (
	"encoding/json"
	"fee-bot/pkg/notify"
	"fmt"
	"github.com/Logarithm-Labs/go-hyperliquid/hyperliquid"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/imroc/req/v3"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"math"
	"sync"
	"time"
)

type Service struct {
	agentHyper     *hyperliquid.Hyperliquid
	accountHyper   *hyperliquid.Hyperliquid
	tradeCoins     map[string]*Coin
	runInterval    time.Duration
	accountAddress string
	spotAccount    spotAccount
	perpAccount    perpAccount
	notify         *notify.Service
}

type spotAccount struct {
	AvailableUSDC float64
}

type perpAccount struct {
	AccountValue         float64
	CrossAccountLeverage float64
	AvailableUSDC        float64
	TotalNtlPos          float64
}

type orderSetting struct {
	reBalanceRatio     float64
	isAllowedOpenOrder bool
}

type OrderParam struct {
	Coin string
	Size float64
}

func NewService(accountAddress string, agentHyper *hyperliquid.Hyperliquid, accountHyper *hyperliquid.Hyperliquid, coins []*Coin, runInterval time.Duration, notify *notify.Service) *Service {
	s := &Service{}

	s.accountAddress = accountAddress
	s.agentHyper = agentHyper
	s.accountHyper = accountHyper

	nameCoins := make(map[string]*Coin)
	for _, coin := range coins {
		nameCoins[coin.Name] = coin
	}

	s.tradeCoins = nameCoins
	s.runInterval = runInterval
	s.notify = notify

	return s
}

func (s *Service) Init() {
	for _, c := range s.tradeCoins {
		if c.Leverage == 0 {
			c.Leverage = 5 //默认5倍
		}
		resp, err := s.agentHyper.UpdateLeverage(c.MarketPerpId, true, c.Leverage)
		if err != nil {
			logrus.Errorf("[设置合约%dx杠杆Failed][%s] %v", c.Leverage, c.Name, err)
			continue
		}
		logrus.Infof("[设置token%dx合约Leverate][%s] %s", c.Leverage, c.Name, resp.Status)
	}

	//set token decimal
	spotMeta, _ := s.agentHyper.GetSpotMeta()
	for _, u := range spotMeta.Universe {
		for _, coin := range s.tradeCoins {
			if u.Name == coin.MarketSpotId {
				tokenIndex := u.Tokens[0]
				meta := spotMeta.Tokens[tokenIndex]
				//todo 没找到
				logrus.Infof("[查找Decimal][Spot][%s] %d", coin.Name, meta.SzDecimals)
				coin.DecimalSpot = decimal.NewFromInt(int64(meta.SzDecimals))
			}
		}
	}

	perpMeta, _ := s.agentHyper.GetMeta()
	for _, u := range perpMeta.Universe {
		for _, coin := range s.tradeCoins {
			if u.Name == coin.MarketPerpId {
				//todo 没找到
				logrus.Infof("[查找Decimal][Perp][%s] %d", coin.Name, u.SzDecimals)
				coin.DecimalPerp = decimal.NewFromInt(int64(u.SzDecimals))
			}
		}
	}
}

func (s *Service) Run() {
	for {
		logrus.Infof("==========================================================================")
		state, err := s.agentHyper.GetUserState(s.accountAddress)
		if err != nil {
			logrus.Errorf("Error getting user state: %v", err)
			continue
		}
		perpAccountValue := state.CrossMarginSummary.AccountValue
		s.perpAccount.AccountValue = perpAccountValue
		s.perpAccount.CrossAccountLeverage = state.CrossMarginSummary.TotalNtlPos / perpAccountValue
		s.perpAccount.TotalNtlPos = state.CrossMarginSummary.TotalNtlPos
		s.perpAccount.AvailableUSDC = perpAccountValue - state.CrossMarginSummary.TotalMarginUsed

		stateSpot, err := s.agentHyper.GetUserStateSpot(s.accountAddress)
		if err != nil {
			logrus.Errorf("Error getting user state: %v", err)
			continue
		}

		//本轮perp的相关初始化
		for _, position := range state.AssetPositions {
			if coin, ok := s.tradeCoins[position.Position.Coin]; !ok {
				logrus.Fatalf("持有头寸中包含了未设置的币种: %v", position.Position.Coin)
			} else {
				coin.PositionUSD = position.Position.PositionValue
				coin.PositionSize = position.Position.Szi
				coin.PositionMarginUsed = position.Position.MarginUsed
			}
		}
		//本轮spot相关的初始化
		for _, balance := range stateSpot.Balances {
			if balance.Coin == "USDC" {
				s.spotAccount.AvailableUSDC = balance.Total
				logrus.Infof("[Spot][Set Available USDC] %f", balance.Total)
				continue
			}
			for i, coin := range s.tradeCoins {
				if coin.OrderSpotId == balance.Coin {
					s.tradeCoins[i].SpotBalance = balance.Total
					s.tradeCoins[i].SpotEntryNtl = balance.EntryNtl
					logrus.Infof("[Spot][Set Balance] %s -> %f", balance.Coin, balance.Total)
				}
			}
		}

		logrus.Warnf("[Perp] account value: %.2f, 可用USDC: %f, 杠杆倍数: %.3fx", perpAccountValue, s.perpAccount.AvailableUSDC, s.perpAccount.CrossAccountLeverage)

		if need, toPerp, transferAmount := s.needReBalanceLeverage(); need {
			result, err := s.AccountTransferUSDC(transferAmount, toPerp)
			if err != nil {
				logrus.Errorf("[资金杠杆ReBalance] 目前杠杆: %.3f, 是否为转移到perp：%v, 转移usdc数量：%f, err: %v", s.perpAccount.CrossAccountLeverage, toPerp, transferAmount, err)
			}
			logrus.Warnf("[资金杠杆ReBalance] 目前杠杆: %.3f, 是否为转移到perp：%v, 转移usdc数量：%f, result: %s", s.perpAccount.CrossAccountLeverage, toPerp, transferAmount, result)
			//直接进入下一轮检查
			continue
		}

		if s.needForceLiquidation(s.perpAccount.CrossAccountLeverage) {
			//查找杠杆最多的，先平最多position的
			var liquidationCoin *Coin
			var maxLeverage float64
			for _, c := range s.tradeCoins {
				coinLeverage := c.GetLeverage(s.perpAccount.AccountValue)
				if coinLeverage > maxLeverage {
					liquidationCoin = c
				}
			}

			logrus.Warnf("[强制平仓][选择Token: %s] 仓位目前杠杆率: %.2fx", liquidationCoin.Name, maxLeverage)
			//依旧看买1卖1，慢慢平下去。不强制用usd转换数量
			orderParam, err := s.GetOrderParam(OrderSellSpotBuyPerp, liquidationCoin)
			if err != nil {
				s.LogErrorAndNotifyDev(fmt.Sprintf("[GetOrderParamFailed][%s]: %v", liquidationCoin.Name, err))
				continue
			}
			//平空合约，卖现货
			s.ExecOrder(OrderSellSpotBuyPerp, liquidationCoin, orderParam)
			//直接进入下一轮检查
			continue
		}

		//检查交易币种
		for _, c := range s.tradeCoins {
			//单币下单参数阈值初始化
			coinOrderSettings := s.getOrderSettingsByCoinLeverage(c.GetLeverage(s.perpAccount.AccountValue))
			c.SetOrderSettings(coinOrderSettings)

			var spotPrice, perpPrice float64
			var errSP, errPP error
			priceGroup := &sync.WaitGroup{}

			priceGroup.Add(1)
			go func(spotPrice *float64, errSP *error) {
				sp, errP := s.agentHyper.GetMartketPx(c.MarketSpotId)
				*spotPrice = sp
				*errSP = errP
				priceGroup.Done()
			}(&spotPrice, &errSP)
			priceGroup.Add(1)
			go func(perpPrice *float64, errPP *error) {
				pp, errP := s.agentHyper.GetMartketPx(c.MarketPerpId)
				*perpPrice = pp
				*errPP = errP
				priceGroup.Done()
			}(&perpPrice, &errPP)

			priceGroup.Wait()

			if errSP != nil {
				logrus.Errorf("[%s][ErrGetSpotPrice] %s, skip", c.Name, errSP.Error())
				continue
			}
			if errPP != nil {
				logrus.Errorf("[%s][ErrGetPerpPrice] %s, skip", c.Name, errPP.Error())
				continue
			}

			if !c.SpotPositionEqualWithPerp(spotPrice) {
				s.LogErrorAndNotifyDev(fmt.Sprintf("[%s][头寸核对异常] spot:perp - %f : %f, ratio: %.2f%%", c.Name, c.SpotBalance, -c.PositionSize, (c.SpotBalance-(-c.PositionSize))/(-c.PositionSize)*100))
				s.ReBalanceCoinPosition(c, spotPrice, perpPrice)
				//执行完后跳过其他token直接进行下一轮检查
				break
			}

			//计算仓位占比: (现货购买的usd使用量 + 合约目前的marginUSD) / (现货的USD价值 + 合约accountValue) * 100
			totalSpotEntryUSD := s.GetSpotEntryValue()
			coinUSDRatio := (c.SpotEntryNtl + c.PositionMarginUsed) / (totalSpotEntryUSD + s.perpAccount.AccountValue) * 100
			logrus.Infof("[%s][Perp杠杆率] %.2fx, 能否开仓：%v", c.Name, c.GetLeverage(s.perpAccount.AccountValue), c.OrderSetting.isAllowedOpenOrder)
			if coinUSDRatio > c.PositionMaxRatio {
				logrus.Warnf("[%s][单币开仓比例达到上限] %.2f%%(max %.2f%%)", c.Name, coinUSDRatio, c.PositionMaxRatio)
				//上限后跳过开单检查，直接判断下个coin
				continue
			} else {
				logrus.Infof("[%s][开仓比例] %.2f%%(max %.2f%%), size: %f $%s, $%f", c.Name, coinUSDRatio, c.PositionMaxRatio, c.PositionSize, c.Name, c.PositionUSD)
			}

			priceDiffRatio := (perpPrice - spotPrice) / spotPrice * 100

			logrus.Infof("[%s][差价] 当前差价 %.2f%%（perp: %f : spot: %f）, 开仓差价: %.2f%%, 关仓差价: %.2f%%", c.Name, priceDiffRatio, perpPrice, spotPrice, c.GetAllowOpenPriceDiffRatio(), c.GetAllowClosePriceDiff())
			//check 阈值和是否允许开仓判断也在get action里进行
			action := s.GetCoinOrderAction(c, priceDiffRatio)
			if action != OrderNoAction {
				orderParam, err := s.GetOrderParam(action, c)
				if err != nil {
					logrus.Errorf("[GetOrderParamFailed][%s]: %v", c.Name, err)
					continue
				}
				if math.Abs(orderParam.Size) != 0 {
					s.ExecOrder(action, c, orderParam)
					//执行完后直接进入下一轮，重新检查参数
					break
				}
			}
		}

		time.Sleep(s.runInterval * time.Second)
	}
}

func (ps *orderSetting) SetAllowOpenOrder() {
	ps.isAllowedOpenOrder = true
}

func (ps *orderSetting) SetDenyOpenOrder() {
	ps.isAllowedOpenOrder = false
}

func (s *Service) getOrderSettingsByCoinLeverage(coinLeverage float64) orderSetting {
	var coinOrderSetting orderSetting
	if coinLeverage < 1 {
		coinOrderSetting.reBalanceRatio = -0.2 //todo: 正式的时候修改回来
		coinOrderSetting.SetAllowOpenOrder()
	} else if coinLeverage > 1 && coinLeverage <= 1.5 {
		coinOrderSetting.reBalanceRatio = 0
		coinOrderSetting.SetAllowOpenOrder()
	} else if coinLeverage > 1.5 && coinLeverage <= 1.8 {
		coinOrderSetting.reBalanceRatio = 0.1
		coinOrderSetting.SetAllowOpenOrder()
	} else if coinLeverage > 1.8 && coinLeverage < 2 {
		coinOrderSetting.reBalanceRatio = 0.2
		coinOrderSetting.SetAllowOpenOrder()
	} else if coinLeverage >= 2 && coinLeverage < 2.5 {
		coinOrderSetting.reBalanceRatio = 2
		coinOrderSetting.SetDenyOpenOrder()
	} else {
		// > 2.5
		coinOrderSetting.reBalanceRatio = 2
		coinOrderSetting.SetDenyOpenOrder()
	}
	return coinOrderSetting
}

func (s *Service) needForceLiquidation(crossAccountLeverage float64) bool {
	return crossAccountLeverage >= LeverageForceLiquidation
}

func (s *Service) LogErrorAndNotifyDev(msg string) {
	logrus.Errorf(msg)
	go s.notify.SendMsg("[Fee-Bot Error]", msg)
}

func (s *Service) GetSpotEntryValue() float64 {
	var totalUSDC float64
	for _, c := range s.tradeCoins {
		totalUSDC += c.SpotEntryNtl
	}
	return totalUSDC + s.spotAccount.AvailableUSDC
}

func (s *Service) GetCoinOrderAction(coin *Coin, priceDiffRate float64) OrderAction {
	allowOpenPriceDiff := coin.GetAllowOpenPriceDiffRatio()
	allowClosePriceDiff := coin.GetAllowClosePriceDiff()

	if priceDiffRate > allowOpenPriceDiff && coin.OrderSetting.isAllowedOpenOrder {
		return OrderSellPerpBuySpot
	}
	if priceDiffRate < allowClosePriceDiff {
		return OrderSellSpotBuyPerp
	}
	return OrderNoAction
}

func (s *Service) ExecOrder(direction OrderAction, coin *Coin, orderParam *OrderParam) {
	if direction == OrderSellPerpBuySpot {
		//卖合约，买现货
		resp, err := s.agentHyper.MarketOrder(coin.OrderPerpId, -orderParam.Size, nil)
		if !s.CheckOrder(coin, orderParam, resp, err) {
			return
		}
		resp1, err1 := s.agentHyper.MarketOrderSpot(coin.OrderSpotId, orderParam.Size, nil)
		s.CheckOrder(coin, orderParam, resp1, err1)
	} else if direction == OrderSellSpotBuyPerp {
		//卖现货，买合约
		resp, err := s.agentHyper.MarketOrder(coin.OrderPerpId, orderParam.Size, nil)
		if !s.CheckOrder(coin, orderParam, resp, err) {
			return
		}
		resp1, err1 := s.agentHyper.MarketOrderSpot(coin.OrderSpotId, -orderParam.Size, nil)
		s.CheckOrder(coin, orderParam, resp1, err1)
	} else if direction == OrderMarketSpot {
		resp, err := s.agentHyper.MarketOrderSpot(coin.OrderSpotId, orderParam.Size, nil)
		s.CheckOrder(coin, orderParam, resp, err)
	} else if direction == OrderMarketPerp {
		resp, err := s.agentHyper.MarketOrder(coin.OrderPerpId, orderParam.Size, nil)
		s.CheckOrder(coin, orderParam, resp, err)
	}
}

func (s *Service) CheckOrder(coin *Coin, orderParam *OrderParam, orderResp *hyperliquid.OrderResponse, orderError error) bool {
	paramJson, _ := json.Marshal(orderParam)
	if orderError != nil {
		go s.LogErrorAndNotifyDev(fmt.Sprintf("[CheckOrderErr][%s] err: %s, order param: %s", coin.Name, orderError.Error(), string(paramJson)))
		return false
	}
	respJson, _ := json.Marshal(orderResp)
	if orderResp.Status != OrderStatusSuccess {
		go s.LogErrorAndNotifyDev(fmt.Sprintf("[CheckOrderFailed][%s] status: %s, order param: %s, resp: %s", coin.Name, orderResp.Status, string(paramJson), string(respJson)))
		return false
	}
	logrus.Infof("[OrderSuccess][%s] param: %s, resp: %s", coin.Name, string(paramJson), string(respJson))
	return true
}

func (s *Service) GetOrderParam(direction OrderAction, c *Coin) (*OrderParam, error) {
	var spotBook, perpBook *hyperliquid.L2BookSnapshot
	var spotBookErr, perpBookErr error
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		spotBookS, spotBookErrS := s.agentHyper.GetL2BookSnapshot(c.MarketSpotId)
		spotBook = spotBookS
		spotBookErr = spotBookErrS

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		perpBookS, perpBookErrS := s.agentHyper.GetL2BookSnapshot(c.MarketPerpId)
		perpBook = perpBookS
		perpBookErr = perpBookErrS
	}()
	wg.Wait()

	if spotBookErr != nil {
		logrus.Errorf("Error getting spot book: %v", spotBookErr)
		return nil, spotBookErr
	}
	if len(spotBook.Levels) < 2 {
		return nil, fmt.Errorf("[%s] spot order book lack bid/ask, length: %d, skip coin", c.Name, len(spotBook.Levels))
	}
	if perpBookErr != nil {
		logrus.Errorf("Error getting perp book: %v", perpBookErr)
		return nil, perpBookErr
	}
	if len(perpBook.Levels) < 2 {
		return nil, fmt.Errorf("[%s] perp order book lack bid/ask, length: %d, skip coin", c.Name, len(perpBook.Levels))
	}

	var orderSz float64

	if direction == OrderSellPerpBuySpot {
		//开仓
		//spot 卖1
		if len(spotBook.Levels[1]) < 1 {

			return nil, fmt.Errorf("[%s] spot order book [ASK] lack , skip coin", c.Name)
		}
		ask := spotBook.Levels[1][0]
		//perp 买1
		if len(perpBook.Levels[1]) < 1 {
			return nil, fmt.Errorf("[%s] perp order book [BID] lack , skip coin", c.Name)
		}
		bid := perpBook.Levels[0][0]

		logrus.Infof("[%s][orderBook] 现货卖1: %v, 合约买1: %v", c.Name, ask, bid)

		//choose order size: min(spot ask1, perp bid1) / 2
		basicSize := math.Min(ask.Sz*0.5, bid.Sz*0.5)
		orderPriceDiff := (bid.Px - ask.Px) / ask.Px * 100

		//判断剩余可开仓位,不能超过设置的最大占比
		maxPosition := s.GetSpotEntryValue() * (c.PositionMaxRatio / 100)
		freePositionUSD := maxPosition - c.PositionUSD

		spotAvailableUSD := s.spotAccount.AvailableUSDC

		orderUSD := math.Min(freePositionUSD, spotAvailableUSD)

		//开完单笔后perp最高2倍杠杆
		maxLeverageUSD := 2*s.perpAccount.AccountValue - s.perpAccount.TotalNtlPos
		orderUSD = math.Min(orderUSD, maxLeverageUSD)

		if orderUSD < 10 {
			return nil, fmt.Errorf("order usd size too small, skip")
		}

		freePositionSize := orderUSD / bid.Px

		orderSz = math.Min(freePositionSize, basicSize)

		//判断现货usdc还能开多少
		spotMaxSz := s.spotAccount.AvailableUSDC / ask.Px
		spotMaxSz = TruncateFloat(spotMaxSz, c.DecimalSpot.BigInt().Int64())
		orderSz = math.Min(orderSz, spotMaxSz)

		logrus.Infof("[%s][orderParam] raw size: %f, free size: %f, final size: %f, 差价: %f", c.Name, basicSize, freePositionSize, orderSz, orderPriceDiff)
	} else {
		//关仓 取现货买1，perp卖1
		//spot 买1
		if len(spotBook.Levels[0]) < 1 {
			return nil, fmt.Errorf("[%s] spot order book [Bid] lack , skip coin", c.Name)
		}
		bid := spotBook.Levels[0][0]

		//perp卖1
		if len(perpBook.Levels[1]) < 1 {
			return nil, fmt.Errorf("[%s] perp order book [Ask] lack , skip coin", c.Name)
		}
		ask := perpBook.Levels[1][0]

		logrus.Infof("[%s][orderBook] 现货买1: %v, 合约卖1: %v", c.Name, bid, ask)

		orderSz = math.Min(ask.Sz*0.5, bid.Sz*0.5)
		orderPriceDiff := (bid.Px - ask.Px) / ask.Px * 100

		//检查持有量，选min（持有量，orderSZ）
		orderSz = math.Min(orderSz, c.SpotBalance)

		logrus.Infof("[%s][orderParam] size: %f, 差价:  %f", c.Name, orderSz, orderPriceDiff)
	}

	//修正精度
	orderDecimal := decimal.Min(c.DecimalSpot, c.DecimalPerp)
	orderSz = TruncateFloat(orderSz, orderDecimal.BigInt().Int64())

	return &OrderParam{
		Coin: c.Name,
		Size: orderSz,
	}, nil
}

func (s *Service) ReBalanceCoinPosition(c *Coin, spotPrice float64, perpPrice float64) {
	var action OrderAction
	var tokenSize float64
	spotBalance := c.SpotBalance
	perpPositionSizeHold := c.PositionSize * -1 //做空的position size是负的
	//当前杠杆否允许继续开
	if c.OrderSetting.isAllowedOpenOrder {
		//允许开，先计算差多少，再检查账户usdc是否充足
		if spotBalance > perpPositionSizeHold {
			//现货多，继续做空
			action = OrderMarketPerp
			tokenSize = (spotBalance - perpPositionSizeHold) * -1 //卖空的是负数
			needUSDC := perpPrice * tokenSize
			if s.perpAccount.AvailableUSDC < needUSDC {
				//合约账户usdc不够，转为卖掉现货头寸来平衡
				action = OrderMarketSpot
			}
		} else {
			//现货少，买入现货
			action = OrderMarketSpot
			tokenSize = perpPositionSizeHold - spotBalance
			needUSDC := spotPrice * tokenSize
			if s.spotAccount.AvailableUSDC < needUSDC {
				//现货账户usdc不够，转为平掉一部分空单头寸来平衡
				action = OrderMarketPerp
			}
		}
	} else {
		//杠杆过高，不允许开。把多的现货卖掉 or 空单数量减少
		if spotBalance > perpPositionSizeHold {
			//现货多，卖掉现货
			tokenSize = perpPositionSizeHold - spotBalance
			action = OrderMarketSpot
		} else {
			//空单多，平掉空单
			tokenSize = spotBalance - perpPositionSizeHold
			action = OrderMarketPerp
		}
	}

	s.ExecOrder(action, c, &OrderParam{Coin: c.Name, Size: tokenSize})
}

func (s *Service) AccountTransferUSDC(amount float64, toPerp bool) (string, error) {
	nonce := GetNonce()
	transferUSDC := decimal.NewFromFloat(TruncateFloat(amount, 2))
	action := map[string]interface{}{
		"type":             "usdClassTransfer",
		"hyperliquidChain": "Mainnet",
		"signatureChainId": "0xa4b1",
		"amount":           transferUSDC.String(),
		"toPerp":           toPerp,
		"nonce":            nonce, // 使用uint64类型
	}
	types := []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "toPerp", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}

	v, r, s1, err := s.accountHyper.SignUserSignableAction(action, types, "HyperliquidTransaction:UsdClassTransfer")
	if err != nil {
		return "err", err
	}
	payload := map[string]interface{}{
		"action":    action,
		"signature": ToTypedSig(r, s1, v),
		"nonce":     nonce,
	}

	// 发送请求
	resp, err := req.R().SetBodyJsonMarshal(payload).Post("https://api.hyperliquid.xyz/exchange")
	if err != nil {
		return "err", err
	}

	var result RespDataExchange
	if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
		logrus.Errorf("fail to unmarshal AccountTransferUSDC response: %v", err)
		return "err", err
	}

	return result.Status, nil
}

/*
  - 检查期货账户的杠杆率，如果超过1.5，则从现货账户转钱至期货账户
    转移数量为：min(期货账户1.1倍杠杆还需要的usdc数量，现货账户现有的usdc数量) + 大于100U
  - 如果低于0.7，则从期货账户转钱至现货账户
    转移数量为：0.55 总资金 - 当前期货头寸 （同时也须大于100）
*/
func (s *Service) needReBalanceLeverage() (bool, bool, float64) {
	var toPerp bool
	if s.perpAccount.TotalNtlPos < 1 {
		//没有头寸，不用调(小于1)
		return false, toPerp, 0
	}
	if s.perpAccount.CrossAccountLeverage == 0 {
		//额外多判断一个杠杆率
		return false, toPerp, 0
	}
	if s.perpAccount.CrossAccountLeverage > 1.5 {
		toPerp = true
		targetLeverage := 1.1
		needMoreUSDC := (s.perpAccount.TotalNtlPos - targetLeverage*s.perpAccount.AccountValue) / targetLeverage
		needMoreUSDC = math.Max(needMoreUSDC, 15)
		//判断现货账户是否有这么多的余额
		if s.spotAccount.AvailableUSDC < needMoreUSDC {
			//资金不够不转移，等待差价回归 or 强制平仓
			return false, toPerp, 0
		}
		return true, toPerp, needMoreUSDC
	}
	if s.perpAccount.CrossAccountLeverage < 0.7 {
		toPerp = false
		needMoreUSDC := (s.perpAccount.AccountValue - s.perpAccount.TotalNtlPos) * 0.55
		needMoreUSDC = math.Max(needMoreUSDC, 15)
		//判断perp账户余额
		if s.perpAccount.AvailableUSDC < needMoreUSDC {
			return false, toPerp, 0
		}
		return true, toPerp, needMoreUSDC
	}
	return false, toPerp, 0
}
