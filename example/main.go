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

	proccess(r)
}

func proccess(r *hcl.Request) {
	// get request
	resp, err := r.SetUrl("http://localhost:3000/networkprofile/1122334455").
		SetHeaders(map[string]string{"Content-Type": "application/json"}).
		SetQueryParams(map[string]string{"a": "b", "c": "d"}).
		SetHeader("cicak", "cicak").
		SetHeader("X-API-KEY", "abcdefghijklmnopqrstu").
		SetHeader("API_KEY", "abcdefghijklmnopqrstu").
		SetMaskedFields([]string{"Cicak", "x-api-key"}).
		SetMaskedFields([]string{"api_key"}).
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
