package otredis

import (
	"fmt"

	"github.com/DoNewsCode/std/pkg/async"
	"github.com/DoNewsCode/std/pkg/contract"
	"github.com/DoNewsCode/std/pkg/di"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
)

// RedisConfigurationInterceptor intercepts the redis.UniversalOptions before
// creating the client so you can make amendment to it. Useful because some
// configuration can not be mapped to a text representation. For example, you
// cannot add OnConnect callback in a configuration file, but you can add it
// here.
type RedisConfigurationInterceptor func(name string, opts *redis.UniversalOptions)

// RedisIn is the injection parameter for ProvideRedis.
type RedisIn struct {
	di.In

	Logger      log.Logger
	Conf        contract.ConfigAccessor
	Interceptor RedisConfigurationInterceptor `optional:"true"`
	Tracer      opentracing.Tracer            `optional:"true"`
}

// RedisOut is the result of ProvideRedis.
type RedisOut struct {
	di.Out
	di.Module

	Maker   Maker
	Factory Factory
	Client  redis.UniversalClient
}

// ProvideConfig exports the default redis configuration
func (r RedisOut) ProvideConfig() []contract.ExportedConfig {
	return []contract.ExportedConfig{
		{
			Name: "redis",
			Data: map[string]interface{}{
				"redis": map[string]map[string]interface{}{
					"default": {
						"addrs":              []string{"127.0.0.1:6379"},
						"DB":                 0,
						"username":           "",
						"password":           "",
						"sentinelPassword":   "",
						"maxRetries":         0,
						"minRetryBackoff":    0,
						"maxRetryBackoff":    0,
						"dialTimeout":        0,
						"readTimeout":        0,
						"writeTimeout":       0,
						"poolSize":           0,
						"minIdleConns":       0,
						"maxConnAge":         0,
						"poolTimeout":        0,
						"idleTimeout":        0,
						"idleCheckFrequency": 0,
						"maxRedirects":       0,
						"readOnly":           false,
						"routeByLatency":     false,
						"routeRandomly":      false,
						"masterName":         "",
					},
				},
			},
			Comment: "The configuration of redis clients",
		},
	}
}

// ProvideRedis creates Factory and redis.UniversalClient. It is a valid
// dependency for package core.
func ProvideRedis(p RedisIn) (RedisOut, func()) {
	var err error
	var dbConfs map[string]redis.UniversalOptions
	err = p.Conf.Unmarshal("redis", &dbConfs)
	if err != nil {
		level.Warn(p.Logger).Log("err", err)
	}
	factory := async.NewFactory(func(name string) (async.Pair, error) {
		var (
			ok   bool
			conf redis.UniversalOptions
		)
		if conf, ok = dbConfs[name]; !ok {
			return async.Pair{}, fmt.Errorf("redis configuration %s not valid", name)
		}
		if p.Interceptor != nil {
			p.Interceptor(name, &conf)
		}
		client := redis.NewUniversalClient(&conf)
		if p.Tracer != nil {
			client.AddHook(
				hook{
					addrs:    conf.Addrs,
					database: conf.DB,
					tracer:   p.Tracer,
				},
			)
		}
		return async.Pair{
			Conn: client,
			Closer: func() {
				_ = client.Close()
			},
		}, nil
	})
	redisFactory := Factory{factory}
	redisOut := RedisOut{
		Maker:   redisFactory,
		Factory: redisFactory,
		Client:  nil,
	}
	defaultRedisClient, _ := redisFactory.Make("default")
	redisOut.Client = defaultRedisClient
	return redisOut, redisFactory.Close
}

// Maker is models Factory
type Maker interface {
	Make(name string) (redis.UniversalClient, error)
}

// Factory is a *async.Factory that creates redis.UniversalClient using a
// specific configuration entry.
type Factory struct {
	*async.Factory
}

// Make creates redis.UniversalClient using a specific configuration entry.
func (r Factory) Make(name string) (redis.UniversalClient, error) {
	client, err := r.Factory.Make(name)
	if err != nil {
		return nil, err
	}
	return client.(redis.UniversalClient), nil
}
