package hcl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetUrl(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var r *Request = nil
		result := r.SetUrl("https://example.com")
		assert.Nil(t, result, "SetUrl with nil receiver should return nil")
	})

	t.Run("empty URL", func(t *testing.T) {
		r := &Request{
			errs: make([]error, 0),
		}
		result := r.SetUrl("")

		assert.Equal(t, r, result, "SetUrl should return the same Request instance")
		assert.Len(t, r.errs, 1, "Should have one error")
		assert.Equal(t, msgEmptyUrl, r.errs[0].Error(), "Error message should match")
		assert.Nil(t, r.url, "URL should be nil")
	})

	t.Run("invalid URL format", func(t *testing.T) {
		r := &Request{
			errs: make([]error, 0),
		}
		result := r.SetUrl("://invalid-url")

		assert.Equal(t, r, result, "SetUrl should return the same Request instance")
		assert.Len(t, r.errs, 1, "Should have one error")
		assert.Contains(t, r.errs[0].Error(), "invalid URL", "Error should mention invalid URL")
		assert.Nil(t, r.url, "URL should be nil")
	})

	t.Run("valid URL", func(t *testing.T) {
		r := &Request{
			errs: make([]error, 0),
		}
		result := r.SetUrl("https://example.com/path?query=value")

		assert.Equal(t, r, result, "SetUrl should return the same Request instance")
		assert.Empty(t, r.errs, "Should have no errors")
		assert.NotNil(t, r.url, "URL should not be nil")
		assert.Equal(t, "https", r.url.Scheme, "Scheme should be https")
		assert.Equal(t, "example.com", r.url.Host, "Host should be example.com")
		assert.Equal(t, "/path", r.url.Path, "Path should be /path")
		assert.Equal(t, "query=value", r.url.RawQuery, "Query should be query=value")
	})

	t.Run("URL with special characters", func(t *testing.T) {
		r := &Request{
			errs: make([]error, 0),
		}
		result := r.SetUrl("https://example.com/path with spaces?query=value with spaces")

		assert.Equal(t, r, result, "SetUrl should return the same Request instance")
		assert.Empty(t, r.errs, "Should have no errors")
		assert.NotNil(t, r.url, "URL should not be nil")
		assert.Equal(t, "https", r.url.Scheme, "Scheme should be https")
		assert.Equal(t, "example.com", r.url.Host, "Host should be example.com")
		assert.Equal(t, "/path with spaces", r.url.Path, "Path should be preserved")
	})

	t.Run("URL with port", func(t *testing.T) {
		r := &Request{
			errs: make([]error, 0),
		}
		result := r.SetUrl("http://localhost:8080/api")

		assert.Equal(t, r, result, "SetUrl should return the same Request instance")
		assert.Empty(t, r.errs, "Should have no errors")
		assert.NotNil(t, r.url, "URL should not be nil")
		assert.Equal(t, "http", r.url.Scheme, "Scheme should be http")
		assert.Equal(t, "localhost:8080", r.url.Host, "Host should include port")
		assert.Equal(t, "/api", r.url.Path, "Path should be /api")
	})
}
