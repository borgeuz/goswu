//go:build amd64 || arm64

package goswu

import "encoding/binary"

const (
	ipcMsgDataSize int = 3112
)

type sizeT uint64

func appendSizeT(buf []byte, v sizeT) []byte {
	return binary.NativeEndian.AppendUint64(buf, uint64(v))
}
