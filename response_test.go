package hcl

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseByteResult(t *testing.T) {
	t.Run("successful read", func(t *testing.T) {
		body := io.NopCloser(bytes.NewBufferString(`{"name":"test"}`))
		resp := &Response{
			Body: body,
		}

		result, err := resp.ByteResult()

		assert.NoError(t, err)
		assert.Equal(t, []byte(`{"name":"test"}`), result)
	})

	t.Run("read error", func(t *testing.T) {
		// Create a reader that returns an error
		errReader := &errorReader{err: errors.New("read error")}
		resp := &Response{
			Body: io.NopCloser(errReader),
		}

		_, err := resp.ByteResult()

		assert.Error(t, err)
		assert.Equal(t, "read error", err.Error())
	})
}

func TestResponseResultJson(t *testing.T) {
	t.Run("nil target", func(t *testing.T) {
		resp := &Response{}
		err := resp.ResultJson(nil)

		assert.Error(t, err)
		assert.Equal(t, "target struct cannot be nil", err.Error())
	})

	t.Run("non-pointer target", func(t *testing.T) {
		resp := &Response{}
		var target struct{}
		err := resp.ResultJson(target)

		assert.Error(t, err)
		assert.Equal(t, "target must be a pointer", err.Error())
	})

	t.Run("read error", func(t *testing.T) {
		errReader := &errorReader{err: errors.New("read error")}
		resp := &Response{
			Body: io.NopCloser(errReader),
		}

		var target struct{}
		err := resp.ResultJson(&target)

		assert.Error(t, err)
		assert.Equal(t, "read error", err.Error())
	})

	t.Run("unmarshal error", func(t *testing.T) {
		body := io.NopCloser(bytes.NewBufferString(`invalid json`))
		resp := &Response{
			Body: body,
		}

		var target struct{}
		err := resp.ResultJson(&target)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode json response")
	})

	t.Run("successful unmarshal", func(t *testing.T) {
		body := io.NopCloser(bytes.NewBufferString(`{"name":"test","age":30}`))
		resp := &Response{
			Body: body,
		}

		type Person struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		var target Person
		err := resp.ResultJson(&target)

		assert.NoError(t, err)
		assert.Equal(t, "test", target.Name)
		assert.Equal(t, 30, target.Age)
	})
}

func TestResponseResultXML(t *testing.T) {
	t.Run("nil target", func(t *testing.T) {
		resp := &Response{}
		err := resp.ResultXML(nil)

		assert.Error(t, err)
		assert.Equal(t, "target struct cannot be nil", err.Error())
	})

	t.Run("non-pointer target", func(t *testing.T) {
		resp := &Response{}
		var target struct{}
		err := resp.ResultXML(target)

		assert.Error(t, err)
		assert.Equal(t, "target must be a pointer", err.Error())
	})

	t.Run("read error", func(t *testing.T) {
		errReader := &errorReader{err: errors.New("read error")}
		resp := &Response{
			Body: io.NopCloser(errReader),
		}

		var target struct{}
		err := resp.ResultXML(&target)

		assert.Error(t, err)
		assert.Equal(t, "read error", err.Error())
	})

	t.Run("unmarshal error", func(t *testing.T) {
		body := io.NopCloser(bytes.NewBufferString(`invalid xml`))
		resp := &Response{
			Body: body,
		}

		var target struct{}
		err := resp.ResultXML(&target)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode XML response")
	})

	t.Run("successful unmarshal", func(t *testing.T) {
		body := io.NopCloser(bytes.NewBufferString(`<person><name>test</name><age>30</age></person>`))
		resp := &Response{
			Body: body,
		}

		type Person struct {
			Name string `xml:"name"`
			Age  int    `xml:"age"`
		}

		var target Person
		err := resp.ResultXML(&target)

		assert.NoError(t, err)
		assert.Equal(t, "test", target.Name)
		assert.Equal(t, 30, target.Age)
	})
}

// Helper type for testing read errors
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}
