package hcl

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var ErrRefuse = errors.New("request refused. the circuit is open")

type CircuitBreaker interface {
	Execute(func() (interface{}, error)) (interface{}, error)
	State() string
}

type Policy int

type State string

const (
	MaxFails Policy = iota
	MaxConsecutiveFails
)

const (
	open     State = "open"
	closed   State = "closed"
	halfOpen State = "half-open"
)

type ExtraOptions struct {
	Policy              Policy
	MaxFails            *uint64
	MaxConsecutiveFails *uint64
	OpenInterval        *time.Duration
}

type circuitBreaker struct {
	policy              Policy
	maxFails            uint64
	maxConsecutiveFails uint64
	openInterval        time.Duration

	fails       uint64
	state       State
	openChannel chan struct{}
	once        sync.Once
	mutex       sync.Mutex
}

func NewCircuitBreaker(opts ...ExtraOptions) CircuitBreaker {
	opt := ExtraOptions{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.MaxFails == nil {
		opt.MaxFails = ToPointer(uint64(5))
	}
	if opt.MaxConsecutiveFails == nil {
		opt.MaxConsecutiveFails = ToPointer(uint64(5))
	}
	if opt.OpenInterval == nil {
		opt.OpenInterval = ToPointer(5 * time.Second)
	}

	cb := &circuitBreaker{
		policy:              opt.Policy,
		maxFails:            *opt.MaxFails,
		maxConsecutiveFails: *opt.MaxConsecutiveFails,
		openInterval:        *opt.OpenInterval,
		state:               closed,
		openChannel:         make(chan struct{}, 1),
	}

	go cb.openWatcher()
	return cb
}

func (cb *circuitBreaker) Execute(req func() (any, error)) (any, error) {
	if cb.state == open {
		return nil, ErrRefuse
	}

	res, err := req()
	cb.handleRequestResult(err)
	return res, err
}

func (cb *circuitBreaker) State() string {
	return string(cb.state)
}

func (cb *circuitBreaker) handleRequestResult(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if err == nil {
		atomic.StoreUint64(&cb.fails, 0)
		cb.state = closed
		return
	}

	if cb.state == halfOpen {
		cb.transitionToOpen()
		return
	}

	atomic.AddUint64(&cb.fails, 1)
	if cb.failsExceededThreshold() {
		cb.transitionToOpen()
	}
}

func (cb *circuitBreaker) failsExceededThreshold() bool {
	switch cb.policy {
	case MaxConsecutiveFails:
		return atomic.LoadUint64(&cb.fails) >= cb.maxConsecutiveFails
	case MaxFails:
		return atomic.LoadUint64(&cb.fails) >= cb.maxFails
	}
	return false
}

func (cb *circuitBreaker) transitionToOpen() {
	cb.state = open
	select {
	case cb.openChannel <- struct{}{}:
	default:
	}
}

func (cb *circuitBreaker) openWatcher() {
	for range cb.openChannel {
		time.Sleep(cb.openInterval)
		cb.mutex.Lock()
		cb.state = halfOpen
		atomic.StoreUint64(&cb.fails, 0)
		cb.mutex.Unlock()
	}
}
