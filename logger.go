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
	Time    string   `json:"time"`
	Level   string   `json:"level"`
	Latency string   `json:"latency"`
	Error   string   `json:"error"`
	Req     request  `json:"request,omitempty"`
	Resp    response `json:"response,omitempty"`
}

type request struct {
	Host   string      `json:"host,omitempty"`
	Path   string      `json:"path,omitempty"`
	Query  url.Values  `json:"query,omitempty"`
	Header http.Header `json:"header,omitempty"`
	Method string      `json:"method,omitempty"`
	Body   string      `json:"payload,omitempty"`
}

type response struct {
	StatusCode int    `json:"statusCode,omitempty"`
	Body       string `json:"body,omitempty"`
}

type Log struct {
	start        time.Time
	l            log
	maskedConfig []*MaskConfig
}

func NewLog() *Log {
	return &Log{
		l:            log{},
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
	if lg == nil || req == nil || req.URL == nil {
		return
	}

	lg.l.Req.Host = req.URL.Host
	lg.l.Req.Path = req.URL.Path
	lg.l.Req.Query = req.URL.Query()
	lg.l.Req.Header = req.Header
	lg.l.Req.Method = req.Method

	if req.Body != nil {
		reqBody, err := io.ReadAll(req.Body)
		if err == nil {
			lg.l.Req.Body = strings.Join(strings.Fields(string(reqBody)), "")
			req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}
	}
}

func (lg *Log) setResponse(resp *http.Response) {
	if lg == nil || resp == nil {
		return
	}

	lg.l.Resp.StatusCode = resp.StatusCode

	if resp.Body != nil {
		respBody, err := io.ReadAll(resp.Body)
		if err == nil {
			lg.l.Resp.Body = strings.Join(strings.Fields(string(respBody)), "")
			resp.Body = io.NopCloser(bytes.NewBuffer(respBody))
		}
	}
}

func (lg *Log) setError(err error) {
	if lg == nil || err == nil {
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
	lg.l.Latency = fmt.Sprintf("%d ms", latency)

	dataLog := convertInterfaceToJson(lg.l)
	dtLog := dataLog

	if len(lg.maskedConfig) > 0 {
		masked := maskJSON(dataLog, lg.maskedConfig)
		mapped := lg.mapperLog(masked)
		if mapped != "" {
			dtLog = mapped
		}
	}

	fmt.Println(dtLog)
}

func (lg *Log) mapperLog(jsonStr string) string {
	if lg == nil || jsonStr == "" {
		return ""
	}

	var l log
	err := json.Unmarshal([]byte(jsonStr), &l)
	if err != nil {
		return ""
	}

	return convertInterfaceToJson(l)
}
