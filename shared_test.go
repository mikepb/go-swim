package swim

import (
	"encoding"
	"testing"
)

func testEqualBytes(t *testing.T, subject, expect []byte) {
	for i, b := range expect {
		if subject[i] != b {
			t.Fatalf("expected %v got %v", expect, subject)
		}
	}
}

func testMarshalBinary(t *testing.T, that encoding.BinaryMarshaler, expect []byte) {
	subject, err := that.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	testEqualBytes(t, subject, expect)
}

func testUnmarshalBinary(t *testing.T, that encoding.BinaryUnmarshaler, data []byte) {
	if err := that.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
}
