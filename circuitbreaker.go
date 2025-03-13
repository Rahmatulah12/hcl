package hcl

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var errRefuse = errors.New("request refused. the circuit is open")

type CircuitBreaker struct {
	Client       *redis.Client
	FailureLimit int
	ResetTimeout time.Duration
	ctx          context.Context
}

func NewCircuitBreaker(conf *CircuitBreaker) *CircuitBreaker {
	return &CircuitBreaker{
		Client:       conf.Client,
		FailureLimit: conf.FailureLimit,
		ResetTimeout: conf.ResetTimeout,
		ctx:          context.Background(),
	}
}

func (c *CircuitBreaker) recordFailure(key string) {
	failures, err := c.Client.Incr(c.ctx, key).Result()

	if err != nil {
		panic(err.Error())
	}

	if failures == 1 {
		c.Client.Expire(c.ctx, key, c.ResetTimeout) // set timeout at first failure
	}
}

func (c *CircuitBreaker) reset(key string) {
	c.Client.Del(c.ctx, key)
}

func (c *CircuitBreaker) allowRequest(key string) error {
	failures, err := c.Client.Get(c.ctx, key).Int()
	if err != nil && err != redis.Nil {
		return err
	}

	if failures >= c.FailureLimit {
		return errRefuse
	}
	return nil
}
