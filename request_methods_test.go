package hcl

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloseRequestAfterResponse(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var r *Request = nil
		result := r.CloseRequestAfterResponse()
		assert.Nil(t, result, "CloseRequestAfterResponse with nil receiver should return nil")
	})

	t.Run("set close request flag", func(t *testing.T) {
		r := &Request{}
		result := r.CloseRequestAfterResponse()

		assert.Equal(t, r, result, "CloseRequestAfterResponse should return the same Request instance")
		assert.True(t, r.closeRequest, "closeRequest flag should be set to true")
	})
}

func TestSetErrorHttpCodesCircuitBreaker(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var r *Request = nil
		result := r.SetErrorHttpCodesCircuitBreaker([]int{500, 502})
		assert.Nil(t, result, "SetErrorHttpCodesCircuitBreaker with nil receiver should return nil")
	})

	t.Run("empty codes", func(t *testing.T) {
		r := &Request{}
		result := r.SetErrorHttpCodesCircuitBreaker([]int{})

		assert.Equal(t, r, result, "SetErrorHttpCodesCircuitBreaker should return the same Request instance")
		assert.Nil(t, r.errHttpCodes, "errHttpCodes should remain nil")
	})

	t.Run("set error codes", func(t *testing.T) {
		r := &Request{}
		codes := []int{500, 502, 503}
		result := r.SetErrorHttpCodesCircuitBreaker(codes)

		assert.Equal(t, r, result, "SetErrorHttpCodesCircuitBreaker should return the same Request instance")
		assert.Equal(t, codes, r.errHttpCodes, "errHttpCodes should be set to the provided codes")
	})
}

func TestFetchErrors(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var r *Request = nil
		err := r.fetchErrors()

		assert.Error(t, err, "Error should be returned for nil receiver")
		assert.Contains(t, err.Error(), "failed to execute process", "Error message should match")
	})

	t.Run("no errors", func(t *testing.T) {
		r := &Request{
			errs: make([]error, 0),
		}
		err := r.fetchErrors()

		assert.NoError(t, err, "No error should be returned when errs is empty")
	})

	t.Run("with errors", func(t *testing.T) {
		expectedErr := errors.New("test error")
		r := &Request{
			errs: []error{expectedErr, errors.New("another error")},
		}
		err := r.fetchErrors()

		assert.Equal(t, expectedErr, err, "First error should be returned")
	})
}

// Mock implementation for sendRequest to use in tests
var sendRequest = func(r *Request, method RequestMethod) (*Response, error) {
	return nil, nil
}
