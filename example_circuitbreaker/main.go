package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
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
	pwd, err := os.Getwd()

	if err != nil {
		panic(err)
	}

	file, err := os.Open(pwd + "/mask.go")
	if err != nil {
		panic(err.Error())
	}
	// get request
	resp, err := r.SetUrl("http://localhost:3000/networkprofile/1122334455").
		SetHeaders(map[string]string{"Content-Type": "application/json"}).
		SetQueryParams(map[string]string{"a": "b", "c": "d"}).
		SetHeader("cicak", "cicak").
		SetFormData(map[string]interface{}{
			"test":  "123",
			"image": file,
		}).
		Get()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			return
		}
	}()

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
