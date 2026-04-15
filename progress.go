package goswu

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ProgressMsg maps the C struct progress_msg that SWUpdate sends on the
// progress socket. The struct is packed (no padding between fields).
type ProgressMsg struct {
	ApiVersion uint32
	Status     Status
	DWLPercent uint32
	DWLBytes   uint64
	NSteps     uint32
	CurStep    uint32
	CurPercent uint32
	CurImage   [256]byte
	HndName    [64]byte
	Source     uint32
	InfoLen    uint32
	Info       [2048]byte
}

// unmarshal reads exactly 2408 bytes from r and decodes them into the struct.
func (m *ProgressMsg) unmarshal(r io.Reader) error {
	var msgBuf [2408]byte
	if _, err := io.ReadFull(r, msgBuf[:]); err != nil {
		return fmt.Errorf("goswu: reading progress message: %w", err)
	}
	m.ApiVersion = binary.NativeEndian.Uint32(msgBuf[:4])
	m.Status = Status(binary.NativeEndian.Uint32(msgBuf[4:8]))
	m.DWLPercent = binary.NativeEndian.Uint32(msgBuf[8:12])
	m.DWLBytes = binary.NativeEndian.Uint64(msgBuf[12:20])
	m.NSteps = binary.NativeEndian.Uint32(msgBuf[20:24])
	m.CurStep = binary.NativeEndian.Uint32(msgBuf[24:28])
	m.CurPercent = binary.NativeEndian.Uint32(msgBuf[28:32])
	copy(m.CurImage[:], msgBuf[32:288])
	copy(m.HndName[:], msgBuf[288:352])
	m.Source = binary.NativeEndian.Uint32(msgBuf[352:356])
	m.InfoLen = binary.NativeEndian.Uint32(msgBuf[356:360])
	copy(m.Info[:], msgBuf[360:2408])
	return nil
}
