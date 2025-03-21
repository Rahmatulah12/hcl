# HCL (Http Client Helper)

Helper for http client.

## Installation

```bash
  go get github.com/Rahmatulah12/hcl@latest
```

### Example Without Circuit Breaker

```go
package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/Rahmatulah12/hcl"
)

type Response struct {
	Transaction    Transaction    `json:"transaction"`
	Service        Service        `json:"service"`
	NetworkProfile NetworkProfile `json:"network_profile"`
}

type Transaction struct {
	TransactionID string `json:"transaction_id"`
	Channel       string `json:"channel"`
	StatusCode    string `json:"status_code"`
	StatusDesc    string `json:"status_desc"`
}

type Service struct {
	ServiceID string `json:"service_id"`
}

type NetworkProfile struct {
	ProductID string `json:"product_id"`
	ScpID     string `json:"scp_id"`
}

func main() {
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        100,
		IdleConnTimeout:     60 * time.Second,
		MaxIdleConnsPerHost: 100,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	r := hcl.New(&hcl.HCL{Client: client})

	for i := 0; i < 5; i++ {
		proccess(r)
	}
}

func proccess(r *hcl.Request) {
	resp, err := r.SetUrl("http://localhost:3000/networkprofile/1122334455").
		SetHeaders(map[string]string{"Content-Type": "application/json"}).
		SetQueryParams(map[string]string{"a": "b", "c": "d"}).
		SetHeader("cicak", "cicak").
		Get()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer resp.Body.Close()

	s := &Response{}
	err = resp.Result(hcl.JSON, s)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(s)
}
```

### Example with log
```go
package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/Rahmatulah12/hcl"
)

type Response struct {
	Transaction    Transaction    `json:"transaction"`
	Service        Service        `json:"service"`
	NetworkProfile NetworkProfile `json:"network_profile"`
}

type Transaction struct {
	TransactionID string `json:"transaction_id"`
	Channel       string `json:"channel"`
	StatusCode    string `json:"status_code"`
	StatusDesc    string `json:"status_desc"`
}

type Service struct {
	ServiceID string `json:"service_id"`
}

type NetworkProfile struct {
	ProductID string `json:"product_id"`
	ScpID     string `json:"scp_id"`
}

func main() {
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        100,
		IdleConnTimeout:     60 * time.Second,
		MaxIdleConnsPerHost: 100,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	r := hcl.New(&hcl.HCL{
		Client:    client,
		EnableLog: true,
	})

	for i := 0; i < 500; i++ {
		time.Sleep(500 * time.Millisecond)
		proccess(r)
	}
}

func proccess(r *hcl.Request) {
	// get request
	resp, err := r.SetUrl("http://localhost:3000/networkprofile/1122334455").
		SetHeaders(map[string]string{"Content-Type": "application/json"}).
		SetQueryParams(map[string]string{"a": "b", "c": "d"}).
		SetHeader("cicak", "cicak").
		Get()

	if err != nil {
		return
	}
	defer resp.Body.Close()

	// byte response example
	// b, err := resp.ByteResult()
	// if err != nil {
	// 	panic(err.Error())
	// }
	// fmt.Println(string(b))

	s := &Response{}
	err = resp.Result(hcl.JSON, s)
	if err != nil {
		return
	}
	fmt.Println(s)
}
```

### It's success log
```json
{"time":"2025-03-19T21:52:59+07:00","level":"info","latency":"27 ms","error":null,"request":{"host":"localhost:3000","path":"/networkprofile/1122334455","query":{"a":["b"],"c":["d"]},"header":{"Cicak":["cicak"],"Content-Type":["application/json"]},"method":"GET","body":""},"response":{"statusCode":200,"body":"{\"transaction\":{\"transaction_id\":\"C002250225184414229215310\",\"channel\":\"b0\",\"status_code\":\"00000\",\"status_desc\":\"Success\"},\"service\":{\"service_id\":\"6281292021531\"},\"network_profile\":{\"product_id\":\"Simpati\",\"scp_id\":\"R01\"}}"}}
```

### It's error/failed log
```json
{"time":"2025-03-19T21:54:50+07:00","level":"error","latency":"1 ms","error":"Get \"http://localhost:3000/networkprofile/1122334455?a=b\u0026c=d\": dial tcp 127.0.0.1:3000: connect: connection refused","request":{"host":"localhost:3000","path":"/networkprofile/1122334455","query":{"a":["b"],"c":["d"]},"header":{"Cicak":["cicak"],"Content-Type":["application/json"]},"method":"GET","body":""},"response":{"statusCode":null,"body":null}}
```

