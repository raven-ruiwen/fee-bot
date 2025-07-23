package base

type BotConfig struct {
	Hyper HyperConfig `mapstructure:"hyper"`
}

type HyperConfig struct {
	AccountPk string `mapstructure:"account_pk"`
	AgentPk   string `mapstructure:"agent_pk"`
}
