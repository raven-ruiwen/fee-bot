package base

type BotConfig struct {
	Hyper    HyperConfig `mapstructure:"hyper"`
	Notifies Notifies    `mapstructure:"notifies"`
}

type HyperConfig struct {
	AccountPk string `mapstructure:"account_pk"`
	AgentPk   string `mapstructure:"agent_pk"`
}

type Notifies []Notify

type Notify struct {
	Platform string
	Token    string `mapstructure:"token"`
	Channel  string `mapstructure:"channel"`
}
