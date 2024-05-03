// SPDX-License-Identifier: MIT
//
// Copyright (c) 2024 Berachain Foundation
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following
// conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package ssz_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/berachain/beacon-kit/mod/primitives/pkg/math"
	"github.com/berachain/beacon-kit/mod/primitives/pkg/ssz"
	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshalU256(t *testing.T) {
	original := math.U256L{
		0x01,
		0x02,
		0x03,
		0x04,
		0x05,
		0x06,
		0x07,
		0x08,
		0x09,
		0x0A,
		0x0B,
		0x0C,
		0x0D,
		0x0E,
		0x0F,
		0x10,
		0x11,
		0x12,
		0x13,
		0x14,
		0x15,
		0x16,
		0x17,
		0x18,
		0x19,
		0x1A,
		0x1B,
		0x1C,
		0x1D,
		0x1E,
		0x1F,
		0x20,
	}
	marshaled := ssz.MarshalU256(original)
	unmarshaled := ssz.UnmarshalU256L[[32]byte](marshaled)
	require.Equal(t, marshaled, unmarshaled[:], "Marshal/Unmarshal U256 failed")
}

func TestMarshalUnmarshalU128(t *testing.T) {
	original := [16]byte{
		0x01,
		0x02,
		0x03,
		0x04,
		0x05,
		0x06,
		0x07,
		0x08,
		0x09,
		0x0A,
		0x0B,
		0x0C,
		0x0D,
		0x0E,
		0x0F,
		0x10,
	}
	marshaled := ssz.MarshalU128(original)
	unmarshaled := ssz.UnmarshalU128L[[16]byte](marshaled)
	require.Equal(t, marshaled, unmarshaled[:], "Marshal/Unmarshal U128 failed")
}

func TestMarshalUnmarshalU64(t *testing.T) {
	original := uint64(0x0102030405060708)
	marshaled := ssz.MarshalU64(original)
	unmarshaled := ssz.UnmarshalU64[uint64](marshaled)
	require.Equal(t, original, unmarshaled, "Marshal/Unmarshal U64 failed")
}

func TestMarshalUnmarshalU32(t *testing.T) {
	original := uint32(0x01020304)
	marshaled := ssz.MarshalU32[uint32](original)
	unmarshaled := ssz.UnmarshalU32[uint32](marshaled)
	require.Equal(t, original, unmarshaled, "Marshal/Unmarshal U32 failed")
}

func TestMarshalUnmarshalU16(t *testing.T) {
	original := uint16(0x0102)
	marshaled := ssz.MarshalU16[uint16](original)
	unmarshaled := ssz.UnmarshalU16[uint16](marshaled)
	require.Equal(t, original, unmarshaled, "Marshal/Unmarshal U16 failed")
}

func TestMarshalUnmarshalU8(t *testing.T) {
	original := uint8(0x01)
	marshaled := ssz.MarshalU8(original)
	unmarshaled := ssz.UnmarshalU8[uint8](marshaled)
	require.Equal(t, original, unmarshaled, "Marshal/Unmarshal U8 failed")
}

func TestMarshalUnmarshalBool(t *testing.T) {
	original := true
	marshaled := ssz.MarshalBool(original)
	unmarshaled := ssz.UnmarshalBool[bool](marshaled)
	require.Equal(t, original, unmarshaled, "Marshal/Unmarshal Bool failed")
}

func FuzzMarshalUnmarshalU256(f *testing.F) {
	f.Fuzz(func(t *testing.T, byte1 byte, byte2 byte, byte3 byte, byte4 byte,
		byte5 byte, byte6 byte, byte7 byte, byte8 byte, byte9 byte, byte10 byte,
		byte11 byte, byte12 byte, byte13 byte, byte14 byte, byte15 byte,
		byte16 byte, byte17 byte, byte18 byte, byte19 byte, byte20 byte,
		byte21 byte, byte22 byte, byte23 byte, byte24 byte, byte25 byte,
		byte26 byte, byte27 byte, byte28 byte, byte29 byte, byte30 byte,
		byte31 byte, byte32 byte) {
		original := [32]byte{
			byte1,
			byte2,
			byte3,
			byte4,
			byte5,
			byte6,
			byte7,
			byte8,
			byte9,
			byte10,
			byte11,
			byte12,
			byte13,
			byte14,
			byte15,
			byte16,
			byte17,
			byte18,
			byte19,
			byte20,
			byte21,
			byte22,
			byte23,
			byte24,
			byte25,
			byte26,
			byte27,
			byte28,
			byte29,
			byte30,
			byte31,
			byte32,
		}

		marshaled := ssz.MarshalU256(original)
		unmarshaled := ssz.UnmarshalU256L[[32]byte](marshaled)
		require.Equal(t, original, unmarshaled, "Marshal/Unmarshal U256 failed")
	})
}

