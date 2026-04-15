package goswu

import "errors"

var (
	// ErrNack is returned when SWUpdate sends a NACK response.
	ErrNack = errors.New("goswu: nack received from swupdate")

	// ErrUnexpectedResponse is returned when the response
	// does not match the expected type or cannot be unmarshalled.
	ErrUnexpectedResponse = errors.New("goswu: unexpected response from swupdate")
)
