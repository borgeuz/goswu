package goswu

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"unsafe"
)

var requestSize = 1040 + int(unsafe.Sizeof(sizeT(0)))

// fakeSWUpdate is a minimal SWUpdate daemon simulator that listens on a Unix
// socket and speaks the IPC protocol. Tests configure its behavior through
// the exported fields before calling start().
type fakeSWUpdate struct {
	controlPath string
	listener    net.Listener

	// Response to send after receiving REQ_INSTALL.
	responseType msgType

	// Captured state after a client connects.
	receivedMsg  ipcMsg
	receivedData []byte // firmware bytes received after the handshake

	// done is closed when the server goroutine finishes handling a connection.
	done chan struct{}
	err  error
}

func newFakeSWUpdate(t *testing.T) *fakeSWUpdate {
	t.Helper()
	dir := t.TempDir()
	return &fakeSWUpdate{
		controlPath:  filepath.Join(dir, "sockinstctrl"),
		responseType: msgACK,
		done:         make(chan struct{}),
	}
}

// start begins listening on the control socket. It handles exactly one
// connection in a goroutine and then stops.
func (f *fakeSWUpdate) start(t *testing.T) {
	t.Helper()
	ln, err := net.Listen("unix", f.controlPath)
	if err != nil {
		t.Fatalf("fakeSWUpdate: listen: %v", err)
	}
	f.listener = ln
	t.Cleanup(func() { ln.Close() })

	go func() {
		defer close(f.done)

		conn, err := ln.Accept()
		if err != nil {
			f.err = err
			return
		}
		defer conn.Close()

		// Read the IPC header (magic + type).
		var header [8]byte
		if _, err := io.ReadFull(conn, header[:]); err != nil {
			f.err = err
			return
		}
		f.receivedMsg.magic = int32(binary.NativeEndian.Uint32(header[:4]))
		f.receivedMsg.typ = msgType(binary.NativeEndian.Uint32(header[4:8]))

		// Read the request payload (fixed size for REQ_INSTALL).
		if f.receivedMsg.typ == msgReqInstall {
			payload := make([]byte, requestSize)
			if _, err := io.ReadFull(conn, payload); err != nil {
				f.err = err
				return
			}
			f.receivedMsg.data = payload
		}

		// Send response.
		resp := ipcMsg{magic: ipcMagic, typ: f.responseType}
		if _, err := conn.Write(resp.Marshal()); err != nil {
			f.err = err
			return
		}

		// If we ACK'd, read any firmware data the client streams.
		if f.responseType == msgACK {
			data, err := io.ReadAll(conn)
			if err != nil {
				f.err = err
				return
			}
			f.receivedData = data
		}
	}()
}

// wait blocks until the server goroutine finishes and fails the test on error.
func (f *fakeSWUpdate) wait(t *testing.T) {
	t.Helper()
	<-f.done
	if f.err != nil {
		t.Fatalf("fakeSWUpdate: %v", f.err)
	}
}

// --- Tests ---

func TestInstallLocalWithReader(t *testing.T) {
	fake := newFakeSWUpdate(t)
	fake.start(t)

	firmware := []byte("fake-swu-image-content-12345")

	sock := NewSocket(
		WithControlPath(fake.controlPath),
		WithImageReader(bytes.NewReader(firmware)),
	)
	client := NewClient(sock, ParseSelection("stable,main"))

	if err := client.Install(SourceLocal); err != nil {
		t.Fatalf("Install() error: %v", err)
	}
	fake.wait(t)

	// Verify IPC header.
	if fake.receivedMsg.magic != ipcMagic {
		t.Errorf("magic = %#x, want %#x", fake.receivedMsg.magic, ipcMagic)
	}
	if fake.receivedMsg.typ != msgReqInstall {
		t.Errorf("msg type = %d, want %d (reqInstall)", fake.receivedMsg.typ, msgReqInstall)
	}

	// Verify request fields in the payload.
	data := fake.receivedMsg.data
	apiVersion := binary.NativeEndian.Uint32(data[0:4])
	if apiVersion != swUpdateAPIVersion {
		t.Errorf("api_version = %d, want %d", apiVersion, swUpdateAPIVersion)
	}

	source := SourceType(binary.NativeEndian.Uint32(data[4:8]))
	if source != SourceLocal {
		t.Errorf("source = %d, want %d (SourceLocal)", source, SourceLocal)
	}

	// offsets after Len depend on sizeof(sizeT): 4 on 32-bit, 8 on 64-bit.
	sizeTLen := int(unsafe.Sizeof(sizeT(0)))
	infoOff := 12 + sizeTLen
	swSetOff := infoOff + infoFieldSize
	runModeOff := swSetOff + softwareSetFieldSize

	softwareSet := string(bytes.TrimRight(data[swSetOff:swSetOff+softwareSetFieldSize], "\x00"))
	if softwareSet != "stable" {
		t.Errorf("software_set = %q, want %q", softwareSet, "stable")
	}
	runningMode := string(bytes.TrimRight(data[runModeOff:runModeOff+runningModeFieldSize], "\x00"))
	if runningMode != "main" {
		t.Errorf("running_mode = %q, want %q", runningMode, "main")
	}

	// Verify firmware data.
	if !bytes.Equal(fake.receivedData, firmware) {
		t.Errorf("firmware data = %q, want %q", fake.receivedData, firmware)
	}
}

