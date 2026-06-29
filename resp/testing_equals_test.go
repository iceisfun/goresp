package resp_test

import (
	"testing"

	"github.com/iceisfun/goresp/resp"
)

func generateRESPValues() []resp.RESPValue {
	return []resp.RESPValue{
		&resp.RESPSimpleString{Value: "SimpleString"},
		&resp.RESPError{Value: "Error"},
		&resp.RESPInteger{Value: 42},
		&resp.RESPBulkString{Value: []byte("BulkString")},
		&resp.RESPBulkString{Value: nil}, // Null BulkString
		&resp.RESPArray{Items: []resp.RESPValue{
			&resp.RESPSimpleString{Value: "ArrayItem"},
			&resp.RESPInteger{Value: 1},
		}},
		&resp.RESPArray{Items: nil}, // Null Array
	}
}

func TestRESPValueEquals(t *testing.T) {
	tests := []struct {
		name     string
		value1   resp.RESPValue
		value2   resp.RESPValue
		expected bool
	}{
		{"SimpleString Equal", &resp.RESPSimpleString{Value: "OK"}, &resp.RESPSimpleString{Value: "OK"}, true},
		{"SimpleString Not Equal", &resp.RESPSimpleString{Value: "OK"}, &resp.RESPSimpleString{Value: "NOT OK"}, false},
		{"Error Equal", &resp.RESPError{Value: "Error"}, &resp.RESPError{Value: "Error"}, true},
		{"Error Not Equal", &resp.RESPError{Value: "Error1"}, &resp.RESPError{Value: "Error2"}, false},
		{"Integer Equal", &resp.RESPInteger{Value: 42}, &resp.RESPInteger{Value: 42}, true},
		{"Integer Not Equal", &resp.RESPInteger{Value: 42}, &resp.RESPInteger{Value: 43}, false},
		{"BulkString Equal", &resp.RESPBulkString{Value: []byte("hello")}, &resp.RESPBulkString{Value: []byte("hello")}, true},
		{"BulkString Not Equal", &resp.RESPBulkString{Value: []byte("hello")}, &resp.RESPBulkString{Value: []byte("world")}, false},
		{"Null BulkString Equal", &resp.RESPBulkString{Value: nil}, &resp.RESPBulkString{Value: nil}, true},
		{"Null BulkString Not Equal", &resp.RESPBulkString{Value: nil}, &resp.RESPBulkString{Value: []byte{}}, false},
		{"Empty Array Equal", &resp.RESPArray{Items: []resp.RESPValue{}}, &resp.RESPArray{Items: []resp.RESPValue{}}, true},
		{"Null Array Equal", &resp.RESPArray{Items: nil}, &resp.RESPArray{Items: nil}, true},
		{"Array Equal",
			&resp.RESPArray{Items: []resp.RESPValue{&resp.RESPSimpleString{Value: "OK"}, &resp.RESPInteger{Value: 42}}},
			&resp.RESPArray{Items: []resp.RESPValue{&resp.RESPSimpleString{Value: "OK"}, &resp.RESPInteger{Value: 42}}},
			true,
		},
		{"Array Not Equal",
			&resp.RESPArray{Items: []resp.RESPValue{&resp.RESPSimpleString{Value: "OK"}, &resp.RESPInteger{Value: 42}}},
			&resp.RESPArray{Items: []resp.RESPValue{&resp.RESPSimpleString{Value: "OK"}, &resp.RESPInteger{Value: 43}}},
			false,
		},
		{"Different Types Not Equal", &resp.RESPSimpleString{Value: "42"}, &resp.RESPInteger{Value: 42}, false},
		{"Nested Array Equal",
			&resp.RESPArray{Items: []resp.RESPValue{
				&resp.RESPArray{Items: []resp.RESPValue{&resp.RESPSimpleString{Value: "nested"}}},
				&resp.RESPBulkString{Value: []byte("outer")},
			}},
			&resp.RESPArray{Items: []resp.RESPValue{
				&resp.RESPArray{Items: []resp.RESPValue{&resp.RESPSimpleString{Value: "nested"}}},
				&resp.RESPBulkString{Value: []byte("outer")},
			}},
			true,
		},
		{"Nested Array Not Equal",
			&resp.RESPArray{Items: []resp.RESPValue{
				&resp.RESPArray{Items: []resp.RESPValue{&resp.RESPSimpleString{Value: "nested"}}},
				&resp.RESPBulkString{Value: []byte("outer")},
			}},
			&resp.RESPArray{Items: []resp.RESPValue{
				&resp.RESPArray{Items: []resp.RESPValue{&resp.RESPSimpleString{Value: "different"}}},
				&resp.RESPBulkString{Value: []byte("outer")},
			}},
			false,
		},
		{
			"Empty arrays",
			&resp.RESPArray{Items: []resp.RESPValue{}},
			&resp.RESPArray{Items: []resp.RESPValue{}},
			true,
		},
		{
			"BulkString Not Equal",
			&resp.RESPBulkString{Value: []byte("hello")},
			&resp.RESPInteger{Value: 42},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.value1.Equal(tt.value2)
			if result != tt.expected {
				t.Errorf("Equal() = %v, want %v", result, tt.expected)
			}

			// Test symmetry
			reverseResult := tt.value2.Equal(tt.value1)
			if reverseResult != result {
				t.Errorf("Equality is not symmetric: a.Equal(b) = %v, but b.Equal(a) = %v", result, reverseResult)
			}
		})
	}
}

func TestRESPValueEqualsSelf(t *testing.T) {
	values := []resp.RESPValue{
		&resp.RESPSimpleString{Value: "OK"},
		&resp.RESPError{Value: "Error"},
		&resp.RESPInteger{Value: 42},
		&resp.RESPBulkString{Value: []byte("hello")},
		&resp.RESPBulkString{Value: nil},
		&resp.RESPArray{Items: []resp.RESPValue{&resp.RESPSimpleString{Value: "OK"}, &resp.RESPInteger{Value: 42}}},
		&resp.RESPArray{Items: nil},
	}

	for _, v := range values {
		t.Run(v.Type(), func(t *testing.T) {
			if !v.Equal(v) {
				t.Errorf("%v is not equal to itself", v)
			}
		})
	}
}

func TestRESPValueEqualsDifferentTypes(t *testing.T) {
	values := generateRESPValues()

	for i, v1 := range values {
		for j, v2 := range values {
			t.Run(v1.Type()+"_vs_"+v2.Type(), func(t *testing.T) {
				result := v1.Equal(v2)
				expected := i == j // Only equal if it's the same item

				if result != expected {
					t.Errorf("%v.Equal(%v) = %v, want %v", v1, v2, result, expected)
				}

				// Test symmetry
				reverseResult := v2.Equal(v1)
				if reverseResult != result {
					t.Errorf("Equality is not symmetric: %v.Equal(%v) = %v, but %v.Equal(%v) = %v",
						v1, v2, result, v2, v1, reverseResult)
				}
			})
		}
	}
}
