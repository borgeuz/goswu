//go:build 386 || arm

package goswu

import "encoding/binary"

type sizeT uint32

func appendSizeT(buf []byte, v sizeT) []byte {
	return binary.NativeEndian.AppendUint32(buf, uint32(v))
}
