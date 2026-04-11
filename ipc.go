package goswu

import (
	"encoding/binary"
	"fmt"
	"io"
)

// msgType represents the IPC command types used in the SWUpdate protocol.
type msgType int32

const (
	msgReqInstall msgType = iota // Request a new installation.
	msgACK                       // Positive acknowledgement.
	msgNACK                      // Negative acknowledgement.
	msgGetStatus                 // Query the daemon status.
	msgPostUpdate                // Post-update notification.
	msgSubprocess                // SWUpdate subprocess command.
	msgSetAESKey                 // Set an AES decryption key.
)

// ipcMsg represents the message sento over IPC socket to communicate with SWUpdate,
// start updating the device or querying the status.
type ipcMsg struct {
	magic int32
	typ   msgType
	data  [ipcMsgDataSize]byte
}

// Marshal encodes the message into its binary format.
func (m *ipcMsg) Marshal() []byte {
	buf := make([]byte, 8+ipcMsgDataSize)
	binary.NativeEndian.PutUint32(buf[:4], uint32(m.magic))
	binary.NativeEndian.PutUint32(buf[4:8], uint32(m.typ))
	copy(buf[8:], m.data[:])
	return buf
}

// Unmarshal decodes the message from its binary format into the ipcMsg struct.
func (m *ipcMsg) Unmarshal(r io.Reader) error {
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return fmt.Errorf("goswu: reading ipc header: %w", err)
	}
	m.magic = int32(binary.NativeEndian.Uint32(header[:4]))
	m.typ = msgType(binary.NativeEndian.Uint32(header[4:8]))
	return nil
}

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
