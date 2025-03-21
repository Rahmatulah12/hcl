package hcl

import (
	"encoding/json"
	"net"
	"reflect"
	"strings"
)

func isPointer(v interface{}) bool {
	return reflect.TypeOf(v).Kind() == reflect.Ptr
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

func maskString(input string) string {
	length := len(input)
	return strings.Repeat("*", length)
}

func shouldMask(key string, maskFields []string) bool {
	lowerKey := strings.ToLower(key)
	for _, field := range maskFields {
		if lowerKey == strings.ToLower(field) {
			return true
		}
	}
	return false
}

func maskValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return maskString(v)
	case int, int8, int16, int32, int64:
		return "*****"
	case float32, float64:
		return "*****"
	case []interface{}:
		for i, item := range v {
			v[i] = maskValue(item)
		}
		return v
	case map[string][]string:
		for key, arr := range v {
			for i, item := range arr {
				arr[i] = maskString(item)
			}
			v[key] = arr
		}
		return v
	case map[string][]interface{}:
		for key, arr := range v {
			for i, item := range arr {
				arr[i] = maskValue(item)
			}
			v[key] = arr
		}
		return v
	case map[interface{}][]string:
		for key, arr := range v {
			for i, item := range arr {
				arr[i] = maskString(item)
			}
			v[key] = arr
		}
		return v
	case map[interface{}][]interface{}:
		for key, arr := range v {
			for i, item := range arr {
				arr[i] = maskValue(item)
			}
			v[key] = arr
		}
		return v
	case map[string]interface{}:
		for key, item := range v {
			v[key] = maskValue(item)
		}
		return v
	case map[interface{}]interface{}:
		for key, item := range v {
			v[key] = maskValue(item)
		}
		return v
	default:
		// Handle unknown types
		if reflect.TypeOf(v).Kind() == reflect.Map {
			mapVal := reflect.ValueOf(v)
			for _, key := range mapVal.MapKeys() {
				mapVal.SetMapIndex(key, reflect.ValueOf(maskValue(mapVal.MapIndex(key).Interface())))
			}
			return mapVal.Interface()
		}
		return value
	}
}

func maskJSON(jsonStr string, maskFields []string) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return ""
	}

	maskNestedJSON(&data, maskFields)

	return convertInterfaceToJson(data)
}

func maskNestedJSON(data *map[string]interface{}, maskFields []string) {
	for key, value := range *data {
		if shouldMask(key, maskFields) {
			(*data)[key] = maskValue(value)
		} else if nestedMap, ok := value.(map[string]interface{}); ok {
			maskNestedJSON(&nestedMap, maskFields)
			(*data)[key] = nestedMap
		}
	}
}

func parseHostPort(addr string) (host, port string) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", ""
	}
	return host, port
}
