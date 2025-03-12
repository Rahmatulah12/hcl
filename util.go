package hcl

import "reflect"

func isPointer(v interface{}) bool {
	return reflect.TypeOf(v).Kind() == reflect.Ptr
}

func ToPointer[T any](l T) *T {
	return &l
}
