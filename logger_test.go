package hcl

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLog(t *testing.T) {
	log := NewLog()
	assert.NotNil(t, log)
	assert.Empty(t, log.l)
	assert.NotNil(t, log.maskedConfig)
	assert.Len(t, log.maskedConfig, 0)
}

func TestLogInitiate(t *testing.T) {
	log := NewLog()
	log.initiate()

	assert.Equal(t, INFO, log.l.Level)
	assert.NotEmpty(t, log.l.Time)
	assert.False(t, log.start.IsZero())

	log = nil
	log.initiate()
}

func TestLogSetRequest(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		log := NewLog()
		log.setRequest(nil)
		assert.Empty(t, log.l.Req)
	})

	t.Run("valid request without body", func(t *testing.T) {
		log := NewLog()
		req, _ := http.NewRequest("GET", "https://example.com/test?param=value", nil)
		req.Header.Add("Content-Type", "application/json")

		log.setRequest(req)

		assert.Equal(t, "example.com", log.l.Req.Host)
		assert.Equal(t, "/test", log.l.Req.Path)
		assert.Equal(t, "GET", log.l.Req.Method)
		assert.Equal(t, "value", log.l.Req.Query.Get("param"))
		assert.Equal(t, "application/json", log.l.Req.Header.Get("Content-Type"))
		assert.Empty(t, log.l.Req.Body)
	})

	t.Run("valid request with body", func(t *testing.T) {
		log := NewLog()
		body := strings.NewReader(`{"key": "value"}`)
		req, _ := http.NewRequest("POST", "https://example.com/test", body)

		log.setRequest(req)

		assert.Equal(t, `{"key":"value"}`, log.l.Req.Body)

		// Verify body can still be read
		bodyBytes, _ := io.ReadAll(req.Body)
		assert.Equal(t, `{"key": "value"}`, string(bodyBytes))
	})
}

func TestLogSetResponse(t *testing.T) {
	t.Run("nil response", func(t *testing.T) {
		log := NewLog()
		log.setResponse(nil)
		assert.Empty(t, log.l.Resp)
	})

	t.Run("valid response without body", func(t *testing.T) {
		log := NewLog()
		resp := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString("")),
		}

		log.setResponse(resp)

		assert.Equal(t, 200, log.l.Resp.StatusCode)
		assert.Empty(t, log.l.Resp.Body)
	})

	t.Run("valid response with body", func(t *testing.T) {
		log := NewLog()
		resp := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(`{"result": "success"}`)),
		}

		log.setResponse(resp)

		assert.Equal(t, 200, log.l.Resp.StatusCode)
		assert.Equal(t, `{"result":"success"}`, log.l.Resp.Body)

		// Verify body can still be read
		bodyBytes, _ := io.ReadAll(resp.Body)
		assert.Equal(t, `{"result": "success"}`, string(bodyBytes))
	})
}

func TestLogSetError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		log := NewLog()
		log.setError(nil)
		assert.Empty(t, log.l.Error)
		assert.NotEqual(t, ERROR, log.l.Level)
	})

	t.Run("valid error", func(t *testing.T) {
		log := NewLog()
		err := errors.New("test error")
		log.setError(err)

		assert.Equal(t, "test error", log.l.Error)
		assert.Equal(t, ERROR, log.l.Level)
	})
}

func TestLogMapperLog(t *testing.T) {
	t.Run("empty json", func(t *testing.T) {
		log := NewLog()
		result := log.mapperLog("")
		assert.Empty(t, result)
	})

	t.Run("invalid json", func(t *testing.T) {
		log := NewLog()
		result := log.mapperLog("invalid json")
		assert.Empty(t, result)
	})

	t.Run("valid json", func(t *testing.T) {
		log := NewLog()
		jsonStr := `{"time":"2023-01-01T00:00:00Z","level":"info","latency":"10 ms"}`
		result := log.mapperLog(jsonStr)

		assert.Contains(t, result, `"time":"2023-01-01T00:00:00Z"`)
		assert.Contains(t, result, `"level":"info"`)
		assert.Contains(t, result, `"latency":"10 ms"`)
	})
}

// Helper function to capture stdout for testing writeLog
func captureOutput(f func()) string {
	// Save original stdout
	oldStdout := os.Stdout

	// Create a pipe to capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute the function that writes to stdout
	f()

	// Close the writer and restore original stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Return captured output
	return buf.String()
}

func TestLogWriteLog(t *testing.T) {
	t.Run("basic log without masking", func(t *testing.T) {
		// Setup test
		log := NewLog()
		log.start = time.Now().Add(-100 * time.Millisecond) // Set start time 100ms in the past
		log.l.Time = log.start.Format(time.RFC3339)
		log.l.Level = INFO

		// Capture output
		output := captureOutput(func() {
			log.writeLog()
		})

		// Verify output contains expected fields
		assert.Contains(t, output, `"level":"info"`)
		assert.Contains(t, output, `"latency":"`)
		assert.Contains(t, output, " ms\"")
	})

	t.Run("log with masking", func(t *testing.T) {
		// Setup test
		log := NewLog()
		log.start = time.Now().Add(-100 * time.Millisecond)
		log.l.Time = log.start.Format(time.RFC3339)
		log.l.Level = INFO

		// Add sensitive data
		log.l.Error = "password: secret123"

		// Add mask config
		log.maskedConfig = append(log.maskedConfig, &MaskConfig{
			Field:    "Error",
			MaskType: FullMask,
		})

		// Capture output
		output := captureOutput(func() {
			log.writeLog()
		})

		// Verify output is masked
		assert.Contains(t, output, `"level":"info"`)
		assert.NotContains(t, output, "secret123")
	})

	t.Run("nil log", func(t *testing.T) {
		// We can't actually call a method on a nil pointer
		// This test is just to verify it doesn't panic
		// The function should return early when lg is nil

		// Create a deferred function to recover from panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("writeLog panicked with nil receiver: %v", r)
			}
		}()

		var log *Log = nil
		log.writeLog() // This should return early without panic
	})
}
