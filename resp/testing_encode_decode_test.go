package resp_test

import (
	"bytes"
	"reflect"
	"testing"

	resp "github.com/iceisfun/goresp/resp"
)

func TestDecodeEncodeCycle(t *testing.T) {
	for _, tt := range TestCases {
		if tt.WantsMoreData || tt.WantsErr {
			continue // Skip incomplete or error cases for this test
		}

		t.Run(tt.Name, func(t *testing.T) {
			// Test Decoding
			decoder := resp.NewDecode()
			decoder.Provide(tt.Input)
			got, err := decoder.Parse()
			if err != nil {
				t.Fatalf("Decode error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.Expected) {
				t.Errorf("Decode() = %v, want %v", got, tt.Expected)
			}

			// Test Encoding
			buf := &bytes.Buffer{}
			err = got.Encode(buf)
			if err != nil {
				t.Fatalf("Encode error: %v", err)
			}
			if !bytes.Equal(buf.Bytes(), tt.Input) {
				t.Errorf("Encode() = %v, want %v", buf.Bytes(), tt.Input)
			}

			// Test Decode -> Encode -> Decode cycle
			decoder.Reset()
			decoder.Provide(buf.Bytes())
			gotAgain, err := decoder.Parse()
			if err != nil {
				t.Fatalf("Second Decode error: %v", err)
			}
			if !reflect.DeepEqual(gotAgain, tt.Expected) {
				t.Errorf("Second Decode() = %v, want %v", gotAgain, tt.Expected)
			}
		})
	}
}

func TestIncompleteInput(t *testing.T) {
	for _, tt := range TestCases {
		if !tt.WantsMoreData {
			continue // Skip complete cases for this test
		}

		t.Run(tt.Name, func(t *testing.T) {
			decoder := resp.NewDecode()
			decoder.Provide(tt.Input)
			got, err := decoder.Parse()
			if err != nil {
				t.Errorf("Expected nil error for incomplete input, got %v", err)
			}
			if got != nil {
				t.Errorf("Expected nil result for incomplete input, got %v", got)
			}
		})
	}
}

func TestInvalidInput(t *testing.T) {
	for _, tt := range TestCases {
		if !tt.WantsErr {
			continue // Skip valid cases for this test
		}

		t.Run(tt.Name, func(t *testing.T) {
			decoder := resp.NewDecode()
			decoder.Provide(tt.Input)
			_, err := decoder.Parse()
			if err == nil {
				t.Errorf("Expected error for invalid input, got nil")
			}
		})
	}
}

func TestStreamDecode(t *testing.T) {
	for _, tt := range TestCases {
		t.Run(tt.Name, func(t *testing.T) {
			decoder := resp.NewDecode()
			var finalGot resp.RESPValue
			var finalErr error

			for i, b := range tt.Input {
				decoder.Provide([]byte{b})
				got, err := decoder.Parse()

				if err != nil {
					if !tt.WantsErr {
						t.Errorf("Unexpected error at byte %d: %v", i, err)
					}
					finalErr = err
					break
				}

				if got != nil {
					if i != len(tt.Input)-1 {
						t.Errorf("Got unexpected value at byte %d: %v", i, got)
					}
					finalGot = got
					break
				}
			}

			// Check final state
			if tt.WantsErr {
				if finalErr == nil {
					t.Errorf("Expected error, got nil")
				}
			} else if tt.WantsMoreData {
				if finalGot != nil || finalErr != nil {
					t.Errorf("Expected (nil, nil) for incomplete data, got (%v, %v)", finalGot, finalErr)
				}
			} else {
				if finalErr != nil {
					t.Errorf("Unexpected final error: %v", finalErr)
				}
				if !reflect.DeepEqual(finalGot, tt.Expected) {
					t.Errorf("StreamDecode() = %v, want %v", finalGot, tt.Expected)
				}
			}
		})
	}
}
