package hcl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	INFO  = "info"
	ERROR = "error"
)

type log struct {
	Time    any      `json:"time"`
	Level   any      `json:"level"`
	Latency any      `json:"latency"`
	Error   any      `json:"error"`
	Req     request  `json:"request,omitempty"`
	Resp    response `json:"response,omitempty"`
}

type request struct {
	Scheme string      `json:"scheme,omitempty"`
	Host   string      `json:"host,omitempty"`
	Path   string      `json:"path,omitempty"`
	Query  url.Values  `json:"query,omitempty"`
	Header http.Header `json:"header,omitempty"`
	Method string      `json:"method,omitempty"`
	Body   any         `json:"payload,omitempty"`
}

type response struct {
	StatusCode int `json:"statusCode,omitempty"`
	Body       any `json:"body,omitempty"`
}

type Log struct {
	start        time.Time
	l            log
	maskedConfig []*MaskConfig
}

func NewLog() *Log {
	return &Log{
		maskedConfig: make([]*MaskConfig, 0),
	}
}

func (lg *Log) initiate() {
	if lg == nil {
		return
	}

	lg.start = time.Now()
	lg.l.Time = lg.start.Format(time.RFC3339)
	lg.l.Level = INFO
}

func (lg *Log) setRequest(req *http.Request) {
	if lg == nil || req == nil {
		return
	}
	// request
	lg.l.Req.Scheme = req.URL.Scheme
	lg.l.Req.Host = req.URL.Host
	lg.l.Req.Path = req.URL.Path
	lg.l.Req.Query = req.URL.Query()
	lg.l.Req.Header = req.Header
	lg.l.Req.Method = req.Method

	var reqBody []byte
	if req.Body != nil {
		reqBody, _ = io.ReadAll(req.Body)                 // read all body
		req.Body = io.NopCloser(bytes.NewBuffer(reqBody)) // write back body
	}

	lg.l.Req.Body = strings.Join(strings.Fields(string(reqBody)), "")
}

func (lg *Log) setResponse(resp *http.Response) {
	if lg == nil || resp == nil {
		return
	}

	// response
	lg.l.Resp.StatusCode = resp.StatusCode
	var respBody []byte
	if resp.Body != nil {
		respBody, _ = io.ReadAll(resp.Body)                 // read all body
		resp.Body = io.NopCloser(bytes.NewBuffer(respBody)) // write back body
	}

	lg.l.Resp.Body = strings.Join(strings.Fields(string(respBody)), "")
}

func (lg *Log) setError(err error) {
	if lg == nil {
		return
	}

	lg.l.Error = err.Error()
	lg.l.Level = ERROR
}

func (lg *Log) writeLog() {
	if lg == nil {
		return
	}

	latency := time.Since(lg.start).Milliseconds()
	// latency
	lg.l.Latency = fmt.Sprintf("%d ms", latency)
	// write log
	dataLog := convertInterfaceToJson(lg.l)
	var dtLog string

	if len(lg.maskedConfig) > 0 {
		dtLog = maskJSON(dataLog, lg.maskedConfig)
		dtLog = lg.mapperLog(dtLog)
	}

	if dtLog != "" {
		dataLog = dtLog
	}

	fmt.Println(dataLog)
}

func (lg *Log) mapperLog(jsonStr string) string {
	var l *log
	err := json.Unmarshal([]byte(jsonStr), &l)
	if err != nil {
		return ""
	}

	return convertInterfaceToJson(l)
}
