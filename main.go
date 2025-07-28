package main

import (
	"fee-bot/pkg/base"
	fee_bot "fee-bot/pkg/fee-bot"
	"github.com/Logarithm-Labs/go-hyperliquid/hyperliquid"
)

var coins []*fee_bot.Coin

func main() {
	config := base.GetBotConfig()

	accountAddress := base.PkToAddress(config.Hyper.AccountPk)
	agentAddress := base.PkToAddress(config.Hyper.AgentPk)

	coins = append(coins, &fee_bot.Coin{Name: "HYPE", MarketSpotId: "@107", MarketPerpId: "HYPE", PositionMaxRatio: 100, OrderSpotId: "HYPE", OrderPerpId: "HYPE", Leverage: 5})
	//coins = append(coins, &fee_bot.Coin{Name: "PUMP", MarketSpotId: "@188", MarketPerpId: "PUMP", PositionMaxRatio: 50, OrderSpotId: "UPUMP", OrderPerpId: "PUMP", Leverage: 5})

	accountHyper := hyperliquid.NewHyperliquid(&hyperliquid.HyperliquidClientConfig{
		IsMainnet:      true,
		AccountAddress: accountAddress,         // Main address of the Hyperliquid account that you want to use
		PrivateKey:     config.Hyper.AccountPk, // Private key of the account or API private key from Hyperliquid
	})

	agentHyper := hyperliquid.NewHyperliquid(&hyperliquid.HyperliquidClientConfig{
		IsMainnet:      true,
		AccountAddress: agentAddress,         // Main address of the Hyperliquid account that you want to use
		PrivateKey:     config.Hyper.AgentPk, // Private key of the account or API private key from Hyperliquid
	})

	service := fee_bot.NewService(accountAddress, agentHyper, accountHyper, coins, 7)
	service.Init()
	service.Run()
}
