package fee_bot

const BasicOpenOrderPriceDiffRatio = 0.3
const BasicCloseOrderPriceDiffRatio = -0.1
const LeverageForceLiquidation = 2.5
const SingleLiquidationValueUSD = 3000

type OrderAction int

const (
	OrderSellPerpBuySpot OrderAction = iota
	OrderSellSpotBuyPerp
	OrderNoAction
	OrderMarketSpot
	OrderMarketPerp
)
