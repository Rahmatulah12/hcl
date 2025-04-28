package hcl

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var errRefuse = errors.New("request refused. the circuit breaker is open")

type CircuitBreakerRedis struct {
	Client       *redis.Client
	FailureLimit int
	ResetTimeout time.Duration
	ctx          context.Context
}

func NewCircuitBreakerRedis(conf *CircuitBreakerRedis) *CircuitBreakerRedis {
	return &CircuitBreakerRedis{
		Client:       conf.Client,
		FailureLimit: conf.FailureLimit,
		ResetTimeout: conf.ResetTimeout,
		ctx:          context.Background(),
	}
}

func (c *CircuitBreakerRedis) recordFailure(key string) {
	failures, err := c.Client.Incr(c.ctx, key).Result()

	if err != nil {
		panic(err.Error())
	}

	if failures == 1 {
		c.Client.Expire(c.ctx, key, c.ResetTimeout) // set timeout at first failure
	}
}

func (c *CircuitBreakerRedis) reset(key string) {
	c.Client.Del(c.ctx, key)
}

func (c *CircuitBreakerRedis) allowRequest(key string) error {
	failures, err := c.Client.Get(c.ctx, key).Int()
	if err != nil && err != redis.Nil {
		return err
	}

	if failures >= c.FailureLimit {
		return errRefuse
	}
	return nil
}
