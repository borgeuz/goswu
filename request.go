package goswu

import (
	"encoding/binary"
	"strings"
	"unsafe"
)

// Fixed-size field lengths matching the SWUpdate C struct swupdate_request.
const (
	infoFieldSize        = 512
	softwareSetFieldSize = 256
	runningModeFieldSize = 256
)

// Request holds the parameters for an installation request.
// Fields map to the C struct swupdate_request in network_ipc.h.
type Request struct {
	APIVersion      uint32
	Source          SourceType
	DryRun          RunType
	Len             sizeT
	Info            string
	SoftwareSet     string
	RunningMode     string
	DisableStoreSWU bool
}

// Marshal serializes the request into the binary layout expected by SWUpdate.
//
// Wire layout (little-endian on most targets):
//
//	apiversion      uint32     (4 bytes)
//	source          int32      (4 bytes)
//	cmd             int32      (4 bytes)
//	cmdlen          sizeT      (4 bytes on 32-bit, 8 bytes on 64-bit)
//	info            char[512]  (512 bytes, zero-padded)
//	software_set    char[256]  (256 bytes, zero-padded)
//	running_mode    char[256]  (256 bytes, zero-padded)
//	disable_store   uint32     (4 bytes)
//	                           ──────────
//	total                      1044 bytes on 32-bit, 1056 bytes on 64-bit
func (r *Request) marshal() []byte {
	// 1040 bytes for the fixed fields, plus the size of sizeT
	buf := make([]byte, 0, 1040+int(unsafe.Sizeof(sizeT(0))))
	buf = binary.NativeEndian.AppendUint32(buf, r.APIVersion)
	buf = binary.NativeEndian.AppendUint32(buf, uint32(r.Source))
	buf = binary.NativeEndian.AppendUint32(buf, uint32(r.DryRun))
	buf = appendSizeT(buf, r.Len)
	buf = append(buf, fixedString(r.Info, infoFieldSize)...)
	buf = append(buf, fixedString(r.SoftwareSet, softwareSetFieldSize)...)
	buf = append(buf, fixedString(r.RunningMode, runningModeFieldSize)...)
	if r.DisableStoreSWU {
		buf = binary.NativeEndian.AppendUint32(buf, 1)
	} else {
		buf = binary.NativeEndian.AppendUint32(buf, 0)
	}
	return buf
}

// fixedString writes s into a zero-filled buffer of the given size,
// truncating if s is longer than size.
func fixedString(s string, size int) []byte {
	b := make([]byte, size)
	copy(b, s)
	return b
}

// Selection identifies which software set and running mode to install.
type Selection struct {
	SoftwareSet string
	RunningMode string
}

// ParseSelection parses a comma-separated "software_set,running_mode" string.
// Returns nil if the input is empty or malformed.
func ParseSelection(s string) *Selection {
	if s == "" {
		return nil
	}
	parts := strings.SplitN(strings.TrimSpace(s), ",", 2)
	if len(parts) != 2 {
		return nil
	}
	return &Selection{
		SoftwareSet: parts[0],
		RunningMode: parts[1],
	}
}
