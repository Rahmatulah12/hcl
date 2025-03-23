package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Rahmatulah12/hcl"
)

type ResponseProfile struct {
	Profiles struct {
		Balance      string `json:"balance"`
		CustomerName string `json:"customer_name"`
		CustomerType string `json:"customer_type"`
		CustType     string `json:"custtype"`
		Location     string `json:"location"`
	} `json:"profiles"`
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
	file, err := os.Open("/home/rahmatullah/go/src/plugin/hcl/util.go")
	if err != nil {
		panic(err.Error())
	}
	// get request
	resp, err := r.SetUrl("http://localhost:3000/networkprofile/1122334455").
		SetHeaders(map[string]string{"Content-Type": "application/json"}).
		SetQueryParams(map[string]string{"a": "b", "c": "d"}).
		SetHeader("cicak", "cicak").
		SetHeader("X-API-KEY", "abcdefghijklmnopqrstu").
		SetHeader("API_KEY", "abcdefghijklmnopqrstu").
		SetMaskedFields([]string{"Cicak", "x-api-key"}).
		SetMaskedFields([]string{"api_key", "payload"}).
		SetFormData(map[string]interface{}{
			"msisdn":     "081292021531",
			"initialize": false,
			"file":       file,
		}).
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

	s := &ResponseProfile{}
	err = resp.Result(hcl.JSON, s)
	if err != nil {
		return
	}
	fmt.Println(s)
}
