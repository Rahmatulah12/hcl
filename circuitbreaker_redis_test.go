package hcl

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestNewCircuitBreakerRedis(t *testing.T) {
	// Setup
	db, _ := redismock.NewClientMock()
	conf := &CircuitBreakerRedis{
		Client:       db,
		FailureLimit: 3,
		ResetTimeout: 10 * time.Second,
	}

	// Execute
	cb := NewCircuitBreakerRedis(conf)

	// Assert
	assert.Equal(t, db, cb.Client)
	assert.Equal(t, 3, cb.FailureLimit)
	assert.Equal(t, 10*time.Second, cb.ResetTimeout)
	assert.NotNil(t, cb.ctx)
}

func TestCircuitBreakerRedisRecordFailure(t *testing.T) {
	// Setup
	db, mock := redismock.NewClientMock()
	key := "test_service"

	cb := &CircuitBreakerRedis{
		Client:       db,
		FailureLimit: 3,
		ResetTimeout: 10 * time.Second,
		ctx:          context.Background(),
	}

	// Test case 1: First failure
	mock.ExpectIncr(key).SetVal(1)
	mock.ExpectExpire(key, 10*time.Second).SetVal(true)

	// Execute
	cb.recordFailure(key)

	// Assert
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	// Test case 2: Subsequent failure
	mock.ExpectIncr(key).SetVal(2)

	// Execute
	cb.recordFailure(key)

	// Assert
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCircuitBreakerRedisReset(t *testing.T) {
	// Setup
	db, mock := redismock.NewClientMock()
	key := "test_service"

	cb := &CircuitBreakerRedis{
		Client:       db,
		FailureLimit: 3,
		ResetTimeout: 10 * time.Second,
		ctx:          context.Background(),
	}

	mock.ExpectDel(key).SetVal(1)

	// Execute
	cb.reset(key)

	// Assert
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCircuitBreakerRedisAllowRequest(t *testing.T) {
	// Setup
	db, mock := redismock.NewClientMock()
	key := "test_service"

	cb := &CircuitBreakerRedis{
		Client:       db,
		FailureLimit: 3,
		ResetTimeout: 10 * time.Second,
		ctx:          context.Background(),
	}

	// Test case 1: Key doesn't exist (no failures)
	mock.ExpectGet(key).SetErr(redis.Nil)

	// Execute
	cb.allowRequest(key)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	// Test case 2: Failures below limit
	mock.ExpectGet(key).SetVal("2")

	// Execute
	cb.allowRequest(key)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	// Test case 3: Failures at limit
	mock.ExpectGet(key).SetVal("3")

	// Execute
	cb.allowRequest(key)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	// Test case 4: Failures above limit
	mock.ExpectGet(key).SetVal("4")

	// Execute
	cb.allowRequest(key)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	// Test case 5: Redis error
	redisErr := redis.NewUniversalClient(&redis.UniversalOptions{}).Ping(context.Background()).Err()
	mock.ExpectGet(key).SetErr(redisErr)

	// Execute
	cb.allowRequest(key)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
