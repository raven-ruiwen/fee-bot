package redis

import (
	"crypto/tls"
	"fee-bot/pkg/base"
	redisV8 "github.com/go-redis/redis/v8"
)

func NewRedis(redisConfig *base.RedisConfig) *redisV8.Client {
	//redis init
	config := &redisV8.Options{
		Addr:     redisConfig.Addr,
		Password: redisConfig.Pass, // no password set
		DB:       redisConfig.DB,   // use default DB
		PoolSize: redisConfig.PoolSize,
	}
	if redisConfig.TlsSkipVerify == true {
		config.TLSConfig = &tls.Config{InsecureSkipVerify: redisConfig.TlsSkipVerify}
	}
	_redisClient := redisV8.NewClient(config)

	return _redisClient
}
