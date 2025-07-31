package base

type BotConfig struct {
	Hyper    HyperConfig `mapstructure:"hyper"`
	Notifies Notifies    `mapstructure:"notifies"`
}

type HyperConfig struct {
	AccountPk string  `mapstructure:"account_pk"`
	AgentPk   string  `mapstructure:"agent_pk"`
	Tokens    []Token `mapstructure:"tokens"`
}

type Notifies []Notify

type Notify struct {
	Platform string
	Token    string `mapstructure:"token"`
	Channel  string `mapstructure:"channel"`
}

type Token struct {
	Name             string  `mapstructure:"name"`
	OrderSpotId      string  `mapstructure:"order_spot_id"`
	OrderPerpId      string  `mapstructure:"order_perp_id"`
	MarketSpotId     string  `mapstructure:"market_spot_id"`
	MarketPerpId     string  `mapstructure:"market_perp_id"`
	PositionMaxRatio float64 `mapstructure:"position_max_ratio"`
	Leverage         int     `mapstructure:"leverage"`
}