### It's log, with masking
```json
{"error":"Get \"http://localhost:3000/networkprofile/1122334455?a=b\u0026c=d\": dial tcp 127.0.0.1:3000: connect: connection refused","latency":"0 ms","level":"error","request":{"body":"","header":{"Api_key":["*********************"],"Cicak":["*****"],"Content-Type":["application/json"],"X-Api-Key":["*********************"]},"host":"localhost:3000","method":"GET","path":"/networkprofile/1122334455","query":{"a":["b"],"c":["d"]}},"response":{},"time":"2025-03-21T22:12:23+07:00"}
```

### Example Circuit Breaker with Redis

```go
package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/Rahmatulah12/hcl"
	"github.com/redis/go-redis/v9"
)

type Response struct {
	Transaction    Transaction    `json:"transaction"`
	Service        Service        `json:"service"`
	NetworkProfile NetworkProfile `json:"network_profile"`
}

type Transaction struct {
	TransactionID string `json:"transaction_id"`
	Channel       string `json:"channel"`
	StatusCode    string `json:"status_code"`
	StatusDesc    string `json:"status_desc"`
}

type Service struct {
	ServiceID string `json:"service_id"`
}

type NetworkProfile struct {
	ProductID string `json:"product_id"`
	ScpID     string `json:"scp_id"`
}

func main() {
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        100,
		IdleConnTimeout:     60 * time.Second,
		MaxIdleConnsPerHost: 100,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}
	cbRedis := hcl.NewCircuitBreakerRedis(&hcl.CircuitBreakerRedis{
		Client: redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "", // no password set
			DB:       0,  // use default DB
		}),
		FailureLimit: 3,
		ResetTimeout: 2 * time.Second,
	})

	r := hcl.New(&hcl.HCL{Client: client, CbRedis: cbRedis})

	for i := 0; i < 10; i++ {
		proccess(r)
		time.Sleep(1)
	}
}

func proccess(r *hcl.Request) {
	// get request
	resp, err := r.SetUrl("http://localhost:3000/networkprofile/1122334455").
		SetHeaders(map[string]string{"Content-Type": "application/json"}).
		SetQueryParams(map[string]string{"a": "b", "c": "d"}).
		SetCircuitBreakerKey("test_a").
		SetHeader("cicak", "cicak").
		Get()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer resp.Body.Close()

	// byte response example
	// b, err := resp.ByteResult()
	// if err != nil {
	// 	panic(err.Error())
	// }
	// fmt.Println(string(b))

	s := &Response{}
	err = resp.Result(hcl.JSON, s)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(s)
}
```

### Example Circuit Breaker without Redis
```go
package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/Rahmatulah12/hcl"
)

type Response struct {
	Transaction    Transaction    `json:"transaction"`
	Service        Service        `json:"service"`
	NetworkProfile NetworkProfile `json:"network_profile"`
}

type Transaction struct {
	TransactionID string `json:"transaction_id"`
	Channel       string `json:"channel"`
	StatusCode    string `json:"status_code"`
	StatusDesc    string `json:"status_desc"`
}

type Service struct {
	ServiceID string `json:"service_id"`
}

type NetworkProfile struct {
	ProductID string `json:"product_id"`
	ScpID     string `json:"scp_id"`
}

func main() {
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        100,
		IdleConnTimeout:     60 * time.Second,
		MaxIdleConnsPerHost: 100,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	cb := hcl.NewCircuitBreaker(hcl.CircuitBreakerOption{
		MaxFailures:   10,
		HalfOpenLimit: 5,
		ResetTimeout:  10 * time.Second,
	})

	r := hcl.New(&hcl.HCL{Client: client, Cb: cb})

	for i := 0; i < 1000; i++ {
		proccess(r)
		time.Sleep(1 * time.Second)
	}
}

func proccess(r *hcl.Request) {
	// get request
	resp, err := r.SetUrl("http://localhost:3000/networkprofile/1122334455").
		SetHeaders(map[string]string{"Content-Type": "application/json"}).
		SetQueryParams(map[string]string{"a": "b", "c": "d"}).
		SetHeader("cicak", "cicak").
		Get()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer resp.Body.Close()

	// byte response example
	// b, err := resp.ByteResult()
	// if err != nil {
	// 	panic(err.Error())
	// }
	// fmt.Println(string(b))

	s := &Response{}
	err = resp.Result(hcl.JSON, s)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(s)
}
```