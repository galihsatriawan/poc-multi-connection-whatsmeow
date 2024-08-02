package main

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

// ProvideRedisMutex create new instance of redis distributed lock
func ProvideRedisMutex(redis *redis.Client) *redsync.Redsync {
	return redsync.New(goredis.NewPool(redis))
}

// RedisNewClient create new instance of redis
func RedisNewClient(host, port, password string, database int) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       database,
	})

	pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		panic(err)
	}
	fmt.Println(pong, err)

	return client
}

type DistributedMutex interface {
	Lock() error
	Unlock() (bool, error)
}
type DistMutexConfig struct {
	options []redsync.Option
}
type Option func(*DistMutexConfig)

func WithDelayFunc(delayFunc func(tries int) time.Duration) Option {
	return func(dmc *DistMutexConfig) {
		dmc.options = append(dmc.options, redsync.WithRetryDelayFunc(delayFunc))
	}
}
func WithRetries(number int) Option {
	return func(dmc *DistMutexConfig) {
		dmc.options = append(dmc.options, redsync.WithTries(number))
	}
}
func WithExpiry(expiry time.Duration) Option {
	return func(dmc *DistMutexConfig) {
		dmc.options = append(dmc.options, redsync.WithExpiry(expiry))
	}
}
func defaultDistMutexConfig() *DistMutexConfig {
	return &DistMutexConfig{
		options: []redsync.Option{},
	}
}

var (
	redSync     *redsync.Redsync
	redisClient *redis.Client
)

func getRedisClient() *redis.Client {
	if redisClient == nil {
		redisClient = RedisNewClient("localhost", "6379", "", 0)
	}
	return redisClient
}
func getRedsync() *redsync.Redsync {
	if redSync == nil {
		redSync = ProvideRedisMutex(getRedisClient())
	}
	return redSync
}
func NewDistMutex(key string, opts ...Option) DistributedMutex {
	defaultConfig := defaultDistMutexConfig()
	for _, opt := range opts {
		opt(defaultConfig)
	}
	key = fmt.Sprintf("%s.%s", key, "stage")
	// log.Printf("create new distributed mutex with key : %s\n", key)
	res := getRedsync().NewMutex(key, defaultConfig.options...)
	return res
}
