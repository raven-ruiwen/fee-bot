package fee_bot

import (
	"github.com/shopspring/decimal"
	"math"
)

type Coin struct {
	Name               string
	OrderSpotId        string
	OrderPerpId        string
	MarketSpotId       string
	MarketPerpId       string
	PositionMaxRatio   float64
	PositionUSD        float64
	PositionSize       float64
	PositionMarginUsed float64
	SpotBalance        float64
	SpotEntryNtl       float64
	SpotValueUSD       float64
	DecimalSpot        decimal.Decimal
	DecimalPerp        decimal.Decimal
	Leverage           int
	OrderSetting       orderSetting
}

func (c *Coin) SpotPositionEqualWithPerp(spotPrice float64) bool {
	sizeDiff := c.SpotBalance - math.Abs(c.PositionSize)
	usdDiff := math.Abs(sizeDiff * spotPrice)
	if c.SpotEntryNtl == 0 && c.PositionUSD == 0 {
		//都卖光了
		return true
	}
	if usdDiff < 10 {
		return true
	}
	//usd 差异达到10 u以上
	return false
}

func (c *Coin) ResetPositions() {
	c.PositionUSD = 0
	c.PositionSize = 0
	c.PositionMarginUsed = 0
	c.SpotBalance = 0
	c.SpotEntryNtl = 0
}

func (c *Coin) GetLeverage(accountValue float64) float64 {
	return c.PositionUSD / (accountValue * (c.PositionMaxRatio / 100))
}

func (c *Coin) SetOrderSettings(os orderSetting) {
	c.OrderSetting = os
}

func (c *Coin) GetAllowOpenPriceDiffRatio() float64 {
	return BasicOpenOrderPriceDiffRatio + c.OrderSetting.reBalanceRatio
}

func (c *Coin) GetAllowClosePriceDiff() float64 {
	return BasicCloseOrderPriceDiffRatio
}

type RespDataExchange struct {
	Status   string `json:"status"`
	Response struct {
		Type string `json:"type"`
	} `json:"response"`
}

type MarketData struct {
	SpotBidPrice          float64
	SpotBidSize           float64
	SpotAskPrice          float64
	SpotAskSize           float64
	PerpBidPrice          float64
	PerpBidSize           float64
	PerpAskPrice          float64
	PerpAskSize           float64
	OpenSpreadPercentage  float64
	CloseSpreadPercentage float64
}
