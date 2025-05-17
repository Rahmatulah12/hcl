package hcl

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
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
	method          string
	url             *url.URL
	header          http.Header
	body            io.ReadCloser
	ctx             context.Context
	client          *http.Client
	Cb              *CircuitBreaker
	cbRedis         *CircuitBreakerRedis
	cbKey           string
	log             *Log
	errs            []error
	isRepeatableLog bool
	closeRequest    bool
	errHttpCodes    []int
}

type HCL struct {
	Context context.Context
	Client  *http.Client
	Cb      *CircuitBreaker
	CbRedis *CircuitBreakerRedis
}

func New(hcl *HCL) *Request {
	var (
		ctx     context.Context
		client  *http.Client
		cb      *CircuitBreaker
		cbRedis *CircuitBreakerRedis
	)

	if hcl != nil {
		ctx = hcl.Context
		client = hcl.Client
		cb = cloneCircuitBreaker(hcl.Cb)
		cbRedis = cloneCircuitBreakerRedis(hcl.CbRedis)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	return &Request{
		ctx:     ctx,
		client:  client,
		Cb:      cb,
		cbRedis: cbRedis,
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

func (r *Request) EnableLog(isRepeatableLog bool) *Request {
	if r == nil {
		return nil
	}

	r.isRepeatableLog = isRepeatableLog
	r.log = initializeLog(true)
	return r
}

func (r *Request) turnOffLog() {
	r.isRepeatableLog = false
	r.log = nil
}

func (r *Request) SetUrl(uri string) *Request {
	// Check if the request object is nil
	if r == nil {
		return nil
	}

	if uri == "" {
		r.errs = append(r.errs, errors.New(msgEmptyUrl))
		return r
	}

	_, err := url.ParseRequestURI(uri)
	if err != nil {
		r.errs = append(r.errs, errors.New("invalid URL: "+err.Error()))
		return r
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		r.errs = append(r.errs, errors.New("failed to parse URL : "+err.Error()))
		return r
	}

	r.url = parsed
	return r
}

func (r *Request) SetQueryParam(key, val string) *Request {
	// Check if the request object is nil
	if r == nil {
		return nil
	}

	if key == "" || val == "" {
		r.errs = append(r.errs, errors.New(msgFailedKeyVal))
		return r
	}

	if r.url == nil {
		r.errs = append(r.errs, fmt.Errorf("url is not set, call SetUrl first"))
		return r
	}

	// Validasi skema dan host URL
	if r.url.Scheme == "" || r.url.Host == "" {
		r.errs = append(r.errs, fmt.Errorf("invalid URL: missing scheme or host"))
		return r
	}

	// Validasi ulang URL untuk memastikan URL valid secara sintaksis
	_, err := url.ParseRequestURI(r.url.String())
	if err != nil {
		r.errs = append(r.errs, errors.New("invalid URL: %w"+err.Error()))
		return r
	}

	q := r.url.Query()
	q.Set(key, val)
	r.url.RawQuery = q.Encode()

	return r
}

func (r *Request) SetQueryParams(val map[string]string) *Request {
	// Check if the request object is nil
	if r == nil {
		return nil
	}

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
		r.errs = append(r.errs, errors.New(msgFailedKeyVal))
		return r
	}

	r.header.Set(key, val)

	return r
}

func (r *Request) SetHeaders(val map[string]string) *Request {
	// Check if the request object is nil
	if r == nil {
		return nil
	}

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
	// Check if the request object is nil
	if r == nil {
		return nil
	}

	if body == nil {
		r.errs = append(r.errs, errors.New(msgFailedBody))
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
			r.errs = append(r.errs, errors.New("failed to marshal json: "+err.Error()))
		}
	}

	r.header.Set(contentType, contentTypeJSON)
	r.body = io.NopCloser(bytes.NewBuffer(b))

	return r
}

func (r *Request) SetXMLPayload(body interface{}) *Request {
	// Check if the request object is nil
	if r == nil {
		return nil
	}

	if body == nil {
		r.errs = append(r.errs, errors.New(msgFailedBody))
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
	// Check if the request object is nil
	if r == nil {
		return nil
	}

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
			err := writer.WriteField(key, v)
			if err != nil {
				r.errs = append(r.errs, err)
				return r
			}
		case int, int8, int16, int32, int64:
			err := writer.WriteField(key, strconv.FormatInt(v.(int64), 10))
			if err != nil {
				r.errs = append(r.errs, err)
				return r
			}
		case float32, float64:
			err := writer.WriteField(key, strconv.FormatFloat(v.(float64), 'f', -1, 64))
			if err != nil {
				r.errs = append(r.errs, err)
				return r
			}
		case bool:
			err := writer.WriteField(key, strconv.FormatBool(v))
			if err != nil {
				r.errs = append(r.errs, err)
				return r
			}
		case *os.File:
			part, err := writer.CreateFormFile(key, v.Name())
			if err != nil {
				r.errs = append(r.errs, err)
				return r
			}
			_, err = io.Copy(part, v)
			if err != nil {
				r.errs = append(r.errs, err)
				return r
			}
		case io.Reader:
			part, err := writer.CreateFormFile(key, "file") // Default filename "file"
			if err != nil {
				r.errs = append(r.errs, err)
				return r
			}
			_, err = io.Copy(part, v)
			if err != nil {
				r.errs = append(r.errs, err)
				return r
			}
		default:
			msg := fmt.Sprintf("Unsupported type for key: %s\n", key)
			r.errs = append(r.errs, errors.New(msg))
			return r
		}
	}

	err := writer.Close()
	if err != nil {
		r.errs = append(r.errs, err)
		return r
	}

	r.header.Set(contentType, writer.FormDataContentType())
	r.body = io.NopCloser(&body)

	return r
}

