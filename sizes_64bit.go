//go:build amd64 || arm64

package goswu

import "encoding/binary"

const (
	sizeOfSizeT        = 8
	paddingAfterDryRun = 4
	paddingAfterBool   = 7
	ipcMsgDataSize     = 3112
)

type sizeT uint64

func appendSizeT(buf []byte, v sizeT) []byte {
	return binary.NativeEndian.AppendUint64(buf, uint64(v))
}
