package fee_bot

// const BasicOpenOrderPriceDiffRatio = 0.3
// const BasicCloseOrderPriceDiffRatio = -0.2
const LeverageForceLiquidation = 2.5
const SingleLiquidationValueUSD = 3000
const MaxOpenOrderUSD = 2000

type OrderAction string

const (
	OrderSellPerpBuySpot OrderAction = "OrderSellPerpBuySpot"
	OrderSellSpotBuyPerp OrderAction = "OrderSellSpotBuyPerp"
	OrderNoAction        OrderAction = "OrderNoAction"
	OrderMarketSpot      OrderAction = "OrderMarketSpot"
	OrderMarketPerp      OrderAction = "OrderMarketPerp"
)

const OrderStatusSuccess = "ok"
