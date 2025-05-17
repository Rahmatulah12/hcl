package hcl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPointer(t *testing.T) {
	t.Run("pointer value", func(t *testing.T) {
		var x int = 5
		ptr := &x
		assert.True(t, isPointer(ptr))
	})

	t.Run("non-pointer value", func(t *testing.T) {
		var x int = 5
		assert.False(t, isPointer(x))
	})
}

func TestConvertInterfaceToJson(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		result := convertInterfaceToJson(nil)
		assert.Equal(t, "", result)
	})

	t.Run("simple struct", func(t *testing.T) {
		type testStruct struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		data := testStruct{Name: "John", Age: 30}
		result := convertInterfaceToJson(data)
		assert.Equal(t, `{"name":"John","age":30}`, result)
	})

	t.Run("map", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "John",
			"age":  30,
		}
		result := convertInterfaceToJson(data)
		assert.Equal(t, `{"age":30,"name":"John"}`, result)
	})

	t.Run("unmarshallable value", func(t *testing.T) {
		// Create a circular reference that can't be marshaled
		type circular struct {
			Self *circular
		}
		c := &circular{}
		c.Self = c

		result := convertInterfaceToJson(c)
		assert.Equal(t, "", result)
	})
}

func TestMaskString(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		config := &MaskConfig{MaskType: Default}
		result := maskString("", config)
		assert.Equal(t, "*", result)
	})

	t.Run("single character", func(t *testing.T) {
		config := &MaskConfig{MaskType: Default}
		result := maskString("a", config)
		assert.Equal(t, "*", result)
	})

	t.Run("very long string", func(t *testing.T) {
		config := &MaskConfig{MaskType: Default}
		longStr := strings.Repeat("a", 300)
		result := maskString(longStr, config)
		assert.Equal(t, "*", result)
	})

	t.Run("default mask", func(t *testing.T) {
		config := &MaskConfig{MaskType: Default}
		result := maskString("password", config)
		assert.Equal(t, "*****", result)
	})

	t.Run("full mask", func(t *testing.T) {
		config := &MaskConfig{MaskType: FullMask}
		result := maskString("password", config)
		assert.Equal(t, "********", result)
	})

	t.Run("partial mask - show first and last", func(t *testing.T) {
		config := &MaskConfig{
			MaskType:  PartialMask,
			ShowFirst: 2,
			ShowLast:  2,
		}
		result := maskString("password", config)
		assert.Equal(t, "pa****rd", result)
	})

	t.Run("partial mask - show first only", func(t *testing.T) {
		config := &MaskConfig{
			MaskType:  PartialMask,
			ShowFirst: 2,
			ShowLast:  0,
		}
		result := maskString("password", config)
		assert.Equal(t, "pa******", result)
	})

	t.Run("partial mask - show last only", func(t *testing.T) {
		config := &MaskConfig{
			MaskType:  PartialMask,
			ShowFirst: 0,
			ShowLast:  2,
		}
		result := maskString("password", config)
		assert.Equal(t, "******rd", result)
	})

	t.Run("partial mask - show too many characters", func(t *testing.T) {
		config := &MaskConfig{
			MaskType:  PartialMask,
			ShowFirst: 4,
			ShowLast:  4,
		}
		result := maskString("password", config)
		assert.Equal(t, "********", result)
	})
}

func TestShouldMask(t *testing.T) {
	configs := []*MaskConfig{
		{Field: "password", MaskType: FullMask},
		{Field: "credit_card", MaskType: PartialMask, ShowFirst: 4, ShowLast: 4},
	}

	t.Run("should mask - exact match", func(t *testing.T) {
		shouldMask, config := shouldMask("password", configs)
		assert.True(t, shouldMask)
		assert.Equal(t, FullMask, config.MaskType)
	})

	t.Run("should mask - case insensitive", func(t *testing.T) {
		shouldMask, config := shouldMask("PASSWORD", configs)
		assert.True(t, shouldMask)
		assert.Equal(t, FullMask, config.MaskType)
	})

	t.Run("should not mask", func(t *testing.T) {
		shouldMask, config := shouldMask("username", configs)
		assert.False(t, shouldMask)
		assert.Nil(t, config)
	})
}

func TestMaskValue(t *testing.T) {
	config := &MaskConfig{MaskType: FullMask}

	t.Run("string value", func(t *testing.T) {
		result := maskValue("secret", config)
		assert.Equal(t, "******", result)
	})

	t.Run("integer value", func(t *testing.T) {
		result := maskValue(12345, config)
		assert.Equal(t, "*****", result)
	})

	t.Run("float value", func(t *testing.T) {
		result := maskValue(123.45, config)
		assert.Equal(t, "*****", result)
	})

	t.Run("slice of interface", func(t *testing.T) {
		data := []interface{}{"secret1", "secret2"}
		result := maskValue(data, config).([]interface{})
		assert.Equal(t, "*******", result[0])
		assert.Equal(t, "*******", result[1])
	})

	t.Run("map string to string slice", func(t *testing.T) {
		data := map[string][]string{
			"passwords": {"secret1", "secret2"},
		}
		result := maskValue(data, config).(map[string][]string)
		assert.Equal(t, "*******", result["passwords"][0])
		assert.Equal(t, "*******", result["passwords"][1])
	})

	t.Run("map string to interface", func(t *testing.T) {
		data := map[string]interface{}{
			"password": "secret",
			"age":      30,
		}
		result := maskValue(data, config).(map[string]interface{})
		assert.Equal(t, "******", result["password"])
		assert.Equal(t, "*****", result["age"])
	})
}

func TestMaskJSON(t *testing.T) {
	configs := []*MaskConfig{
		{Field: "password", MaskType: FullMask},
		{Field: "credit_card", MaskType: PartialMask, ShowFirst: 4, ShowLast: 4},
	}

	t.Run("invalid json", func(t *testing.T) {
		result := maskJSON("invalid json", configs)
		assert.Equal(t, "", result)
	})

	t.Run("simple json", func(t *testing.T) {
		jsonStr := `{"username":"john","password":"secret123"}`
		result := maskJSON(jsonStr, configs)
		assert.Contains(t, result, `"username":"john"`)
		assert.Contains(t, result, `"password":"*********"`)
	})

	t.Run("nested json", func(t *testing.T) {
		jsonStr := `{"user":{"username":"john","password":"secret123"},"payment":{"credit_card":"1234567890123456"}}`
		result := maskJSON(jsonStr, configs)
		assert.Contains(t, result, `"username":"john"`)
		assert.Contains(t, result, `"password":"*********"`)
		assert.Contains(t, result, `"credit_card":"1234********3456"`)
	})
}

func TestInArray(t *testing.T) {
	t.Run("string in array", func(t *testing.T) {
		arr := []string{"apple", "banana", "orange"}
		assert.True(t, inArray("banana", arr))
		assert.False(t, inArray("grape", arr))
	})

	t.Run("int in array", func(t *testing.T) {
		arr := []int{1, 2, 3, 4, 5}
		assert.True(t, inArray(3, arr))
		assert.False(t, inArray(6, arr))
	})

	t.Run("int64 in array", func(t *testing.T) {
		arr := []int64{1, 2, 3, 4, 5}
		assert.True(t, inArray(int64(3), arr))
		assert.False(t, inArray(int64(6), arr))
	})

	t.Run("unsupported type", func(t *testing.T) {
		arr := []float64{1.1, 2.2, 3.3}
		assert.False(t, inArray(3.3, arr))
	})
}
