package hcl

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetQueryParam(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var r *Request = nil
		result := r.SetQueryParam("key", "value")
		assert.Nil(t, result, "SetQueryParam with nil receiver should return nil")
	})

	t.Run("empty key", func(t *testing.T) {
		r := &Request{
			errs: make([]error, 0),
		}
		result := r.SetQueryParam("", "value")
		
		assert.Equal(t, r, result, "SetQueryParam should return the same Request instance")
		assert.Len(t, r.errs, 1, "Should have one error")
		assert.Equal(t, msgFailedKeyVal, r.errs[0].Error(), "Error message should match")
	})

	t.Run("empty value", func(t *testing.T) {
		r := &Request{
			errs: make([]error, 0),
		}
		result := r.SetQueryParam("key", "")
		
		assert.Equal(t, r, result, "SetQueryParam should return the same Request instance")
		assert.Len(t, r.errs, 1, "Should have one error")
		assert.Equal(t, msgFailedKeyVal, r.errs[0].Error(), "Error message should match")
	})

	t.Run("nil URL", func(t *testing.T) {
		r := &Request{
			errs: make([]error, 0),
			url:  nil,
		}
		result := r.SetQueryParam("key", "value")
		
		assert.Equal(t, r, result, "SetQueryParam should return the same Request instance")
		assert.Len(t, r.errs, 1, "Should have one error")
		assert.Contains(t, r.errs[0].Error(), "url is not set", "Error should mention URL not set")
	})

	t.Run("URL missing scheme", func(t *testing.T) {
		r := &Request{
			errs: make([]error, 0),
			url: &url.URL{
				Host: "example.com",
			},
		}
		result := r.SetQueryParam("key", "value")
		
		assert.Equal(t, r, result, "SetQueryParam should return the same Request instance")
		assert.Len(t, r.errs, 1, "Should have one error")
		assert.Contains(t, r.errs[0].Error(), "missing scheme", "Error should mention missing scheme")
	})

	t.Run("URL missing host", func(t *testing.T) {
		r := &Request{
			errs: make([]error, 0),
			url: &url.URL{
				Scheme: "https",
			},
		}
		result := r.SetQueryParam("key", "value")
		
		assert.Equal(t, r, result, "SetQueryParam should return the same Request instance")
		assert.Len(t, r.errs, 1, "Should have one error")
		assert.Contains(t, r.errs[0].Error(), "missing scheme or host", "Error should mention missing host")
	})

	t.Run("valid URL and parameters", func(t *testing.T) {
		r := &Request{
			errs: make([]error, 0),
			url: &url.URL{
				Scheme: "https",
				Host:   "example.com",
				Path:   "/test",
			},
		}
		result := r.SetQueryParam("key", "value")
		
		assert.Equal(t, r, result, "SetQueryParam should return the same Request instance")
		assert.Empty(t, r.errs, "Should have no errors")
		assert.Equal(t, "key=value", r.url.RawQuery, "Query string should be set correctly")
	})

	t.Run("add to existing query parameters", func(t *testing.T) {
		r := &Request{
			errs: make([]error, 0),
			url: &url.URL{
				Scheme:   "https",
				Host:     "example.com",
				Path:     "/test",
				RawQuery: "existing=param",
			},
		}
		result := r.SetQueryParam("key", "value")
		
		assert.Equal(t, r, result, "SetQueryParam should return the same Request instance")
		assert.Empty(t, r.errs, "Should have no errors")
		assert.Contains(t, r.url.RawQuery, "existing=param", "Should preserve existing parameters")
		assert.Contains(t, r.url.RawQuery, "key=value", "Should add new parameter")
	})

	t.Run("update existing query parameter", func(t *testing.T) {
		r := &Request{
			errs: make([]error, 0),
			url: &url.URL{
				Scheme:   "https",
				Host:     "example.com",
				Path:     "/test",
				RawQuery: "key=oldvalue",
			},
		}
		result := r.SetQueryParam("key", "newvalue")
		
		assert.Equal(t, r, result, "SetQueryParam should return the same Request instance")
		assert.Empty(t, r.errs, "Should have no errors")
		assert.Equal(t, "key=newvalue", r.url.RawQuery, "Should update existing parameter")
	})

	t.Run("parameter with special characters", func(t *testing.T) {
		r := &Request{
			errs: make([]error, 0),
			url: &url.URL{
				Scheme: "https",
				Host:   "example.com",
				Path:   "/test",
			},
		}
		result := r.SetQueryParam("key", "value with spaces")
		
		assert.Equal(t, r, result, "SetQueryParam should return the same Request instance")
		assert.Empty(t, r.errs, "Should have no errors")
		assert.Contains(t, r.url.RawQuery, "key=value+with+spaces", "Should URL encode the value")
	})
}