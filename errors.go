package goswu

import "errors"

var (
	// ErrUpdateInProgress is returned when an installation is requested
	// but SWUpdate is already processing another update.
	ErrUpdateInProgress = errors.New("goswu: update already in progress")

	// ErrUnexpectedResponse is returned when the IPC response type
	// does not match the expected value.
	ErrUnexpectedResponse = errors.New("goswu: unexpected ipc response")
)
