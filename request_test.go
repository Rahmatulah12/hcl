package hcl

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

func TestNewWithContext(t *testing.T) {
	cb := &CircuitBreaker{
		failureCount:  1,
		resetTimeout:  100,
		halfOpenLimit: 10,
	}

	redisClient, _ := redismock.NewClientMock()

	cbRedis := &CircuitBreakerRedis{
		ctx:          context.TODO(),
		Client:       redisClient,
		FailureLimit: 5,
		ResetTimeout: 60,
	}

	hcl := &HCL{
		Context:   context.TODO(),
		Client:    &http.Client{},
		Cb:        cb,
		CbRedis:   cbRedis,
		EnableLog: true,
	}

	req := New(hcl)

	assert.NotNil(t, req)
	assert.Equal(t, hcl.Context, req.ctx)
	assert.Equal(t, hcl.Client, req.client)
	assert.NotNil(t, req.Cb)
	assert.NotNil(t, req.cbRedis)
	assert.NotNil(t, req.log)

	assert.Equal(t, cb.failureCount, req.Cb.failureCount)
	assert.Equal(t, cb.resetTimeout, req.Cb.resetTimeout)
	assert.Equal(t, cb.halfOpenLimit, req.Cb.halfOpenLimit)

	assert.Equal(t, cbRedis.FailureLimit, req.cbRedis.FailureLimit)
	assert.Equal(t, cbRedis.ResetTimeout, req.cbRedis.ResetTimeout)
}

func TestNewWithoutContext(t *testing.T) {
	hcl := &HCL{
		Context:   nil,
		Client:    &http.Client{},
		Cb:        nil,
		CbRedis:   nil,
		EnableLog: false,
	}

	req := New(hcl)

	assert.NotNil(t, req)
	assert.NotNil(t, req.ctx)
	assert.Equal(t, context.Background(), req.ctx)
	assert.Equal(t, hcl.Client, req.client)
	assert.Nil(t, req.Cb)
	assert.Nil(t, req.cbRedis)
	assert.Nil(t, req.log)
}

func TestRequestSetUrl(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		wantErr  bool
		errCount int
	}{
		{
			name:     "Valid URL",
			uri:      "https://example.com",
			wantErr:  false,
			errCount: 0,
		},
		{
			name:     "Empty URL",
			uri:      "",
			wantErr:  true,
			errCount: 1,
		},
		{
			name:     "Invalid URL",
			uri:      "://invalid-url",
			wantErr:  true,
			errCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Request{
				errs: make([]error, 0),
			}

			result := r.SetUrl(tt.uri)

			// Check if the result is chainable
			if result != r {
				t.Error("SetUrl should return the same Request instance")
			}

			// Check error count
			if len(r.errs) != tt.errCount {
				t.Errorf("Expected %d errors, got %d", tt.errCount, len(r.errs))
			}

			// For valid URLs, check if URL was properly set
			if !tt.wantErr && r.url == nil {
				t.Error("URL should not be nil for valid URI")
			}
		})
	}
}

func TestRequestSetQueryParam(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		key      string
		value    string
		wantErr  bool
		expected string
	}{
		{
			name:     "Valid query parameter",
			baseURL:  "https://example.com",
			key:      "test",
			value:    "value",
			wantErr:  false,
			expected: "test=value",
		},
		{
			name:     "Empty key",
			baseURL:  "https://example.com",
			key:      "",
			value:    "value",
			wantErr:  true,
			expected: "",
		},
		{
			name:     "Empty value",
			baseURL:  "https://example.com",
			key:      "test",
			value:    "",
			wantErr:  true,
			expected: "",
		},
		{
			name:     "Multiple parameters",
			baseURL:  "https://example.com?existing=param",
			key:      "test",
			value:    "value",
			wantErr:  false,
			expected: "existing=param&test=value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Request{
				errs: make([]error, 0),
			}

			// First set the base URL
			parsedURL, _ := url.Parse(tt.baseURL)
			r.url = parsedURL

			result := r.SetQueryParam(tt.key, tt.value)

			// Check if the result is chainable
			if result != r {
				t.Error("SetQueryParam should return the same Request instance")
			}

			// Check error presence
			if tt.wantErr && len(r.errs) == 0 {
				t.Error("Expected error, but got none")
			}

			if !tt.wantErr {
				if r.url.RawQuery != tt.expected {
					t.Errorf("Expected query string %q, got %q", tt.expected, r.url.RawQuery)
				}
			}
		})
	}
}

