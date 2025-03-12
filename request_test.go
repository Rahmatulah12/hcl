package hcl_test

import (
	"net/http"
	"testing"

	"github.com/Rahmatulah12/hcl"
	"github.com/stretchr/testify/assert"
)

func TestNewRequest(t *testing.T) {
	client := &http.Client{}
	req := hcl.New(client)

	assert.NotNil(t, req, "Request object should not be nil")
	assert.NotNil(t, req.Client, "HTTP Client should not be nil")
}

func TestSetUrl(t *testing.T) {
	client := &http.Client{}
	req := hcl.New(client)
	testUrl := "http://example.com/test"
	req.SetUrl(testUrl)
}

func TestSetQueryParam(t *testing.T) {
	client := &http.Client{}
	req := hcl.New(client)
	req.SetUrl("http://example.com")
	req.SetQueryParam("key", "value")
}

func TestSetQueryParams(t *testing.T) {
	client := &http.Client{}
	req := hcl.New(client)
	req.SetUrl("http://example.com")
	params := map[string]string{"key1": "value1", "key2": "value2"}
	req.SetQueryParams(params)
}
