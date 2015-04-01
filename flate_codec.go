package swim

import (
	"bytes"
	"compress/flate"
	"io"
)

// The flate codec wraps around a concrete codec implementation to provde
// DEFLATE compression.
type FlateCodec struct {
	Codec // The concrete codec to use
}

// Implementation of Codec.Decode()
func (c *FlateCodec) Decode(coded *CodedMessage) error {

	// decompress
	r := flate.NewReader(bytes.NewBuffer(coded.Bytes))
	var b bytes.Buffer
	if _, err := io.Copy(&b, r); err != nil {
		return err
	}
	if err := r.Close(); err != nil {
		return err
	}
	data := b.Bytes()

	// save decompressed
	coded.Bytes = data
	coded.Size = len(data)

	// concrete decode
	return c.Codec.Decode(coded)
}

// Implementation of Codec.Encode()
func (c *FlateCodec) Encode(coded *CodedMessage) error {

	// concrete encode
	if err := c.Codec.Encode(coded); err != nil {
		return err
	}

	// compress
	var b bytes.Buffer
	w, err := flate.NewWriter(&b, flate.BestCompression)
	if err != nil {
		return err
	}
	if _, err := w.Write(coded.Bytes); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	data := b.Bytes()

	// save compressed
	coded.Bytes = data
	coded.Size = len(data)

	return nil
}
