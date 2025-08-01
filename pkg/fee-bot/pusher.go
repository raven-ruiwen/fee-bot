package fee_bot

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
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

func pushData(spotAccount spotAccount, perpAccount perpAccount, tradeCoins map[string]*Coin) {
	if time.Now().Unix()-lastUpdateTime < 15 {
		return
	}

	exporterSpotTotalUSD.Set(spotAccount.TotalValueUSD)
	exporterPerpAccountValue.Set(perpAccount.AccountValue)
	exporterCrossAccountLeverage.Set(perpAccount.CrossAccountLeverage)
	exporterTotalValue.Set(spotAccount.TotalValueUSD + perpAccount.AccountValue)
	exporterInitValue.Set(initValue)
	exporterStartTime.Set(float64(time.Now().Unix() - startAt))

	for _, coin := range tradeCoins {
		exporterCoinValue.WithLabelValues(coin.Name).Set(coin.SpotValueUSD)
		exporterCoinLeverage.WithLabelValues(coin.Name).Set(coin.GetLeverage(perpAccount.AccountValue))
		exporterSpreadOpen.WithLabelValues(coin.Name).Set(coin.MarketData.OpenSpreadPercentage)
		exporterSpreadClose.WithLabelValues(coin.Name).Set(coin.MarketData.CloseSpreadPercentage)
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
		Collector(exporterStartTime)

	err := pusher.Grouping("job", jobName).Push()
	if err != nil {
		logrus.Errorf("Could not push to Pushgateway: %v", err)
	} else {
		logrus.Infof("Metrics pushed successfully")
	}
}
