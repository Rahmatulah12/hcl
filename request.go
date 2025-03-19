package hcl

import (
	"bytes"
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
	MethodGet    RequestMethod = http.MethodGet
	MethodPost   RequestMethod = http.MethodPost
	MethodPatch  RequestMethod = http.MethodPatch
	MethodPut    RequestMethod = http.MethodPut
	MethodDelete RequestMethod = http.MethodDelete

	msg_failed_key_val = "something went wrong, please set key and value request"
	msg_failed_body    = "something went wrong, please set body request"
	example            = "http://example.com"
	msg_empty_url      = "something went wrong, please set uri request"
	content_type       = "Content-Type"
)

type Request struct {
	request *http.Request
	client  *http.Client
	Cb      *CircuitBreaker
	cbRedis *CircuitBreakerRedis
	cbKey   string
	log     *Log
}

type HCL struct {
	Client    *http.Client
	Cb        *CircuitBreaker
	CbRedis   *CircuitBreakerRedis
	EnableLog bool
}

func New(hcl *HCL) *Request {
	req, err := http.NewRequest(http.MethodGet, "", nil)

	if err != nil {
		panic(err.Error())
	}

	var cb *CircuitBreaker
	if hcl.Cb != nil {
		cb = &CircuitBreaker{
			failureCount:  hcl.Cb.failureCount,
			resetTimeout:  hcl.Cb.resetTimeout,
			halfOpenLimit: hcl.Cb.halfOpenLimit,
		}
	}

	var cbRedis *CircuitBreakerRedis
	if hcl.CbRedis != nil {
		cbRedis = &CircuitBreakerRedis{
			ctx:          hcl.CbRedis.ctx,
			Client:       hcl.CbRedis.Client,
			FailureLimit: hcl.CbRedis.FailureLimit,
			ResetTimeout: hcl.CbRedis.ResetTimeout,
		}
	}

	var log *Log
	if hcl.EnableLog {
		log = NewLog()
	}

	return &Request{
		request: req,
		client:  hcl.Client,
		Cb:      cb,
		cbRedis: cbRedis,
		log:     log,
	}
}

func (r *Request) SetUrl(uri string) *Request {
	if uri == "" {
		panic(msg_empty_url)
	}

	parsedUrl, err := url.Parse(uri)
	if err != nil {
		panic(err.Error())
	}

	r.request.URL = parsedUrl

	return r
}

func (r *Request) SetQueryParam(key, val string) *Request {
	if key == "" || val == "" {
		panic(msg_failed_key_val)
	}

	q := r.request.URL.Query()
	q.Set(key, val)
	r.request.URL.RawQuery = q.Encode()

	return r
}

func (r *Request) SetQueryParams(val map[string]string) *Request {
	if val == nil {
		return r
	}
	params := r.request.URL.Query()

	for k, v := range val {
		params.Add(k, v)
	}
	r.request.URL.RawQuery = params.Encode()

	return r
}

func (r *Request) SetHeader(key, val string) *Request {
	if key == "" || val == "" {
		panic(msg_failed_key_val)
	}

	r.request.Header.Set(key, val)

	return r
}

func (r *Request) SetHeaders(val map[string]string) *Request {
	if val == nil {
		panic("something went wrong, please set header request")
	}

	for k, v := range val {
		if k == "" || v == "" {
			continue
		}
		r.request.Header.Set(k, v)
	}

	return r
}

func (r *Request) SetJsonPayload(body interface{}) *Request {
	if body == nil {
		panic(msg_failed_body)
	}

	b, err := json.Marshal(body)
	if err != nil {
		panic(err.Error())
	}

	r.request.Body = io.NopCloser(bytes.NewBuffer(b))
	r.request.Header.Set(content_type, "application/json")

	return r
}

func (r *Request) SetXMLPayload(body interface{}) *Request {
	if body == nil {
		panic(msg_failed_body)
	}

	b, err := xml.Marshal(body)
	if err != nil {
		panic(err.Error())
	}

	r.request.Body = io.NopCloser(bytes.NewBuffer(b))
	r.request.Header.Set(content_type, "application/xml")

	return r
}

