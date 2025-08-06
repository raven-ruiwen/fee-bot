package fee_bot

import (
	"encoding/json"
	"github.com/imroc/req/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"time"
)

var pushGatewayUrl string
var initValue float64
var startAt int64
var jobName = "hyperliquid_multi_token_test"
var lastUpdateTime int64

var (
	exporterSpotTotalUSD = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "hyper_multi_token_spot_total_usd",
		Help: "",
	})
	exporterPerpAccountValue = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "hyper_multi_token_perp_account_value",
		Help: "",
	})
	exporterCrossAccountLeverage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "hyper_multi_token_across_account_leverage",
		Help: "",
	})
	exporterTotalValue = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "hyper_multi_token_value",
		Help: "",
	})
	exporterInitValue = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "hyper_multi_token_init_value",
		Help: "",
	})

	exporterStartTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "hyper_multi_token_up_times",
		Help: "",
	})
	exporterFunding7D = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "hyper_multi_token_funding_7d",
		Help: "",
	})

	exporterCoinValue = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "hyper_multi_token_coin_value",
		Help: "",
	}, []string{"coin"})
	exporterCoinLeverage = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "hyper_multi_token_coin_leverage",
		Help: "",
	}, []string{"coin"})
	exporterSpreadOpen = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "hyper_multi_token_spread_open",
		Help: "",
	}, []string{"coin"})
	exporterSpreadClose = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "hyper_multi_token_spread_close",
		Help: "",
	}, []string{"coin"})
	exporterFundingRate = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "hyper_multi_token_funding_rate",
		Help: "",
	}, []string{"coin"})
)

func init() {
	prometheus.MustRegister(
		exporterSpotTotalUSD,
		exporterPerpAccountValue,
		exporterCrossAccountLeverage,
		exporterTotalValue,
		exporterInitValue,
		exporterCoinValue,
		exporterCoinLeverage,
		exporterSpreadOpen,
		exporterSpreadClose,
		exporterStartTime,
		exporterFundingRate,
		exporterFunding7D,
	)
}

func SetPushGateway(url string) {
	pushGatewayUrl = url
}
func SetInitValue(_initValue float64) {
	initValue = _initValue
}
func SetStartTime(_startAt int64) {
	startAt = _startAt
}

func pushData(accountAddress string, spotAccount spotAccount, perpAccount perpAccount, tradeCoins map[string]*Coin) {
	if pushGatewayUrl == "" {
		return
	}
	if time.Now().Unix()-lastUpdateTime < 15 {
		return
	}

	exporterSpotTotalUSD.Set(spotAccount.TotalValueUSD)
	exporterPerpAccountValue.Set(perpAccount.AccountValue)
	exporterCrossAccountLeverage.Set(perpAccount.CrossAccountLeverage)
	exporterTotalValue.Set(spotAccount.TotalValueUSD + perpAccount.AccountValue)
	exporterInitValue.Set(initValue)
	exporterStartTime.Set(float64(time.Now().Unix() - startAt))
	funding7D, err := GetFunding7D(accountAddress)
	if err == nil {
		exporterFunding7D.Set(funding7D)
	}

	for _, coin := range tradeCoins {
		exporterCoinValue.WithLabelValues(coin.Name).Set(coin.SpotValueUSD)
		exporterCoinLeverage.WithLabelValues(coin.Name).Set(coin.GetLeverage(perpAccount.AccountValue))
		exporterSpreadOpen.WithLabelValues(coin.Name).Set(coin.MarketData.OpenSpreadPercentage)
		exporterSpreadClose.WithLabelValues(coin.Name).Set(coin.MarketData.CloseSpreadPercentage)
		if fundingRate, err := GetFundingRate(coin.Name); err == nil {
			exporterFundingRate.WithLabelValues(coin.Name).Set(fundingRate)
		}
	}

	execPrometheusPush()
	lastUpdateTime = time.Now().Unix()
}

func execPrometheusPush() {
	pusher := push.New(pushGatewayUrl, jobName).
		Collector(exporterSpotTotalUSD).
		Collector(exporterPerpAccountValue).
		Collector(exporterCrossAccountLeverage).
		Collector(exporterTotalValue).
		Collector(exporterInitValue).
		Collector(exporterCoinValue).
		Collector(exporterCoinLeverage).
		Collector(exporterSpreadOpen).
		Collector(exporterSpreadClose).
		Collector(exporterStartTime).
		Collector(exporterFundingRate).
		Collector(exporterFunding7D)

	err := pusher.Grouping("job", jobName).Push()
	if err != nil {
		logrus.Errorf("Could not push to Pushgateway: %v", err)
	} else {
		logrus.Infof("Metrics pushed successfully")
	}
}

func GetFundingRate(coin string) (float64, error) {
	now := time.Now()
	hourStart := time.Date(
		now.Year(), now.Month(), now.Day(),
		now.Hour(), 0, 0, 0, now.Location(),
	)

	// 转为毫秒时间戳
	ms := hourStart.UnixNano() / int64(time.Millisecond)

	payload := map[string]interface{}{
		"type":      "fundingHistory",
		"coin":      coin,
		"startTime": ms,
	}

	// 发送请求
	resp, err := req.R().SetBodyJsonMarshal(payload).Post("https://api.hyperliquid.xyz/info")
	if err != nil {
		return 0.0, err
	}
	var result []FundingRate
	if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
		logrus.Errorf("fail to unmarshal FundingRate response: %v", err)
		return 0, err
	}
	if len(result) == 0 {
		return 0, nil
	}
	fundingRate, _ := decimal.NewFromString(result[0].FundingRate)
	return fundingRate.InexactFloat64() * 100, nil
}

func GetFunding7D(address string) (float64, error) {
	before7D := time.Now().Add(-7 * 24 * time.Hour)

	// 转为毫秒时间戳
	ms := before7D.UnixNano() / int64(time.Millisecond)

	payload := map[string]interface{}{
		"type":      "userFunding",
		"user":      address,
		"startTime": ms,
	}

	// 发送请求
	resp, err := req.R().SetBodyJsonMarshal(payload).Post("https://api.hyperliquid.xyz/info")
	if err != nil {
		return 0.0, err
	}
	var result []Funding
	if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
		logrus.Errorf("fail to unmarshal FundingRate response: %v", err)
		return 0, err
	}
	var total float64
	for _, funding := range result {
		f, _ := decimal.NewFromString(funding.Delta.Usdc)
		total += f.InexactFloat64()
	}
	return total, nil
}

func GetFee7D(address string) (float64, error) {
	before7D := time.Now().Add(-7 * 24 * time.Hour)

	// 转为毫秒时间戳
	ms := before7D.UnixNano() / int64(time.Millisecond)

	payload := map[string]interface{}{
		"type":      "userFees",
		"user":      address,
		"startTime": ms,
	}

	// 发送请求
	resp, err := req.R().SetBodyJsonMarshal(payload).Post("https://api.hyperliquid.xyz/info")
	if err != nil {
		return 0.0, err
	}
	var result []Funding
	if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
		logrus.Errorf("fail to unmarshal FundingRate response: %v", err)
		return 0, err
	}
	var total float64
	for _, funding := range result {
		f, _ := decimal.NewFromString(funding.Delta.Usdc)
		total += f.InexactFloat64()
	}
	return total, nil
}