func (r *Request) SetFormURLEncoded(data map[string]string) *Request {
	// Check if the request object is nil
	if r == nil {
		return nil
	}

	if data == nil {
		r.errs = append(r.errs, errors.New(msgFailedBody))
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
	// Check if the request object is nil
	if r == nil {
		return nil
	}

	if key == "" {
		r.errs = append(r.errs, fmt.Errorf("key cannot be empty"))
		return r
	}

	r.cbKey = key
	return r
}

func (r *Request) CloseRequestAfterResponse() *Request {
	// Check if the request object is nil
	if r == nil {
		return nil
	}

	r.closeRequest = true
	return r
}

func (r *Request) SetMaskedField(conf ...*MaskConfig) *Request {
	// Check if the request object is nil
	if r == nil {
		return nil
	}

	if r.log == nil {
		return r
	}

	if len(conf) <= 0 {
		return r
	}

	r.log.maskedConfig = append(r.log.maskedConfig, conf...)
	return r
}

func (r *Request) SetMaskedFields(configs []*MaskConfig) *Request {
	// Check if the request object is nil
	if r == nil {
		return nil
	}

	if r.log == nil {
		return r
	}

	if len(configs) <= 0 {
		return r
	}

	return r.SetMaskedField(configs...)
}

func (r *Request) SetErrorHttpCodesCircuitBreaker(httpCodes []int) *Request {
	if r == nil {
		return nil
	}

	if len(httpCodes) <= 0 {
		return r
	}

	r.errHttpCodes = httpCodes

	return r
}

// Get sends a GET request
func (r *Request) Get() (*Response, error) {
	// Check if the request object is nil
	if r == nil {
		return nil, errors.New("failed to execute process, please initiate first")
	}

	return r.sendRequest(GET)
}

// Post sends a POST request
func (r *Request) Post() (*Response, error) {
	// Check if the request object is nil
	if r == nil {
		return nil, errors.New("failed to execute process, please initiate first")
	}

	return r.sendRequest(POST)
}

// Patch sends a PATCH request
func (r *Request) Patch() (*Response, error) {
	// Check if the request object is nil
	if r == nil {
		return nil, errors.New("failed to execute process, please initiate first")
	}

	return r.sendRequest(PATCH)
}

// Put sends a PUT request
func (r *Request) Put() (*Response, error) {
	// Check if the request object is nil
	if r == nil {
		return nil, errors.New("failed to execute process, please initiate first")
	}

	return r.sendRequest(PUT)
}

// Delete sends a DELETE request
func (r *Request) Delete() (*Response, error) {
	return r.sendRequest(DELETE)
}

func (r *Request) fetchErrors() error {
	// Check if the request object is nil
	if r == nil {
		return errors.New("failed to execute process, please initiate first")
	}

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

// sendRequest handles all HTTP methods using a single function
func (r *Request) sendRequest(method RequestMethod) (*Response, error) {
	if r == nil {
		return nil, fmt.Errorf("request cannot be nil, please initiate library")
	}

	r.method = string(method)
	return r.chooseExecutionStrategy()
}

// chooseExecutionStrategy determines which execution method to use based on circuit breaker configuration
func (r *Request) chooseExecutionStrategy() (*Response, error) {
	// Check if the request object is nil
	if r == nil {
		return nil, errors.New("failed to execute process, please initiate first")
	}

	// Pre-execution circuit breaker checks
	var cbErr error
	if r.Cb != nil && !r.Cb.allow() {
		cbErr = errRefuse
	} else if r.cbRedis != nil {
		cbErr = r.cbRedis.allowRequest(r.cbKey)
	}

	if cbErr != nil {
		if r.log != nil {
			r.log.setError(cbErr)
		}
		return nil, cbErr
	}

	// Execute the request
	resp, err := r.executeRequest()
	if err != nil {
		return nil, err
	}

	// Post-execution circuit breaker updates
	r.updateCircuitBreaker(resp.StatusCode)

	return resp, nil
}

// executeRequest handles the common HTTP request execution logic
func (r *Request) executeRequest() (*Response, error) {
	if r == nil {
		return nil, fmt.Errorf("request cannot be nil, please initiate library")
	}

	if r.client == nil {
		r.client = http.DefaultClient
	}

	// Initialize logging
	if r.log != nil {
		r.log.initiate()
		r.log.setRequest(&http.Request{
			Method: r.method,
			URL:    r.url,
			Header: r.header,
		})

		defer func() {
			r.log.writeLog()
			if !r.isRepeatableLog {
				r.turnOffLog()
			}
		}()
	}

	// Fetch errors if any
	if err := r.fetchErrors(); err != nil {
		if r.log != nil {
			r.log.setError(err)
		}
		return nil, err
	}

	// Set default context if not provided
	if r.ctx == nil {
		r.ctx = context.Background()
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(r.ctx, r.method, r.url.String(), r.body)
	if err != nil {
		if r.log != nil {
			r.log.setError(err)
		}
		return nil, err
	}

	// Set request details
	req.Header = r.header
	if r.closeRequest {
		req.Close = true
	}

	// Log the request
	if r.log != nil {
		r.log.setRequest(req)
	}

	// Execute request
	resp, err := r.client.Do(req)
	if err != nil {
		if r.log != nil {
			r.log.setError(err)
		}
		return nil, err
	}

	// Log the response
	if r.log != nil {
		r.log.setResponse(resp)
	}

	return (*Response)(resp), nil
}

// updateCircuitBreaker updates the circuit breaker state based on response status
func (r *Request) updateCircuitBreaker(statusCode int) {
	if len(r.errHttpCodes) <= 0 {
		// Define error status codes
		r.errHttpCodes = []int{
			http.StatusRequestTimeout,
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		}
	}

	isErrorStatus := inArray(statusCode, r.errHttpCodes)

	// Update in-memory circuit breaker
	if r.Cb != nil {
		success := true
		if isErrorStatus {
			success = false
		}

		r.Cb.reportResult(success)
	} else if r.cbRedis != nil { // Update Redis-based circuit breaker
		if isErrorStatus {
			r.cbRedis.recordFailure(r.cbKey)
		} else {
			r.cbRedis.reset(r.cbKey)
		}
	}
}
