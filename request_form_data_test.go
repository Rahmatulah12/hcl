package hcl

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetFormData(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var r *Request = nil
		result := r.SetFormData(map[string]interface{}{"key": "value"})
		assert.Nil(t, result, "SetFormData with nil receiver should return nil")
	})

	t.Run("nil data", func(t *testing.T) {
		r := &Request{
			errs:   make([]error, 0),
			header: make(map[string][]string),
		}
		result := r.SetFormData(nil)

		assert.Equal(t, r, result, "SetFormData should return the same Request instance")
		assert.Len(t, r.errs, 1, "Should have one error")
		assert.Contains(t, r.errs[0].Error(), "form data cannot be nil", "Error message should match")
	})

	t.Run("empty data", func(t *testing.T) {
		r := &Request{
			errs:   make([]error, 0),
			header: make(map[string][]string),
		}
		result := r.SetFormData(map[string]interface{}{})

		assert.Equal(t, r, result, "SetFormData should return the same Request instance")
		assert.Empty(t, r.errs, "Should have no errors")
		assert.NotNil(t, r.body, "Body should not be nil")
		assert.Contains(t, r.header.Get("Content-Type"), "multipart/form-data", "Content-Type should be set correctly")
	})

	t.Run("string value", func(t *testing.T) {
		r := &Request{
			errs:   make([]error, 0),
			header: make(map[string][]string),
		}
		result := r.SetFormData(map[string]interface{}{
			"name": "John Doe",
		})

		assert.Equal(t, r, result, "SetFormData should return the same Request instance")
		assert.Empty(t, r.errs, "Should have no errors")

		// Read body content
		bodyBytes, _ := io.ReadAll(r.body)
		bodyStr := string(bodyBytes)

		assert.Contains(t, bodyStr, "name=\"name\"", "Form field name should be in body")
		assert.Contains(t, bodyStr, "John Doe", "Form field value should be in body")
		assert.Contains(t, r.header.Get("Content-Type"), "multipart/form-data", "Content-Type should be set correctly")
	})

	t.Run("integer value", func(t *testing.T) {
		r := &Request{
			errs:   make([]error, 0),
			header: make(map[string][]string),
		}
		result := r.SetFormData(map[string]interface{}{
			"age": int64(30),
		})

		assert.Equal(t, r, result, "SetFormData should return the same Request instance")
		assert.Empty(t, r.errs, "Should have no errors")

		// Read body content
		bodyBytes, _ := io.ReadAll(r.body)
		bodyStr := string(bodyBytes)

		assert.Contains(t, bodyStr, "name=\"age\"", "Form field name should be in body")
		assert.Contains(t, bodyStr, "30", "Form field value should be in body")
	})

	t.Run("float value", func(t *testing.T) {
		r := &Request{
			errs:   make([]error, 0),
			header: make(map[string][]string),
		}
		result := r.SetFormData(map[string]interface{}{
			"price": float64(99.99),
		})

		assert.Equal(t, r, result, "SetFormData should return the same Request instance")
		assert.Empty(t, r.errs, "Should have no errors")

		// Read body content
		bodyBytes, _ := io.ReadAll(r.body)
		bodyStr := string(bodyBytes)

		assert.Contains(t, bodyStr, "name=\"price\"", "Form field name should be in body")
		assert.Contains(t, bodyStr, "99.99", "Form field value should be in body")
	})

	t.Run("boolean value", func(t *testing.T) {
		r := &Request{
			errs:   make([]error, 0),
			header: make(map[string][]string),
		}
		result := r.SetFormData(map[string]interface{}{
			"active": true,
		})

		assert.Equal(t, r, result, "SetFormData should return the same Request instance")
		assert.Empty(t, r.errs, "Should have no errors")

		// Read body content
		bodyBytes, _ := io.ReadAll(r.body)
		bodyStr := string(bodyBytes)

		assert.Contains(t, bodyStr, "name=\"active\"", "Form field name should be in body")
		assert.Contains(t, bodyStr, "true", "Form field value should be in body")
	})

	t.Run("reader value", func(t *testing.T) {
		r := &Request{
			errs:   make([]error, 0),
			header: make(map[string][]string),
		}

		fileContent := "test file content"
		reader := strings.NewReader(fileContent)

		result := r.SetFormData(map[string]interface{}{
			"file": reader,
		})

		assert.Equal(t, r, result, "SetFormData should return the same Request instance")
		assert.Empty(t, r.errs, "Should have no errors")

		// Read body content
		bodyBytes, _ := io.ReadAll(r.body)
		bodyStr := string(bodyBytes)

		assert.Contains(t, bodyStr, "name=\"file\"", "Form field name should be in body")
		assert.Contains(t, bodyStr, "filename=\"file\"", "Default filename should be in body")
		assert.Contains(t, bodyStr, fileContent, "File content should be in body")
	})

	t.Run("unsupported type", func(t *testing.T) {
		r := &Request{
			errs:   make([]error, 0),
			header: make(map[string][]string),
		}

		// Channel is an unsupported type
		ch := make(chan int)

		result := r.SetFormData(map[string]interface{}{
			"channel": ch,
		})

		assert.Equal(t, r, result, "SetFormData should return the same Request instance")
		assert.Len(t, r.errs, 1, "Should have one error")
		assert.Contains(t, r.errs[0].Error(), "Unsupported type for key: channel", "Error message should match")
	})

	t.Run("empty key", func(t *testing.T) {
		r := &Request{
			errs:   make([]error, 0),
			header: make(map[string][]string),
		}
		result := r.SetFormData(map[string]interface{}{
			"":     "value",
			"name": "John",
		})

		assert.Equal(t, r, result, "SetFormData should return the same Request instance")
		assert.Empty(t, r.errs, "Should have no errors")

		// Read body content
		bodyBytes, _ := io.ReadAll(r.body)
		bodyStr := string(bodyBytes)

		assert.NotContains(t, bodyStr, "name=\"\"", "Empty key should be skipped")
		assert.Contains(t, bodyStr, "name=\"name\"", "Valid key should be included")
	})

	t.Run("nil value", func(t *testing.T) {
		r := &Request{
			errs:   make([]error, 0),
			header: make(map[string][]string),
		}
		result := r.SetFormData(map[string]interface{}{
			"nullField": nil,
			"name":      "John",
		})

		assert.Equal(t, r, result, "SetFormData should return the same Request instance")
		assert.Empty(t, r.errs, "Should have no errors")

		// Read body content
		bodyBytes, _ := io.ReadAll(r.body)
		bodyStr := string(bodyBytes)

		assert.NotContains(t, bodyStr, "name=\"nullField\"", "Nil value should be skipped")
		assert.Contains(t, bodyStr, "name=\"name\"", "Valid field should be included")
	})

	t.Run("multiple fields", func(t *testing.T) {
		r := &Request{
			errs:   make([]error, 0),
			header: make(map[string][]string),
		}
		result := r.SetFormData(map[string]interface{}{
			"name":   "John Doe",
			"age":    int64(30),
			"active": true,
		})

		assert.Equal(t, r, result, "SetFormData should return the same Request instance")
		assert.Empty(t, r.errs, "Should have no errors")

		// Read body content
		bodyBytes, _ := io.ReadAll(r.body)
		bodyStr := string(bodyBytes)

		assert.Contains(t, bodyStr, "name=\"name\"", "Name field should be in body")
		assert.Contains(t, bodyStr, "John Doe", "Name value should be in body")
		assert.Contains(t, bodyStr, "name=\"age\"", "Age field should be in body")
		assert.Contains(t, bodyStr, "30", "Age value should be in body")
		assert.Contains(t, bodyStr, "name=\"active\"", "Active field should be in body")
		assert.Contains(t, bodyStr, "true", "Active value should be in body")
	})
}
