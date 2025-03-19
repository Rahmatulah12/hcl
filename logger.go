package hcl

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
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
	Host   any `json:"host"`
	Path   any `json:"path"`
	Query  any `json:"query"`
	Header any `json:"header"`
	Method any `json:"method"`
	Body   any `json:"body"`
}

type response struct {
	StatusCode any `json:"statusCode"`
	Body       any `json:"body"`
}

type Log struct {
	start time.Time
	l     log
}

func NewLog() *Log {
	return &Log{}
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
	fmt.Println(convertInterfaceToJson(lg.l))
}
