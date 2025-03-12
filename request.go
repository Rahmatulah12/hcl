package hcl

import (
	"bytes"
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

type Client *http.Client

const (
	msg_failed_key_val = "something went wrong, please set key and value request"
	msg_failed_body    = "something went wrong, please set body request"
)

type Request struct {
	request *http.Request
	Client  *http.Client
}

func New(client Client) *Request {
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)

	if err != nil {
		panic(err.Error())
	}

	return &Request{
		request: req,
		Client:  client,
	}
}

func (r *Request) SetUrl(uri string) *Request {
	if uri == "" {
		panic("something went wrong, please set uri request")
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
	r.request.Header.Set("Content-Type", "application/json")

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
	r.request.Header.Set("Content-Type", "application/xml")

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
	r.request.Header.Set("Content-Type", writer.FormDataContentType())

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
	r.request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return r
}

func (r *Request) execute() (*Response, error) {
	resp, err := r.Client.Do(r.request)
	return (*Response)(resp), err
}

func (r *Request) Get() (*Response, error) {
	r.request.Method = http.MethodGet
	return r.execute()
}

func (r *Request) Post() (*Response, error) {
	r.request.Method = http.MethodPost
	return r.execute()
}

func (r *Request) Patch() (*Response, error) {
	r.request.Method = http.MethodPatch
	return r.execute()
}

func (r *Request) Put() (*Response, error) {
	r.request.Method = http.MethodPut
	return r.execute()
}

func (r *Request) Delete() (*Response, error) {
	r.request.Method = http.MethodDelete
	return r.execute()
}
