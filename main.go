package main

import (
	"fee-bot/pkg/base"
	fee_bot "fee-bot/pkg/fee-bot"
	"fee-bot/pkg/notify"
	"fee-bot/pkg/redis"
	"github.com/Logarithm-Labs/go-hyperliquid/hyperliquid"
	"github.com/sirupsen/logrus"
)

var coins []*fee_bot.Coin

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
	config := base.GetBotConfig()

	notifyClient := notify.NewService()
	for _, nc := range config.Notifies {
		notifyClient.AddNotify(nc.Platform, nc.Token, nc.Channel)
	}

	debugNotifyClient := notify.NewService()
	for _, nc := range config.DebugNotifies {
		debugNotifyClient.AddNotify(nc.Platform, nc.Token, nc.Channel)
	}

	accountAddress := base.PkToAddress(config.Hyper.AccountPk)
	agentAddress := base.PkToAddress(config.Hyper.AgentPk)

	for _, token := range config.Hyper.Tokens {
		coins = append(coins, &fee_bot.Coin{
			Name:             token.Name,
			OrderSpotId:      token.OrderSpotId,
			OrderPerpId:      token.OrderPerpId,
			MarketSpotId:     token.MarketSpotId,
			MarketPerpId:     token.MarketPerpId,
			Leverage:         token.Leverage,
			PositionMaxRatio: token.PositionMaxRatio,
		})
	}

	//coins = append(coins, &fee_bot.Coin{Name: "HYPE", MarketSpotId: "@107", MarketPerpId: "HYPE", PositionMaxRatio: 30, OrderSpotId: "HYPE", OrderPerpId: "HYPE", Leverage: 5})
	//coins = append(coins, &fee_bot.Coin{Name: "FARTCOIN", MarketSpotId: "@162", MarketPerpId: "FARTCOIN", PositionMaxRatio: 50, OrderSpotId: "UFART", OrderPerpId: "FARTCOIN", Leverage: 5})
	//coins = append(coins, &fee_bot.Coin{Name: "PURR", MarketSpotId: "PURR/USDC", MarketPerpId: "PURR", PositionMaxRatio: 20, OrderSpotId: "PURR", OrderPerpId: "PURR", Leverage: 3})
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

	_redisClient := redis.NewRedis(&config.Redis)

	fee_bot.SetBasicOpenOrderPriceDiffRatio(config.Hyper.BasicOpenOrderPriceDiffRatio)
	fee_bot.SetBasicCloseOrderPriceDiffRatio(config.Hyper.BasicCloseOrderPriceDiffRatio)
	fee_bot.SetPushGateway(config.PushGateway)
	fee_bot.SetInitValue(config.Hyper.InitValue)
	fee_bot.SetStartTime(config.Hyper.StartAt)

	service := fee_bot.NewService(accountAddress, agentHyper, accountHyper, coins, 3, notifyClient, debugNotifyClient, _redisClient)
	service.Init()
	service.Run()
}
