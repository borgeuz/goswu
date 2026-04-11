package goswu

// Transport is the interface between [Client] and SWUpdate.
// The only built-in implementation is [Socket]; implement this
// interface if you need a different communication channel.
type Transport interface {
	// Install sends a [Request] to SWUpdate and handles the full
	// installation flow for the given source type.
	Install(req *Request) error

	// ReadProgress reads a single [ProgressMsg] from SWUpdate.
	ReadProgress() (*ProgressMsg, error)
}