func (r *Request) SetFormData(data map[string]interface{}) *Request {
	if data == nil {
		panic("form data cannot be nil")
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
				panic(err.Error())
			}
			io.Copy(part, v)
		case *os.File:
			part, err := writer.CreateFormFile(key, v.Name())
			if err != nil {
				panic(err)
			}
			io.Copy(part, v)
		default:
			msg := fmt.Sprintf("Unsupported type for key: %s\n", key)
			panic(msg)
		}
	}

	writer.Close()

	r.request.Body = io.NopCloser(&body)
	r.request.Header.Set(content_type, writer.FormDataContentType())

	return r
}

func (r *Request) SetFormURLEncoded(data map[string]string) *Request {
	if data == nil {
		panic(msg_failed_body)
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

	r.request.Body = io.NopCloser(bytes.NewBufferString(encodedForm))
	r.request.Header.Set(content_type, "application/x-www-form-urlencoded")

	return r
}

func (r *Request) SetCircuitBreakerKey(key string) *Request {
	if key == "" {
		panic("key cannot be empty")
	}

	r.cbKey = key
	return r
}

// sendRequest handles all HTTP methods using a single function
func (r *Request) sendRequest(method RequestMethod) (*Response, error) {
	r.request.Method = string(method)
	return r.chooseExecutionStrategy()
}

// Get sends a GET request
func (r *Request) Get() (*Response, error) {
	return r.sendRequest(MethodGet)
}

// Post sends a POST request
func (r *Request) Post() (*Response, error) {
	return r.sendRequest(MethodPost)
}

// Patch sends a PATCH request
func (r *Request) Patch() (*Response, error) {
	return r.sendRequest(MethodPatch)
}

// Put sends a PUT request
func (r *Request) Put() (*Response, error) {
	return r.sendRequest(MethodPut)
}

// Delete sends a DELETE request
func (r *Request) Delete() (*Response, error) {
	return r.sendRequest(MethodDelete)
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

func (r *Request) execute() (*Response, error) {
	if err := r.validateURL(); err != nil {
		return nil, err
	}

	return r.performRequest(nil)
}

func (r *Request) executeWithCb() (*Response, error) {
	if err := r.validateURL(); err != nil {
		return nil, err
	}

	if !r.Cb.allow() {
		r.log.setError(errRefuse)
		return nil, errRefuse
	}

	return r.performRequest(func(resp *http.Response, err error) {
		if err == nil && resp.StatusCode < http.StatusInternalServerError {
			r.Cb.reportResult(true)
		} else if resp != nil && resp.StatusCode >= http.StatusInternalServerError {
			r.Cb.reportResult(false)
		}
	})
}

func (r *Request) executeWithCbRedis() (*Response, error) {
	if err := r.validateURL(); err != nil {
		return nil, err
	}

	if err := r.validateRedisKey(); err != nil {
		return nil, err
	}

	if err := r.cbRedis.allowRequest(r.cbKey); err != nil {
		r.log.setError(err)
		return nil, err
	}

	return r.performRequest(func(resp *http.Response, err error) {
		if err == nil && resp.StatusCode < http.StatusBadRequest {
			r.cbRedis.reset(r.cbKey)
		} else if resp != nil && resp.StatusCode >= http.StatusInternalServerError {
			r.cbRedis.recordFailure(r.cbKey)
		}
	})
}

func (r *Request) validateURL() error {
	if r.request.URL.String() == "" {
		return errors.New(msg_empty_url)
	}
	return nil
}

func (r *Request) validateRedisKey() error {
	if r.cbKey == "" {
		err := fmt.Errorf("circuit breaker key cannot be empty")
		r.log.setError(err)
		return err
	}
	return nil
}

// performRequest handles the common request execution logic
func (r *Request) performRequest(callback func(*http.Response, error)) (*Response, error) {
	r.log.initiate()
	r.log.setRequest(r.request)
	defer r.log.writeLog()

	resp, err := r.client.Do(r.request)
	r.log.setResponse(resp)

	if callback != nil {
		callback(resp, err)
	}

	if err != nil {
		r.log.setError(err)
		return (*Response)(resp), err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		err = fmt.Errorf("error, response from client: %s", resp.Status)
		r.log.setError(err)
		return (*Response)(resp), err
	}

	return (*Response)(resp), nil
}
