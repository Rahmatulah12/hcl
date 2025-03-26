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

func maskString(input string, config *MaskConfig) string {
	length := len(input)
	if length <= 1 {
		return "*"
	}

	if length >= 255 {
		return "*"
	}

	switch config.MaskType {
	case Default:
		return strings.Repeat("*", 5)
	case FullMask:
		return strings.Repeat("*", length)
	case PartialMask:
		// Ensure we don't show more characters than the string length
		showFirst := min(config.ShowFirst, length)
		showLast := min(config.ShowLast, length)

		// If showing first and last would cover most of the string, fall back to full mask
		if showFirst+showLast >= length {
			return strings.Repeat("*", length)
		}

		return input[:showFirst] +
			strings.Repeat("*", length-showFirst-showLast) +
			input[length-showLast:]
	default:
		return input
	}
}

func shouldMask(key string, configs []*MaskConfig) (bool, *MaskConfig) {
	lowerKey := strings.ToLower(key)
	for _, config := range configs {
		if lowerKey == strings.ToLower(config.Field) {
			return true, config
		}
	}
	return false, nil
}

func maskValue(value interface{}, config *MaskConfig) interface{} {
	switch v := value.(type) {
	case string:
		return maskString(v, config)
	case int, int8, int16, int32, int64:
		return "*****"
	case float32, float64:
		return "*****"
	case []interface{}:
		for i, item := range v {
			v[i] = maskValue(item, config)
		}
		return v
	case map[string][]string:
		for key, arr := range v {
			for i, item := range arr {
				arr[i] = maskString(item, config)
			}
			v[key] = arr
		}
		return v
	case map[string][]interface{}:
		for key, arr := range v {
			for i, item := range arr {
				arr[i] = maskValue(item, config)
			}
			v[key] = arr
		}
		return v
	case map[interface{}][]string:
		for key, arr := range v {
			for i, item := range arr {
				arr[i] = maskString(item, config)
			}
			v[key] = arr
		}
		return v
	case map[interface{}][]interface{}:
		for key, arr := range v {
			for i, item := range arr {
				arr[i] = maskValue(item, config)
			}
			v[key] = arr
		}
		return v
	case map[string]interface{}:
		for key, item := range v {
			v[key] = maskValue(item, config)
		}
		return v
	case map[interface{}]interface{}:
		for key, item := range v {
			v[key] = maskValue(item, config)
		}
		return v
	default:
		// Handle unknown types
		if reflect.TypeOf(v).Kind() == reflect.Map {
			mapVal := reflect.ValueOf(v)
			for _, key := range mapVal.MapKeys() {
				mapVal.SetMapIndex(key, reflect.ValueOf(maskValue(mapVal.MapIndex(key).Interface(), config)))
			}
			return mapVal.Interface()
		}
		return value
	}
}

func maskJSON(jsonStr string, configs []*MaskConfig) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return ""
	}

	maskNestedJSON(data, configs)

	return convertInterfaceToJson(data)
}

func maskNestedJSON(data map[string]interface{}, configs []*MaskConfig) {
	for key, value := range data {
		if isMasked, config := shouldMask(key, configs); isMasked {
			(data)[key] = maskValue(value, config)
		} else if nestedMap, ok := value.(map[string]interface{}); ok {
			maskNestedJSON(nestedMap, configs)
			(data)[key] = nestedMap
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