func FuzzMarshalUnmarshalU128(f *testing.F) {
	f.Fuzz(func(t *testing.T, byte1 byte, byte2 byte, byte3 byte, byte4 byte,
		byte5 byte, byte6 byte, byte7 byte, byte8 byte, byte9 byte,
		byte10 byte, byte11 byte, byte12 byte, byte13 byte, byte14 byte,
		byte15 byte, byte16 byte) {
		original := [16]byte{
			byte1,
			byte2,
			byte3,
			byte4,
			byte5,
			byte6,
			byte7,
			byte8,
			byte9,
			byte10,
			byte11,
			byte12,
			byte13,
			byte14,
			byte15,
			byte16,
		}

		marshaled := ssz.MarshalU128(original)
		unmarshaled := ssz.UnmarshalU128L[[16]byte](marshaled)
		require.Equal(t, original, unmarshaled, "Marshal/Unmarshal U128L failed")
	})
}

func FuzzMarshalUnmarshalU64(f *testing.F) {
	f.Fuzz(func(t *testing.T, original uint64) {
		marshaled := ssz.MarshalU64(original)
		unmarshaled := ssz.UnmarshalU64[uint64](marshaled)
		require.Equal(t, original, unmarshaled, "Marshal/Unmarshal U64 failed")
	})
}

func FuzzMarshalUnmarshalU32(f *testing.F) {
	f.Fuzz(func(t *testing.T, original uint32) {
		marshaled := ssz.MarshalU32[uint32](original)
		unmarshaled := ssz.UnmarshalU32[uint32](marshaled)
		require.Equal(t, original, unmarshaled, "Marshal/Unmarshal U32 failed")
	})
}

func FuzzMarshalUnmarshalU16(f *testing.F) {
	f.Fuzz(func(t *testing.T, original uint16) {
		marshaled := ssz.MarshalU16[uint16](original)
		unmarshaled := ssz.UnmarshalU16[uint16](marshaled)
		require.Equal(t, original, unmarshaled, "Marshal/Unmarshal U16 failed")
	})
}

func FuzzMarshalUnmarshalU8(f *testing.F) {
	f.Fuzz(func(t *testing.T, original uint8) {
		marshaled := ssz.MarshalU8(original)
		unmarshaled := ssz.UnmarshalU8[uint8](marshaled)
		require.Equal(t, original, unmarshaled, "Marshal/Unmarshal U8 failed")
	})
}

func FuzzMarshalUnmarshalBool(f *testing.F) {
	f.Fuzz(func(t *testing.T, original bool) {
		marshaled := ssz.MarshalBool(original)
		unmarshaled := ssz.UnmarshalBool[bool](marshaled)
		require.Equal(t, original, unmarshaled, "Marshal/Unmarshal Bool failed")
	})
}

func TestMarshalBitVector(t *testing.T) {
	var tests = []struct {
		name   string
		bv     []bool
		expect []byte
	}{
		{
			"empty bitvector",
			[]bool{},
			[]byte{},
		},
		{
			"single true value",
			[]bool{true},
			[]byte{1},
		},
		{
			"single false value",
			[]bool{false},
			[]byte{0},
		},
		{
			"multiple values with true at end",
			[]bool{false, false, true, false, false, false, true, true},
			[]byte{0b11000100},
		},
		{
			"multiple values with false at end",
			[]bool{true, true, false, true, true, false, false, false},
			[]byte{0b00011011},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ssz.MarshalBitVector(tt.bv)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf(
					"MarshalBitVector(%v) = %08b; expect %08b",
					tt.bv,
					got,
					tt.expect,
				)
			}
		})
	}
}

