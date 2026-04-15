package goswu

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
)

const (
	// DefaultControlSocket is the default path for the SWUpdate control socket.
	DefaultControlSocket = "/tmp/sockinstctrl"

	// DefaultProgressSocket is the default path for the SWUpdate progress socket.
	DefaultProgressSocket = "/tmp/swupdateprog"
)

// Socket talks to SWUpdate over local Unix domain sockets.
// It implements [Transport].
type Socket struct {
	controlPath  string
	progressPath string

	// image source: either a reader or a file path, set via options.
	imageReader io.Reader
	imagePath   string
}

// SocketOption is a functional option for [NewSocket].
type SocketOption func(*Socket)

// WithControlPath overrides the default control socket path.
func WithControlPath(path string) SocketOption {
	return func(s *Socket) { s.controlPath = path }
}

// WithProgressPath overrides the default progress socket path.
func WithProgressPath(path string) SocketOption {
	return func(s *Socket) { s.progressPath = path }
}

// WithImageReader sets an [io.Reader] as the firmware source for local installations.
func WithImageReader(r io.Reader) SocketOption {
	return func(s *Socket) { s.imageReader = r }
}

// WithImagePath sets a file path as the firmware source for local installations.
// The file will be opened and closed automatically during [Socket.Install].
func WithImagePath(path string) SocketOption {
	return func(s *Socket) { s.imagePath = path }
}

// NewSocket creates a [Socket] with the given options.
// Defaults to [DefaultControlSocket] and [DefaultProgressSocket].
func NewSocket(opts ...SocketOption) *Socket {
	s := &Socket{
		controlPath:  DefaultControlSocket,
		progressPath: DefaultProgressSocket,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Install sends a REQ_INSTALL to SWUpdate and waits for an ACK.
// If the source is [SourceLocal], it also streams the firmware image
// over the same connection. Returns [ErrUpdateInProgress] on NACK.
func (s *Socket) Install(req *Request) error {
	conn, err := net.Dial("unix", s.controlPath)
	if err != nil {
		return fmt.Errorf("failed to connect to control socket: %w", err)
	}
	defer conn.Close()

	msg := ipcMsg{
		magic: ipcMagic,
		typ:   msgReqInstall,
	}
	copy(msg.data[:], req.marshal())

	marshalledMsg := msg.Marshal()
	if _, err := conn.Write(marshalledMsg); err != nil {
		return fmt.Errorf("goswu: sending install request: %w", err)
	}

	var resp ipcMsg
	if err := resp.Unmarshal(conn); err != nil {
		return err
	}

	switch resp.typ {
	case msgNACK:
		return ErrNack
	case msgACK:
		// ok
	default:
		return fmt.Errorf("%w: expected ACK, got %d", ErrUnexpectedResponse, resp.typ)
	}

	if req.Source == SourceLocal {
		if err := s.streamImage(conn); err != nil {
			return err
		}
	}
	return nil
}

// streamImage resolves the image source (path or reader) and copies it to the connection.
func (s *Socket) streamImage(conn net.Conn) error {
	var r io.Reader

	switch {
	case s.imagePath != "":
		f, err := os.Open(s.imagePath)
		if err != nil {
			return fmt.Errorf("goswu: opening image file: %w", err)
		}
		defer f.Close()
		r = f
	case s.imageReader != nil:
		r = s.imageReader
	default:
		return fmt.Errorf("goswu: source is local but no image provided, use WithImagePath or WithImageReader")
	}

	if _, err := io.Copy(conn, r); err != nil {
		return fmt.Errorf("goswu: writing image data: %w", err)
	}
	return nil
}

// ReadProgress connects to the progress socket and reads one [ProgressMsg].
func (s *Socket) ReadProgress() (*ProgressMsg, error) {
	conn, err := net.Dial("unix", s.progressPath)
	if err != nil {
		return nil, fmt.Errorf("goswu: connecting to progress socket: %w", err)
	}
	defer conn.Close()

	var msg ProgressMsg
	if err := msg.unmarshal(conn); err != nil {
		return nil, err
	}
	return &msg, nil
}

// StreamProgress connects to the progress socket and streams [ProgressMsg]s to the channel.
// The channel is closed when the context is done or when the connection is closed.
func (s *Socket) StreamProgress(ctx context.Context) (<-chan *ProgressMsg, error) {
	conn, err := net.Dial("unix", s.progressPath)
	if err != nil {
		return nil, fmt.Errorf("goswu: connecting to progress socket: %w", err)
	}

	ch := make(chan *ProgressMsg)
	go func() {
		defer close(ch)
		defer conn.Close()
		for {
			var msg ProgressMsg
			if err := msg.unmarshal(conn); err != nil {
				return
			}

			select {
			case ch <- &msg:
			case <-ctx.Done():
				return
			}

			if msg.Status == StatusSuccess || msg.Status == StatusFailure {
				return
			}
		}
	}()
	return ch, nil
}
