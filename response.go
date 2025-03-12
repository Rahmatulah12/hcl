package hcl

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const (
	JSON = "json"
	XML  = "xml"
)

type Response http.Response

func (r *Response) ByteResult() ([]byte, error) {
	return io.ReadAll(r.Body)
}

func (r *Response) Result(format string, target interface{}) error {
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

	switch format {
	case JSON:
		err = json.Unmarshal(bByte, target)
	case XML:
		err = xml.Unmarshal(bByte, target)
	default:
		return errors.New("unsupported format, use 'json' or 'xml'")
	}

	if err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}
