package swim

import (
	"github.com/bkaradzic/go-lz4"
)

// The LZ4 codec wraps around a concrete codec implementation to provde fast
// LZ4 compression.
type LZ4Codec struct {
	Codec // The concrete codec to use
}

// Implementation of Codec.Decode()
func (c *LZ4Codec) Decode(coded *CodedMessage) error {

	// decompress
	data, err := lz4.Decode(nil, coded.Bytes)
	if err != nil {
		return err
	}

	// save decompressed
	coded.Bytes = data
	coded.Size = len(data)

	// concrete decode
	return c.Codec.Decode(coded)
}

// Implementation of Codec.Encode()
func (c *LZ4Codec) Encode(coded *CodedMessage) error {

	// concrete encode
	if err := c.Codec.Encode(coded); err != nil {
		return err
	}

	// compress
	data, err := lz4.Encode(nil, coded.Bytes)
	if err != nil {
		return err
	}

	// save compressed
	coded.Bytes = data
	coded.Size = len(data)

	return nil
}
