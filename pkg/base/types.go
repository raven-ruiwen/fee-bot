package base

type BotConfig struct {
	PushGateway   string      `mapstructure:"push_gateway"`
	Hyper         HyperConfig `mapstructure:"hyper"`
	Notifies      Notifies    `mapstructure:"notifies"`
	DebugNotifies Notifies    `mapstructure:"debug_notifies"`
	Redis         RedisConfig `mapstructure:"redis"`
}

type HyperConfig struct {
	StartAt                       int64   `mapstructure:"start_at"`
	InitValue                     float64 `mapstructure:"init_value"`
	AccountPk                     string  `mapstructure:"account_pk"`
	AgentPk                       string  `mapstructure:"agent_pk"`
	BasicOpenOrderPriceDiffRatio  float64 `mapstructure:"basic_open_order_price_diff_ratio"`
	BasicCloseOrderPriceDiffRatio float64 `mapstructure:"basic_close_order_price_diff_ratio"`
	Tokens                        []Token `mapstructure:"tokens"`
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

type RedisConfig struct {
	Addr          string `mapstructure:"addr"`
	Pass          string `mapstructure:"pass"`
	DB            int    `mapstructure:"db"`
	PoolSize      int    `mapstructure:"pool_size"`
	TlsSkipVerify bool   `mapstructure:"tls_skip_verify"`
}