func TestInstallLocalWithPath(t *testing.T) {
	fake := newFakeSWUpdate(t)
	fake.start(t)

	firmware := []byte("firmware-from-file")
	fwPath := filepath.Join(t.TempDir(), "update.swu")
	if err := os.WriteFile(fwPath, firmware, 0644); err != nil {
		t.Fatalf("writing temp firmware file: %v", err)
	}

	sock := NewSocket(
		WithControlPath(fake.controlPath),
		WithImagePath(fwPath),
	)
	client := NewClient(sock, nil)

	if err := client.Install(SourceLocal); err != nil {
		t.Fatalf("Install() error: %v", err)
	}
	fake.wait(t)

	if !bytes.Equal(fake.receivedData, firmware) {
		t.Errorf("firmware data = %q, want %q", fake.receivedData, firmware)
	}
}

func TestInstallNACK(t *testing.T) {
	fake := newFakeSWUpdate(t)
	fake.responseType = msgNACK
	fake.start(t)

	sock := NewSocket(
		WithControlPath(fake.controlPath),
		WithImageReader(bytes.NewReader([]byte("data"))),
	)
	client := NewClient(sock, nil)

	err := client.Install(SourceLocal)
	if !errors.Is(err, ErrUpdateInProgress) {
		t.Errorf("Install() error = %v, want %v", err, ErrUpdateInProgress)
	}
}

func TestInstallDryRun(t *testing.T) {
	fake := newFakeSWUpdate(t)
	fake.start(t)

	sock := NewSocket(
		WithControlPath(fake.controlPath),
		WithImageReader(bytes.NewReader([]byte("img"))),
	)
	client := NewClient(sock, nil)
	client.SetDryRun(true)

	if err := client.Install(SourceLocal); err != nil {
		t.Fatalf("Install() error: %v", err)
	}
	fake.wait(t)

	dryRun := RunType(binary.NativeEndian.Uint32(fake.receivedMsg.data[8:12]))
	if dryRun != RunDryRun {
		t.Errorf("dry_run = %d, want %d (RunDryRun)", dryRun, RunDryRun)
	}
}

func TestInstallLocalNoImage(t *testing.T) {
	fake := newFakeSWUpdate(t)
	fake.start(t)

	sock := NewSocket(WithControlPath(fake.controlPath))
	client := NewClient(sock, nil)

	err := client.Install(SourceLocal)
	if err == nil {
		t.Fatal("Install() should fail when no image is provided for SourceLocal")
	}
}

func TestInstallConnectionRefused(t *testing.T) {
	sock := NewSocket(WithControlPath("/tmp/goswu-nonexistent-socket"))
	client := NewClient(sock, nil)

	err := client.Install(SourceLocal)
	if err == nil {
		t.Fatal("Install() should fail when socket does not exist")
	}
}

func TestRequestMarshalSize(t *testing.T) {
	req := &Request{
		APIVersion:  swUpdateAPIVersion,
		Source:      SourceLocal,
		DryRun:      RunDefault,
		SoftwareSet: "stable",
		RunningMode: "main",
	}
	data := req.marshal()

	// 4 (apiversion) + 4 (source) + 4 (cmd) + sizeof(sizeT) (len) +
	// 512 (info) + 256 (software_set) + 256 (running_mode) + 4 (disable_store)
	want := 1040 + int(unsafe.Sizeof(sizeT(0)))
	if len(data) != want {
		t.Errorf("Marshal() size = %d, want %d", len(data), want)
	}
}

func TestParseSelection(t *testing.T) {
	tests := []struct {
		input   string
		wantNil bool
		wantSW  string
		wantRM  string
	}{
		{"stable,main", false, "stable", "main"},
		{"", true, "", ""},
		{"onlyone", true, "", ""},
	}

	for _, tt := range tests {
		sel := ParseSelection(tt.input)
		if tt.wantNil {
			if sel != nil {
				t.Errorf("ParseSelection(%q) = %+v, want nil", tt.input, sel)
			}
			continue
		}
		if sel == nil {
			t.Fatalf("ParseSelection(%q) = nil, want non-nil", tt.input)
		}
		if sel.SoftwareSet != tt.wantSW {
			t.Errorf("ParseSelection(%q).SoftwareSet = %q, want %q", tt.input, sel.SoftwareSet, tt.wantSW)
		}
		if sel.RunningMode != tt.wantRM {
			t.Errorf("ParseSelection(%q).RunningMode = %q, want %q", tt.input, sel.RunningMode, tt.wantRM)
		}
	}
}

func TestIpcMsgMarshalRoundtrip(t *testing.T) {
	original := ipcMsg{
		magic: ipcMagic,
		typ:   msgReqInstall,
		data:  []byte("test-payload"),
	}
	wire := original.Marshal()

	magic := int32(binary.NativeEndian.Uint32(wire[:4]))
	typ := msgType(binary.NativeEndian.Uint32(wire[4:8]))
	if magic != ipcMagic {
		t.Errorf("magic = %#x, want %#x", magic, ipcMagic)
	}
	if typ != msgReqInstall {
		t.Errorf("typ = %d, want %d", typ, msgReqInstall)
	}

	var decoded ipcMsg
	if err := decoded.Unmarshal(bytes.NewReader(wire)); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}
	if decoded.magic != original.magic || decoded.typ != original.typ {
		t.Errorf("Unmarshal() = {magic:%#x, typ:%d}, want {magic:%#x, typ:%d}",
			decoded.magic, decoded.typ, original.magic, original.typ)
	}
}
