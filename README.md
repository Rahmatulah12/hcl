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

### Example With Circuit Breaker

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
	cb := hcl.NewCircuitBreaker(&hcl.CircuitBreaker{
		Client: redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "", // no password set
			DB:       0,  // use default DB
		}),
		FailureLimit: 3,
		ResetTimeout: 2 * time.Second,
	})

	r := hcl.New(&hcl.HCL{Client: client, Cb: cb})

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