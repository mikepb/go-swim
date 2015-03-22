package swim

import (
	"testing"
)

func TestGobCodec(t *testing.T) {
	var codec GobCodec
	testCodec(t, &codec)
}
