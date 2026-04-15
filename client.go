package goswu

import "context"

// Client wraps a [Transport] and provides a simple API to trigger
// installations and read progress from SWUpdate.
type Client struct {
	transport Transport
	selection *Selection
	dryRun    bool
}

// NewClient returns a Client that talks to SWUpdate through t.
// selection can be nil if no specific software set is needed.
func NewClient(t Transport, selection *Selection) *Client {
	return &Client{
		transport: t,
		selection: selection,
	}
}

// SetDryRun toggles dry-run mode. When enabled, SWUpdate simulates the
// installation without actually writing anything to the device.
func (c *Client) SetDryRun(enabled bool) {
	c.dryRun = enabled
}

// Install triggers an update for the given source type. What happens
// under the hood depends on the transport (e.g. streaming a local .swu
// file over the socket, or telling SWUpdate to pull from a remote server).
func (c *Client) Install(source SourceType) error {
	req := c.buildRequest(source)
	return c.transport.Install(req)
}

// Progress reads the current update progress from SWUpdate.
// Returns a [ProgressMsg] with status, percentage, current step, etc.
func (c *Client) Progress() (*ProgressMsg, error) {
	return c.transport.ReadProgress()
}

// StreamProgress streams the progress of the update.
func (c *Client) StreamProgress(ctx context.Context) (<-chan *ProgressMsg, error) {
	return c.transport.StreamProgress(ctx)
}

func (c *Client) buildRequest(source SourceType) *Request {
	req := &Request{
		APIVersion: swUpdateAPIVersion,
		Source:     source,
		DryRun:     RunDefault,
	}

	if c.dryRun {
		req.DryRun = RunDryRun
	}

	if c.selection != nil {
		req.SoftwareSet = c.selection.SoftwareSet
		req.RunningMode = c.selection.RunningMode
	}

	return req
}
