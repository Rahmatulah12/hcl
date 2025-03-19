package hcl

import (
	"encoding/json"
	"reflect"
)

func isPointer(v interface{}) bool {
	return reflect.TypeOf(v).Kind() == reflect.Ptr
}

func ToPointer[T any](l T) *T {
	return &l
}

func convertInterfaceToJson(data interface{}) string {
	if data == nil {
		return ""
	}

	a, err := json.Marshal(data)

	if err != nil {
		return ""
	}

	return string(a)
}