func TestRequestSetQueryParams(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		params   map[string]string
		expected string
		wantErr  bool
	}{
		{
			name:    "Valid multiple parameters",
			baseURL: "https://example.com",
			params: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expected: "key1=value1&key2=value2",
			wantErr:  false,
		},
		{
			name:     "Nil parameters",
			baseURL:  "https://example.com",
			params:   nil,
			expected: "",
			wantErr:  false,
		},
		{
			name:     "Empty parameters map",
			baseURL:  "https://example.com",
			params:   map[string]string{},
			expected: "",
			wantErr:  false,
		},
		{
			name:    "Parameters with empty key",
			baseURL: "https://example.com",
			params: map[string]string{
				"":     "value1",
				"key2": "value2",
			},
			expected: "key2=value2",
			wantErr:  true,
		},
		{
			name:    "Parameters with empty value",
			baseURL: "https://example.com",
			params: map[string]string{
				"key1": "",
				"key2": "value2",
			},
			expected: "key2=value2",
			wantErr:  true,
		},
		{
			name:    "Add to existing query parameters",
			baseURL: "https://example.com?existing=param",
			params: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expected: "existing=param&key1=value1&key2=value2",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Request{
				errs: make([]error, 0),
			}

			// Set the base URL
			parsedURL, _ := url.Parse(tt.baseURL)
			r.url = parsedURL

			// Call the method being tested
			result := r.SetQueryParams(tt.params)

			// Check if the method is chainable
			if result != r {
				t.Error("SetQueryParams should return the same Request instance")
			}

			// For nil params, verify no changes were made
			if tt.params == nil {
				if r.url.RawQuery != "" {
					t.Error("Query string should not be modified for nil params")
				}
				return
			}

			// Check error collection for invalid inputs
			if tt.wantErr && len(r.errs) == 0 {
				t.Error("Expected errors but got none")
			}
		})
	}
}

func TestRequestSetHeader(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		wantErr  bool
		expected map[string][]string
	}{
		{
			name:    "Valid header",
			key:     "Content-Type",
			value:   "application/json",
			wantErr: false,
			expected: map[string][]string{
				"Content-Type": {"application/json"},
			},
		},
		{
			name:     "Empty key",
			key:      "",
			value:    "application/json",
			wantErr:  true,
			expected: map[string][]string{},
		},
		{
			name:     "Empty value",
			key:      "Content-Type",
			value:    "",
			wantErr:  true,
			expected: map[string][]string{},
		},
		{
			name:    "Case insensitive header",
			key:     "content-type",
			value:   "application/json",
			wantErr: false,
			expected: map[string][]string{
				"Content-Type": {"application/json"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Request{
				header: make(http.Header),
				errs:   make([]error, 0),
			}

			result := r.SetHeader(tt.key, tt.value)

			// Check if the method is chainable
			if result != r {
				t.Error("SetHeader should return the same Request instance")
			}

			// Check error collection
			if tt.wantErr && len(r.errs) == 0 {
				t.Error("Expected error, but got none")
			}

			if !tt.wantErr && len(r.errs) > 0 {
				t.Errorf("Unexpected error: %v", r.errs)
			}

			// Check header values
			if !tt.wantErr {
				for k, v := range tt.expected {
					if !reflect.DeepEqual(r.header[k], v) {
						t.Errorf("Expected header %s to be %v, got %v", k, v, r.header[k])
					}
				}
			}
		})
	}
}

