package hcl

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type RequestMethod string

const (
	GET    RequestMethod = http.MethodGet
	POST   RequestMethod = http.MethodPost
	PATCH  RequestMethod = http.MethodPatch
	PUT    RequestMethod = http.MethodPut
	DELETE RequestMethod = http.MethodDelete

	msgFailedKeyVal = "something went wrong, please set key and value request"
	msgFailedBody   = "something went wrong, please set body request"
	msgEmptyUrl     = "something went wrong, please set uri request"
)

const (
	contentType         = "Content-Type"
	contentTypeJSON     = "application/json"
	contentTypeXML      = "application/xml"
	contentTypeFormData = "application/x-www-form-urlencoded"
)

type Request struct {
	method  string
	url     *url.URL
	header  http.Header
	body    io.ReadCloser
	ctx     context.Context
	client  *http.Client
	Cb      *CircuitBreaker
	cbRedis *CircuitBreakerRedis
	cbKey   string
	log     *Log
	errs    []error
}

type HCL struct {
	Context   context.Context
	Client    *http.Client
	Cb        *CircuitBreaker
	CbRedis   *CircuitBreakerRedis
	EnableLog bool
}

func New(hcl *HCL) *Request {
	ctx := hcl.Context

	if ctx == nil {
		ctx = context.Background()
	}

	return &Request{
		ctx:     ctx,
		client:  hcl.Client,
		Cb:      cloneCircuitBreaker(hcl.Cb),
		cbRedis: cloneCircuitBreakerRedis(hcl.CbRedis),
		log:     initializeLog(hcl.EnableLog),
		header:  make(http.Header),
	}
}

// Helper functions for New
func cloneCircuitBreaker(cb *CircuitBreaker) *CircuitBreaker {
	if cb == nil {
		return nil
	}
	return &CircuitBreaker{
		failureCount:  cb.failureCount,
		resetTimeout:  cb.resetTimeout,
		halfOpenLimit: cb.halfOpenLimit,
	}
}

func cloneCircuitBreakerRedis(cbRedis *CircuitBreakerRedis) *CircuitBreakerRedis {
	if cbRedis == nil {
		return nil
	}
	return &CircuitBreakerRedis{
		ctx:          cbRedis.ctx,
		Client:       cbRedis.Client,
		FailureLimit: cbRedis.FailureLimit,
		ResetTimeout: cbRedis.ResetTimeout,
	}
}

func initializeLog(enableLog bool) *Log {
	if enableLog {
		return NewLog()
	}
	return nil
}

func (r *Request) SetUrl(uri string) *Request {
	if uri == "" {
		r.errs = append(r.errs, fmt.Errorf(msgEmptyUrl))
		return r
	}

	parsedUrl, err := url.Parse(uri)
	if err != nil {
		r.errs = append(r.errs, err)
		return r
	}

	r.url = parsedUrl

	return r
}

func (r *Request) SetQueryParam(key, val string) *Request {
	if key == "" || val == "" {
		r.errs = append(r.errs, fmt.Errorf(msgFailedKeyVal))
		return r
	}

	q := r.url.Query()
	q.Set(key, val)
	r.url.RawQuery = q.Encode()

	return r
}

func (r *Request) SetQueryParams(val map[string]string) *Request {
	if val == nil {
		return r
	}

	for k, v := range val {
		r.SetQueryParam(k, v)
	}

	return r
}

func (r *Request) SetHeader(key, val string) *Request {
	if key == "" || val == "" {
		r.errs = append(r.errs, fmt.Errorf(msgFailedKeyVal))
		return r
	}

	r.header.Set(key, val)

	return r
}

func (r *Request) SetHeaders(val map[string]string) *Request {
	if val == nil {
		r.errs = append(r.errs, fmt.Errorf("something went wrong, please set header request"))
		return r
	}

	for k, v := range val {
		if k == "" || v == "" {
			continue
		}
		r.SetHeader(k, v)
	}

	return r
}

