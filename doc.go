// Package goswu is a Go client for SWUpdate, the software update framework
// for embedded Linux.
//
// It speaks the SWUpdate IPC protocol over Unix domain sockets, letting you
// trigger updates and monitor progress from Go code running on the same device.
//
// # Basic usage
//
//	sock := goswu.NewSocket(goswu.WithImagePath("/tmp/update.swu"))
//	client := goswu.NewClient(sock, goswu.ParseSelection("stable,main"))
//
//	if err := client.Install(goswu.SourceLocal); err != nil {
//	    log.Fatal(err)
//	}
//
// # Progress
//
// Use [Client.Progress] to read a single progress message, or
// [Socket.StreamProgress] to receive progress updates on a channel until the
// update completes.
package goswu