func TestRequestSetHeaders(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		wantErr  bool
		expected map[string][]string
	}{
		{
			name: "Valid headers",
			headers: map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/json",
				"X-Request-ID": "123",
			},
			wantErr: false,
			expected: map[string][]string{
				"Content-Type": {"application/json"},
				"Accept":       {"application/json"},
				"X-Request-Id": {"123"},
			},
		},
		{
			name:     "Nil headers",
			headers:  nil,
			wantErr:  true,
			expected: map[string][]string{},
		},
		{
			name: "Headers with empty key",
			headers: map[string]string{
				"":             "value",
				"Content-Type": "application/json",
			},
			wantErr: false,
			expected: map[string][]string{
				"Content-Type": {"application/json"},
			},
		},
		{
			name: "Headers with empty value",
			headers: map[string]string{
				"Accept":       "",
				"Content-Type": "application/json",
			},
			wantErr: false,
			expected: map[string][]string{
				"Content-Type": {"application/json"},
			},
		},
		{
			name:     "Empty headers map",
			headers:  map[string]string{},
			wantErr:  false,
			expected: map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Request{
				header: make(http.Header),
				errs:   make([]error, 0),
			}

			result := r.SetHeaders(tt.headers)

			// Check if the method is chainable
			if result != r {
				t.Error("SetHeaders should return the same Request instance")
			}

			// Check error collection
			if tt.wantErr && len(r.errs) == 0 {
				t.Error("Expected error, but got none")
			}

			// Check header values
			for k, v := range tt.expected {
				if !reflect.DeepEqual(r.header[k], v) {
					t.Errorf("Expected header %s to be %v, got %v", k, v, r.header[k])
				}
			}

			// Check that no unexpected headers were set
			for k := range r.header {
				if _, exists := tt.expected[k]; !exists {
					t.Errorf("Unexpected header found: %s", k)
				}
			}
		})
	}
}

func TestRequestSetJsonPayload(t *testing.T) {
	// Test struct for JSON payload
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name                string
		payload             interface{}
		wantErr             bool
		expectedJSON        string
		expectedContentType string
	}{
		{
			name: "Valid struct payload",
			payload: TestStruct{
				Name:  "test",
				Value: 123,
			},
			wantErr:             false,
			expectedJSON:        `{"name":"test","value":123}`,
			expectedContentType: "application/json",
		},
		{
			name:                "Valid map payload",
			payload:             map[string]interface{}{"key": "value"},
			wantErr:             false,
			expectedJSON:        `{"key":"value"}`,
			expectedContentType: "application/json",
		},
		{
			name:                "Valid byte slice payload",
			payload:             []byte(`{"test":"data"}`),
			wantErr:             false,
			expectedJSON:        `{"test":"data"}`,
			expectedContentType: "application/json",
		},
		{
			name:                "Nil payload",
			payload:             nil,
			wantErr:             true,
			expectedJSON:        "",
			expectedContentType: "application/json",
		},
		{
			name:                "Invalid JSON payload",
			payload:             make(chan int), // channels cannot be marshaled to JSON
			wantErr:             true,
			expectedJSON:        "",
			expectedContentType: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Request{
				header: make(map[string][]string),
				errs:   make([]error, 0),
			}

			// Call the method being tested
			result := r.SetJsonPayload(tt.payload)

			// Check if the method is chainable
			if result != r {
				t.Error("SetJsonPayload should return the same Request instance")
			}

			// Check error collection
			if tt.wantErr && len(r.errs) == 0 {
				t.Error("Expected error, but got none")
			}

			if !tt.wantErr && len(r.errs) > 0 {
				t.Errorf("Unexpected errors: %v", r.errs)
			}

			// Check Content-Type header
			contentType := r.header.Get("Content-Type")
			if contentType != tt.expectedContentType {
				t.Errorf("Expected Content-Type %s, got %s", tt.expectedContentType, contentType)
			}

			// Check body content
			if r.body != nil {
				bodyBytes, err := io.ReadAll(r.body)
				if err != nil {
					t.Fatalf("Failed to read body: %v", err)
				}

				if tt.expectedJSON != "" {
					// Compare JSON by parsing both expected and actual JSON
					var expected, actual interface{}
					if err := json.Unmarshal([]byte(tt.expectedJSON), &expected); err != nil {
						t.Fatalf("Failed to parse expected JSON: %v", err)
					}
					if err := json.Unmarshal(bodyBytes, &actual); err != nil {
						t.Fatalf("Failed to parse actual JSON: %v", err)
					}

					if !jsonEqual(expected, actual) {
						t.Errorf("Expected JSON %s, got %s", tt.expectedJSON, string(bodyBytes))
					}
				}

				// Reset the body for potential reuse
				r.body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		})
	}
}

