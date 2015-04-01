package swim

import (
	"testing"
)

func TestFlateCodec(t *testing.T) {
	testCodec(t, &FlateCodec{new(GobCodec)})
}