func (r *Request) SetJsonPayload(body interface{}) *Request {
	if body == nil {
		r.errs = append(r.errs, fmt.Errorf(msgFailedBody))
		return r
	}

	var b []byte
	var err error

	switch v := body.(type) {
	case []byte:
		b = v
	default:
		b, err = json.Marshal(v)
		if err != nil {
			r.errs = append(r.errs, fmt.Errorf("failed to marshal json: %w", err))
		}
	}

	r.header.Set(contentType, contentTypeJSON)
	r.body = io.NopCloser(bytes.NewBuffer(b))

	return r
}

func (r *Request) SetXMLPayload(body interface{}) *Request {
	if body == nil {
		r.errs = append(r.errs, fmt.Errorf(msgFailedBody))
		return r
	}

	var b []byte
	var err error

	switch v := body.(type) {
	case []byte:
		b = v
	default:
		b, err = xml.Marshal(body)
		if err != nil {
			r.errs = append(r.errs, err)
		}
	}

	r.header.Set(contentType, contentTypeXML)
	r.body = io.NopCloser(bytes.NewBuffer(b))

	return r
}

func (r *Request) SetFormData(data map[string]interface{}) *Request {
	if data == nil {
		r.errs = append(r.errs, fmt.Errorf("form data cannot be nil"))
		return r
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for key, val := range data {
		if key == "" || val == nil {
			continue // Skip key/value kosong
		}

		switch v := val.(type) {
		case string:
			writer.WriteField(key, v)
		case int, int8, int16, int32, int64:
			writer.WriteField(key, strconv.FormatInt(v.(int64), 10))
		case float32, float64:
			writer.WriteField(key, strconv.FormatFloat(v.(float64), 'f', -1, 64))
		case bool:
			writer.WriteField(key, strconv.FormatBool(v))
		case io.Reader:
			part, err := writer.CreateFormFile(key, "file") // Default filename "file"
			if err != nil {
				r.errs = append(r.errs, err)
				return r
			}
			io.Copy(part, v)
		case *os.File:
			part, err := writer.CreateFormFile(key, v.Name())
			if err != nil {
				r.errs = append(r.errs, err)
			}
			io.Copy(part, v)
		default:
			msg := fmt.Sprintf("Unsupported type for key: %s\n", key)
			r.errs = append(r.errs, fmt.Errorf(msg))
		}
	}

	writer.Close()

	r.header.Set(contentType, writer.FormDataContentType())
	r.body = io.NopCloser(&body)

	return r
}

func (r *Request) SetFormURLEncoded(data map[string]string) *Request {
	if data == nil {
		r.errs = append(r.errs, fmt.Errorf(msgFailedBody))
		return r
	}

	formData := url.Values{}

	for key, val := range data {
		if key == "" || val == "" {
			continue
		}
		formData.Set(key, val)
	}

	// Encode form data
	encodedForm := formData.Encode()

	r.body = io.NopCloser(bytes.NewBufferString(encodedForm))
	r.header.Set(contentType, "application/x-www-form-urlencoded")

	return r
}

func (r *Request) SetCircuitBreakerKey(key string) *Request {
	if key == "" {
		r.errs = append(r.errs, fmt.Errorf("key cannot be empty"))
		return r
	}

	r.cbKey = key
	return r
}

func (r *Request) SetMaskedField(conf ...*MaskConfig) *Request {
	r.log.maskedConfig = append(r.log.maskedConfig, conf...)
	return r
}

func (r *Request) SetMaskedFields(configs []*MaskConfig) *Request {
	return r.SetMaskedField(configs...)
}

// sendRequest handles all HTTP methods using a single function
func (r *Request) sendRequest(method RequestMethod) (*Response, error) {
	r.method = string(method)
	return r.chooseExecutionStrategy()
}

// Get sends a GET request
func (r *Request) Get() (*Response, error) {
	return r.sendRequest(GET)
}

// Post sends a POST request
func (r *Request) Post() (*Response, error) {
	return r.sendRequest(POST)
}

// Patch sends a PATCH request
func (r *Request) Patch() (*Response, error) {
	return r.sendRequest(PATCH)
}

// Put sends a PUT request
func (r *Request) Put() (*Response, error) {
	return r.sendRequest(PUT)
}

// Delete sends a DELETE request
func (r *Request) Delete() (*Response, error) {
	return r.sendRequest(DELETE)
}

// chooseExecutionStrategy determines which execution method to use based on circuit breaker configuration
func (r *Request) chooseExecutionStrategy() (*Response, error) {
	if r.Cb != nil && r.cbRedis != nil {
		return r.execute()
	}
	if r.Cb != nil {
		return r.executeWithCb()
	}
	if r.cbRedis != nil {
		return r.executeWithCbRedis()
	}
	return r.execute()
}

func (r *Request) fetchErrors() error {
	if len(r.errs) > 0 {
		var e error
		for _, err := range r.errs {
			e = err
			break
		}
		return e
	}
	return nil
}

func (r *Request) execute() (*Response, error) {
	r.log.initiate()
	r.log.setRequest(&http.Request{
		Method: r.method,
		URL:    r.url,
		Header: r.header,
	})
	defer r.log.writeLog()
	// Fetch errors if any
	if err := r.fetchErrors(); err != nil {
		r.log.setError(err)
		return nil, err
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(r.ctx, r.method, r.url.String(), r.body)
	if err != nil {
		r.log.setError(err)
		return nil, err
	}
	r.log.setRequest(req)

	// Set headers
	req.Header = r.header

	// Execute request
	resp, err := r.client.Do(req)
	if err != nil {
		r.log.setError(err)
		return nil, err
	}

	return (*Response)(resp), nil
}

func (r *Request) executeWithCb() (*Response, error) {
	r.log.initiate()
	r.log.setRequest(&http.Request{
		Method: r.method,
		URL:    r.url,
		Header: r.header,
	})
	defer r.log.writeLog()

	// Fetch errors if any
	if err := r.fetchErrors(); err != nil {
		r.log.setError(err)
		return nil, err
	}

	if !r.Cb.allow() {
		r.log.setError(errRefuse)
		return nil, errRefuse
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(r.ctx, r.method, r.url.String(), r.body)
	if err != nil {
		r.log.setError(err)
		return nil, err
	}
	r.log.setRequest(req)

	// Set headers
	req.Header = r.header

	// Execute request
	resp, err := r.client.Do(req)
	if err != nil {
		r.log.setError(err)
		return nil, err
	}

	if inArray(
		resp.StatusCode,
		[]int{
			http.StatusRequestTimeout,
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		},
	) {
		r.Cb.reportResult(false)
	} else {
		r.Cb.reportResult(true)
	}

	return (*Response)(resp), nil
}

func (r *Request) executeWithCbRedis() (*Response, error) {
	r.log.initiate()
	r.log.setRequest(&http.Request{
		Method: r.method,
		URL:    r.url,
		Header: r.header,
	})
	defer r.log.writeLog()

	// Fetch errors if any
	if err := r.fetchErrors(); err != nil {
		r.log.setError(err)
		return nil, err
	}

	if err := r.cbRedis.allowRequest(r.cbKey); err != nil {
		r.log.setError(err)
		return nil, err
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(r.ctx, r.method, r.url.String(), r.body)
	if err != nil {
		r.log.setError(err)
		return nil, err
	}
	r.log.setRequest(req)

	// Set headers
	req.Header = r.header

	// Execute request
	resp, err := r.client.Do(req)
	if err != nil {
		r.log.setError(err)
		return nil, err
	}

	if inArray(
		resp.StatusCode,
		[]int{
			http.StatusRequestTimeout,
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		},
	) {
		r.cbRedis.recordFailure(r.cbKey)
	} else {
		r.cbRedis.reset(r.cbKey)
	}

	return (*Response)(resp), nil
}