// jsonEqual compares two interfaces that contain parsed JSON
func jsonEqual(a, b interface{}) bool {
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(aJSON, bJSON)
}

// Helper function to read the entire body and restore it
func readBody(body io.ReadCloser) ([]byte, io.ReadCloser, error) {
	if body == nil {
		return nil, nil, nil
	}

	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, body, err
	}

	// Restore the body for subsequent reads
	return bodyBytes, io.NopCloser(bytes.NewBuffer(bodyBytes)), nil
}

func TestRequestSetXMLPayload(t *testing.T) {
	// Test struct for XML payload
	type TestStruct struct {
		XMLName xml.Name `xml:"test"`
		Name    string   `xml:"name"`
		Value   int      `xml:"value"`
	}

	tests := []struct {
		name                string
		payload             interface{}
		wantErr             bool
		expectedXML         string
		expectedContentType string
	}{
		{
			name: "Valid struct payload",
			payload: TestStruct{
				Name:  "test",
				Value: 123,
			},
			wantErr:             false,
			expectedXML:         `<test><name>test</name><value>123</value></test>`,
			expectedContentType: "application/xml",
		},
		{
			name:                "Valid byte slice payload",
			payload:             []byte(`<test><name>test</name></test>`),
			wantErr:             false,
			expectedXML:         `<test><name>test</name></test>`,
			expectedContentType: "application/xml",
		},
		{
			name:                "Nil payload",
			payload:             nil,
			wantErr:             true,
			expectedXML:         "",
			expectedContentType: "application/xml",
		},
		{
			name:                "Invalid XML payload",
			payload:             make(chan int), // channels cannot be marshaled to XML
			wantErr:             true,
			expectedXML:         "",
			expectedContentType: "application/xml",
		},
		{
			name: "Complex struct payload",
			payload: struct {
				XMLName xml.Name `xml:"root"`
				Items   []struct {
					ID   int    `xml:"id"`
					Name string `xml:"name"`
				} `xml:"items"`
			}{
				Items: []struct {
					ID   int    `xml:"id"`
					Name string `xml:"name"`
				}{
					{ID: 1, Name: "Item 1"},
					{ID: 2, Name: "Item 2"},
				},
			},
			wantErr:             false,
			expectedXML:         `<root><items><id>1</id><name>Item 1</name></items><items><id>2</id><name>Item 2</name></items></root>`,
			expectedContentType: "application/xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Request{
				header: make(map[string][]string),
				errs:   make([]error, 0),
			}

			// Call the method being tested
			result := r.SetXMLPayload(tt.payload)

			// Check if the method is chainable
			if result != r {
				t.Error("SetXMLPayload should return the same Request instance")
			}

			// Check error collection
			if tt.wantErr && len(r.errs) == 0 {
				t.Error("Expected error, but got none")
			}

			if !tt.wantErr && len(r.errs) > 0 {
				t.Errorf("Unexpected errors: %v", r.errs)
			}

			// Check Content-Type header
			contentType := r.header.Get("Content-Type")
			if contentType != tt.expectedContentType {
				t.Errorf("Expected Content-Type %s, got %s", tt.expectedContentType, contentType)
			}

			// Check body content
			if r.body != nil {
				bodyBytes, err := io.ReadAll(r.body)
				if err != nil {
					t.Fatalf("Failed to read body: %v", err)
				}

				if tt.expectedXML != "" {
					// Compare XML by parsing both expected and actual XML
					expectedNormalized := normalizeXML(t, tt.expectedXML)
					actualNormalized := normalizeXML(t, string(bodyBytes))

					if expectedNormalized != actualNormalized {
						t.Errorf("Expected XML %s, got %s", expectedNormalized, actualNormalized)
					}
				}

				// Reset the body for potential reuse
				r.body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		})
	}
}

// normalizeXML helps compare XML strings by removing whitespace and normalizing format
func normalizeXML(t *testing.T, input string) string {
	if input == "" {
		return ""
	}

	// Parse the XML
	decoder := xml.NewDecoder(strings.NewReader(input))
	var doc interface{}
	err := decoder.Decode(&doc)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	// Marshal it back to normalized form
	normalized, err := xml.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal XML: %v", err)
	}

	return string(normalized)
}
