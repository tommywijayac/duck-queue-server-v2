package databases

import (
	"context"
	"time"

	"github.com/beego/beego/v2/client/cache"
	_ "github.com/beego/beego/v2/client/cache/memcache"
	_ "github.com/beego/beego/v2/client/cache/redis"
	"github.com/beego/beego/v2/server/web"
	"github.com/redis/go-redis/v9"
)

var Redis cache.Cache // for general use

func InitCache() error {
	adapter, err := web.AppConfig.String("redis::cache_adapter")
	if err != nil {
		adapter = "memory"
	}

	config, err := web.AppConfig.String("redis::cache_config")
	if err != nil {
		config = `{"interval":60}`
	}

	Redis, err = cache.NewCache(adapter, config)
	if err != nil {
		return err
	}

	return nil
}

var RedisClient *redis.Client // for pipeline and advanced redis operations

func InitRedisClient() error {
	host, _ := web.AppConfig.String("redis::conn")
	if host == "" {
		host = "127.0.0.1:6379"
	}

	password, _ := web.AppConfig.String("redis::password")
	dbNumStr, _ := web.AppConfig.String("redis::dbNum")

	dbNum := 0
	if dbNumStr != "" {
		// Convert string to int
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     host,
		Password: password,
		DB:       dbNum,
	})

	// Test connection
	ctx := context.Background()
	_, err := RedisClient.Ping(ctx).Result()
	return err
}

type RedisPipeline struct {
	pipe redis.Pipeliner
	ctx  context.Context
}

func NewPipeline() *RedisPipeline {
	return &RedisPipeline{
		pipe: RedisClient.Pipeline(),
		ctx:  context.Background(),
	}
}

func (rp *RedisPipeline) Set(key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return rp.pipe.Set(rp.ctx, key, value, expiration)
}

func (rp *RedisPipeline) Get(key string) *redis.StringCmd {
	return rp.pipe.Get(rp.ctx, key)
}

func (rp *RedisPipeline) Del(keys ...string) *redis.IntCmd {
	return rp.pipe.Del(rp.ctx, keys...)
}

func (rp *RedisPipeline) Exec() ([]redis.Cmder, error) {
	return rp.pipe.Exec(rp.ctx)
}

func (rp *RedisPipeline) Discard() {
	rp.pipe.Discard()
}
