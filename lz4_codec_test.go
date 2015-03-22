package swim

import (
	"testing"
)

func TestLZ4Codec(t *testing.T) {
	testCodec(t, &LZ4Codec{new(GobCodec)})
}
