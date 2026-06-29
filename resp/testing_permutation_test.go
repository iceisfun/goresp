package resp_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/iceisfun/goresp/resp"
)

// generatePermutations generates all permutations of the given resp.RESP values
func generatePermutations(values []resp.RESPValue) [][]resp.RESPValue {
	var result [][]resp.RESPValue
	var generate func([]resp.RESPValue, []resp.RESPValue)

	generate = func(current []resp.RESPValue, remaining []resp.RESPValue) {
		if len(remaining) == 0 {
			result = append(result, current)
			return
		}
		for i, v := range remaining {
			newCurrent := append([]resp.RESPValue{}, current...)
			newCurrent = append(newCurrent, v)
			newRemaining := append([]resp.RESPValue{}, remaining[:i]...)
			newRemaining = append(newRemaining, remaining[i+1:]...)
			generate(newCurrent, newRemaining)
		}
	}

	generate([]resp.RESPValue{}, values)
	return result
}

func TestPermutations(t *testing.T) {
	basicValues := []resp.RESPValue{
		&resp.RESPSimpleString{Value: "OK"},
		&resp.RESPError{Value: "Error"},
		&resp.RESPInteger{Value: 42},
		&resp.RESPBulkString{Value: []byte("hello")},
		&resp.RESPArray{Items: []resp.RESPValue{
			&resp.RESPSimpleString{Value: "nested"},
			&resp.RESPInteger{Value: 100},
		}},
	}

	// Generate permutations
	permutations := generatePermutations(basicValues)

	for _, perm := range permutations {
		t.Run("Permutation", func(t *testing.T) {
			// Encode all values in the permutation
			buf := &bytes.Buffer{}
			for _, v := range perm {
				err := v.Encode(buf)
				if err != nil {
					t.Fatalf("Encode error: %v", err)
				}
			}

			// Decode the encoded data
			decoder := resp.NewDecode()
			decoder.Provide(buf.Bytes())

			var decoded []resp.RESPValue
			for {
				value, err := decoder.Parse()
				if err != nil {
					t.Fatalf("Decode error: %v", err)
				}
				if value == nil {
					break
				}
				decoded = append(decoded, value)
			}

			// Compare original and decoded values
			if !reflect.DeepEqual(perm, decoded) {
				t.Errorf("Decoded values don't match original. Got %v, want %v", decoded, perm)
			}
		})
	}
}
