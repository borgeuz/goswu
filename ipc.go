package goswu

import (
	"encoding/binary"
	"fmt"
	"io"
)

// swUpdateAPIVersion is the IPC API version supported by this client.
const swUpdateAPIVersion uint32 = 1

// ipcMagic is the magic number used in the SWUpdate IPC protocol header.
const ipcMagic int32 = 0x14052001

// msgType represents the IPC command types used in the SWUpdate protocol.
type msgType int32

const (
	msgReqInstall       msgType = iota // Request a new installation.
	msgACK                             // Positive acknowledgement.
	msgNACK                            // Negative acknowledgement.
	msgGetStatus                       // Query the daemon status.
	msgPostUpdate                      // Post-update notification.
	msgSubprocess                      // SWUpdate subprocess command.
	msgSetAESKey                       // Set an AES decryption key.
	msgSetUpdateState                  // Set the update state.
	msgGetUpdateState                  // Get the update state.
	msgReqInstallExt                   // Request a new installation with extended options.
	msgSetVersionsRange                // Set the versions range.
	msgNotifyStream                    // Notify a stream.
	msgGetHwRevision                   // Get the hardware revision.
	msgSetSwupdateVars                 // Set SWUpdate variables.
	msgGetSwupdateVars                 // Get SWUpdate variables.
	msgSetDeltaUrl                     // Set the delta URL.
)

// ipcMsg represents the message sent over the IPC socket to communicate with SWUpdate,
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