func TestMarshalBitList(t *testing.T) {
	// Create a slice of booleans to pass as input
	input := []bool{true, false, true, false, true, false, true}

	output := ssz.MarshalBitList(input)
	// Create a byte slice from a list of binary literals. 0b11010101 is the
	// binary representation of the input slice
	expectedOutput := []byte{0b11010101}
	if !reflect.DeepEqual(output, expectedOutput) {
		t.Errorf("Expected output %08b, got %08b", expectedOutput, output)
	}

	// TODO: test multiple bytes
}

func TestMostSignificantBitIndex(t *testing.T) {
	var tests = []struct {
		name     string
		original byte
		result   int
	}{
		{"0", byte('\x00'), -1},
		{"1", byte('\x01'), 0},
		{"2", byte('\x02'), 1},
		{"4", byte('\x04'), 2},
		{"8", byte('\x08'), 3},
		{"16", byte('\x10'), 4},
		{"32", byte('\x20'), 5},
		{"64", byte('\x40'), 6},
		{"128", byte('\x80'), 7},
		{"255", byte('\xFF'), 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ssz.MostSignificantBitIndex(tt.original)
			require.Equal(t, tt.result, result)
		})
	}
}

func FuzzMostSignificantBitIndex(f *testing.F) {
	f.Fuzz(func(t *testing.T, original byte) {
		result := ssz.MostSignificantBitIndex(original)

		// Basic bounds checking
		require.GreaterOrEqual(t, result, -1)
		require.LessOrEqual(t, result, 7)

		// Check each index edge for violations of spec
		switch {
		case int(original) == 0:
			require.Equal(t, -1, result)
		case int(original) < 2:
			require.Equal(t, 0, result)
		case int(original) < 4:
			require.Equal(t, 1, result)
		case int(original) < 8:
			require.Equal(t, 2, result)
		case int(original) < 16:
			require.Equal(t, 3, result)
		case int(original) < 32:
			require.Equal(t, 4, result)
		case int(original) < 64:
			require.Equal(t, 5, result)
		case int(original) < 128:
			require.Equal(t, 6, result)
		default:
			require.Equal(t, 7, result)
		}
	})
}

func BenchmarkMostSignificantBitIndex(b *testing.B) {
	var table = []struct {
		input byte
	}{
		{input: 0},
		{input: 1},
		{input: 2},
		{input: 4},
		{input: 8},
		{input: 16},
		{input: 32},
		{input: 64},
		{input: 128},
		{input: 255},
	}

	for _, v := range table {
		b.Run(fmt.Sprintf("input_size_%d", v.input), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ssz.MostSignificantBitIndex(v.input)
			}
		})
	}
}

func TestUnmarshalBitList(t *testing.T) {
	// Test case 1: Empty input
	var bv []byte
	var expected []bool
	actual := ssz.UnmarshalBitList(bv)
	if !reflect.DeepEqual(len(actual), len(expected)) {
		t.Errorf(
			"TestUnmarshalBitList failed for empty input: expected %v but got %v",
			expected,
			actual,
		)
	}

	// Test case 2: Input with sentinel bit set
	bv = []byte{0b00000011}
	expected = []bool{true}
	actual = ssz.UnmarshalBitList(bv)
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf(
			"TestUnmarshalBitList failed for input with"+
				" sentinel bit set: expected %v but got %v",
			expected,
			actual,
		)
	}

	// Test case 3: Input with multiple bits set
	bv = []byte{0b11001100}
	actual = ssz.UnmarshalBitList(bv)
	expected = []bool{false, false, true, true, false, false, true}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf(
			"TestUnmarshalBitList failed for input with"+
				" multiple bits set: expected %v but got %v",
			expected,
			actual,
		)
	}
	// Test case 3a: Check Unmarshal returns same results as original input to
	// marshal
	expectedBV := ssz.MarshalBitList(actual)
	if !reflect.DeepEqual(expectedBV, bv) {
		t.Errorf(
			"TestUnmarshalBitList failed for input with"+
				" multiple bits set: expected %08b but got %08b",
			expectedBV,
			bv,
		)
	}

	// Test case 4: Input with multiple bits set
	input := []bool{true, false, true, false, true, false, true}
	output := ssz.MarshalBitList(input)
	unmarshalledOutput := ssz.UnmarshalBitList(output)
	if !reflect.DeepEqual(input, unmarshalledOutput) {
		t.Errorf(
			"Expected output %08t, got %08t from %08b",
			unmarshalledOutput,
			input,
			output,
		)
	}

	// TODO: test multiple bytes
}
