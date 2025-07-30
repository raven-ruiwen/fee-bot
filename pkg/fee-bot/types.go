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
	DecimalSpot        decimal.Decimal
	DecimalPerp        decimal.Decimal
	Leverage           int
}

func (c *Coin) SpotPositionEqualWithPerp(spotPrice float64) bool {
	//usdDiff := math.Abs(c.SpotEntryNtl - c.PositionUSD)
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

type RespDataExchange struct {
	Status   string `json:"status"`
	Response struct {
		Type string `json:"type"`
	} `json:"response"`
}
