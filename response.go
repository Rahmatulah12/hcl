package hcl

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type Response http.Response

func (r *Response) ByteResult() ([]byte, error) {
	return io.ReadAll(r.Body)
}

func (r *Response) ResultJson(target interface{}) error {
	if target == nil {
		return errors.New("target struct cannot be nil")
	}

	if !isPointer(target) {
		return errors.New("target must be a pointer")
	}

	bByte, err := r.ByteResult()
	if err != nil {
		return err
	}

	err = json.Unmarshal(bByte, target)
	if err != nil {
		return errors.New("failed to decode json response: " + err.Error())
	}

	return nil
}

func (r *Response) ResultXML(target interface{}) error {
	if target == nil {
		return errors.New("target struct cannot be nil")
	}

	if !isPointer(target) {
		return errors.New("target must be a pointer")
	}

	bByte, err := r.ByteResult()
	if err != nil {
		return err
	}

	err = xml.Unmarshal(bByte, target)
	if err != nil {
		return fmt.Errorf("failed to decode XML response: " + err.Error())
	}

	return nil
}
