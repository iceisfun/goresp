package resp_test

import "github.com/iceisfun/goresp/resp"

type TestCase struct {
	Name          string
	Input         []byte
	Expected      resp.RESPValue
	WantsMoreData bool
	WantsErr      bool
}

var TestCases = []TestCase{
	{
		Name:     "Simple String",
		Input:    []byte("+OK\r\n"),
		Expected: &resp.RESPSimpleString{Value: "OK"},
	},
	{
		Name:     "Error",
		Input:    []byte("-Error message\r\n"),
		Expected: &resp.RESPError{Value: "Error message"},
	},
	{
		Name:     "Integer",
		Input:    []byte(":1000\r\n"),
		Expected: &resp.RESPInteger{Value: 1000},
	},
	{
		Name:     "Bulk String",
		Input:    []byte("$5\r\nhello\r\n"),
		Expected: &resp.RESPBulkString{Value: []byte("hello")},
	},
	{
		Name:     "Empty Bulk String",
		Input:    []byte("$0\r\n\r\n"),
		Expected: &resp.RESPBulkString{Value: []byte{}},
	},
	{
		Name:     "Null Bulk String",
		Input:    []byte("$-1\r\n"),
		Expected: &resp.RESPBulkString{Value: nil},
	},
	{
		Name:  "Array",
		Input: []byte("*3\r\n+OK\r\n:1000\r\n$5\r\nhello\r\n"),
		Expected: &resp.RESPArray{
			Items: []resp.RESPValue{
				&resp.RESPSimpleString{Value: "OK"},
				&resp.RESPInteger{Value: 1000},
				&resp.RESPBulkString{Value: []byte("hello")},
			},
		},
	},
	{
		Name:     "Empty Array",
		Input:    []byte("*0\r\n"),
		Expected: &resp.RESPArray{Items: []resp.RESPValue{}},
	},
	{
		Name:     "Null Array",
		Input:    []byte("*-1\r\n"),
		Expected: &resp.RESPArray{Items: nil},
	},
	{
		Name:  "Nested Array",
		Input: []byte("*2\r\n*2\r\n+nested\r\n:42\r\n$5\r\nouter\r\n"),
		Expected: &resp.RESPArray{
			Items: []resp.RESPValue{
				&resp.RESPArray{
					Items: []resp.RESPValue{
						&resp.RESPSimpleString{Value: "nested"},
						&resp.RESPInteger{Value: 42},
					},
				},
				&resp.RESPBulkString{Value: []byte("outer")},
			},
		},
	},
	{
		Name:          "Incomplete Simple String",
		Input:         []byte("+OK"),
		WantsMoreData: true,
	},
	{
		Name:          "Incomplete Bulk String",
		Input:         []byte("$5\r\nhel"),
		WantsMoreData: true,
	},
	{
		Name:          "Incomplete Array",
		Input:         []byte("*2\r\n+OK\r\n"),
		WantsMoreData: true,
	},
	{
		Name:     "Invalid Op Code",
		Input:    []byte("xInvalid\r\n"),
		WantsErr: true,
	},
	{
		Name:     "Invalid Integer",
		Input:    []byte(":abc\r\n"),
		WantsErr: true,
	},
}
