package hcl

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

func TestNewWithNilHCL(t *testing.T) {
	req := New(nil)
	assert.NotNil(t, req)
	assert.Equal(t, context.Background(), req.ctx)
	assert.Nil(t, req.client)
	assert.Nil(t, req.Cb)
	assert.Nil(t, req.cbRedis)
}

func TestCloneCircuitBreakerNil(t *testing.T) {
	result := cloneCircuitBreaker(nil)
	assert.Nil(t, result)
}

func TestCloneCircuitBreakerRedisNil(t *testing.T) {
	result := cloneCircuitBreakerRedis(nil)
	assert.Nil(t, result)
}

func TestInitializeLogFalse(t *testing.T) {
	result := initializeLog(false)
	assert.Nil(t, result)
}

func TestTurnOffLog(t *testing.T) {
	r := &Request{
		isRepeatableLog: true,
		log:             NewLog(),
	}
	r.turnOffLog()
	assert.False(t, r.isRepeatableLog)
	assert.Nil(t, r.log)
}

func TestSetFormDataWithFile(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("test content")
	assert.NoError(t, err)
	tmpFile.Close()

	// Reopen for reading
	file, err := os.Open(tmpFile.Name())
	assert.NoError(t, err)
	defer file.Close()

	r := &Request{
		errs:   make([]error, 0),
		header: make(map[string][]string),
	}

	result := r.SetFormData(map[string]interface{}{
		"file": file,
	})

	assert.Equal(t, r, result)
	assert.Empty(t, r.errs)
}

func TestSendRequestNil(t *testing.T) {
	var r *Request = nil
	resp, err := r.sendRequest(GET)
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request cannot be nil")
}

func TestChooseExecutionStrategyWithRedisCircuitBreaker(t *testing.T) {
	// Setup Redis mock
	db, mock := redismock.NewClientMock()
	mock.ExpectGet("test_key").SetVal("5") // Above failure limit

	cbRedis := &CircuitBreakerRedis{
		Client:       db,
		FailureLimit: 3,
		ctx:          context.Background(),
	}

	r := &Request{
		cbRedis: cbRedis,
		cbKey:   "test_key",
	}

	resp, err := r.chooseExecutionStrategy()
	assert.Nil(t, resp)
	assert.Equal(t, errRefuse, err)
}

func TestExecuteRequestWithLogging(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	// Parse server URL
	serverURL, _ := url.Parse(server.URL)

	r := &Request{
		method:          "GET",
		url:             serverURL,
		header:          make(http.Header),
		ctx:             context.Background(),
		client:          server.Client(),
		log:             NewLog(),
		isRepeatableLog: false,
		errs:            make([]error, 0),
	}

	resp, err := r.executeRequest()
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestExecuteRequestWithCloseRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Parse server URL
	serverURL, _ := url.Parse(server.URL)

	r := &Request{
		method:       "GET",
		url:          serverURL,
		header:       make(http.Header),
		ctx:          context.Background(),
		client:       server.Client(),
		closeRequest: true,
		errs:         make([]error, 0),
	}

	resp, err := r.executeRequest()
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestExecuteRequestWithErrors(t *testing.T) {
	r := &Request{
		errs: []error{errors.New("test error")},
	}

	resp, err := r.executeRequest()
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Equal(t, "test error", err.Error())
}

func TestUpdateCircuitBreakerWithInMemory(t *testing.T) {
	cb := &CircuitBreaker{}
	r := &Request{
		Cb: cb,
	}

	// Test with success status
	r.updateCircuitBreaker(http.StatusOK)

	// Test with error status
	r.updateCircuitBreaker(http.StatusInternalServerError)
}

func TestUpdateCircuitBreakerWithRedis(t *testing.T) {
	// Setup Redis mock
	db, mock := redismock.NewClientMock()
	mock.ExpectDel("test_key").SetVal(1)  // For success
	mock.ExpectIncr("test_key").SetVal(1) // For failure
	mock.ExpectExpire("test_key", time.Duration(0)).SetVal(true)

	cbRedis := &CircuitBreakerRedis{
		Client: db,
		ctx:    context.Background(),
	}

	r := &Request{
		cbRedis: cbRedis,
		cbKey:   "test_key",
	}

	// Test with success status
	r.updateCircuitBreaker(http.StatusOK)

	// Test with error status
	r.updateCircuitBreaker(http.StatusInternalServerError)
}

func TestUpdateCircuitBreakerWithCustomErrorCodes(t *testing.T) {
	cb := &CircuitBreaker{}
	r := &Request{
		Cb:           cb,
		errHttpCodes: []int{400, 401},
	}

	// Test with custom error code
	r.updateCircuitBreaker(400)

	// Test with non-error code
	r.updateCircuitBreaker(200)
}

func TestSetQueryParamWithInvalidURL(t *testing.T) {
	r := &Request{
		errs: make([]error, 0),
		url: &url.URL{
			Scheme: "https",
			Host:   "example.com",
			Path:   "invalid path with spaces that will cause ParseRequestURI to fail",
		},
	}

	// This should trigger the ParseRequestURI error
	result := r.SetQueryParam("key", "value")
	assert.Equal(t, r, result)
}

func TestSetFormURLEncodedWithEmptyValues(t *testing.T) {
	r := &Request{
		errs:   make([]error, 0),
		header: make(map[string][]string),
	}

	result := r.SetFormURLEncoded(map[string]string{
		"":      "empty_key",
		"empty": "",
		"valid": "value",
	})

	assert.Equal(t, r, result)
	assert.Empty(t, r.errs)

	// Read body to verify only valid key-value pair is included
	bodyBytes, _ := io.ReadAll(r.body)
	bodyStr := string(bodyBytes)
	assert.Equal(t, "valid=value", bodyStr)
}

func TestExecuteRequestWithNilContext(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Parse server URL
	serverURL, _ := url.Parse(server.URL)

	r := &Request{
		method: "GET",
		url:    serverURL,
		header: make(http.Header),
		ctx:    nil, // Nil context
		client: server.Client(),
		errs:   make([]error, 0),
	}

	resp, err := r.executeRequest()
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, context.Background(), r.ctx) // Should be set to Background
}

func TestExecuteRequestWithBody(t *testing.T) {
	// Create a test server that echoes the request body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}))
	defer server.Close()

	// Parse server URL
	serverURL, _ := url.Parse(server.URL)

	testBody := "test request body"
	r := &Request{
		method: "POST",
		url:    serverURL,
		header: make(http.Header),
		ctx:    context.Background(),
		client: server.Client(),
		body:   io.NopCloser(strings.NewReader(testBody)),
		errs:   make([]error, 0),
	}

	resp, err := r.executeRequest()
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Read response body
	respBody, _ := io.ReadAll(resp.Body)
	assert.Equal(t, testBody, string(respBody))
}
