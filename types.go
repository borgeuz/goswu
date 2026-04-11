package goswu

// SWUpdateAPIVersion is the IPC API version supported by this client.
const SWUpdateAPIVersion uint32 = 1

// ipcMagic is the magic number used in the SWUpdate IPC protocol header.
const ipcMagic int32 = 0x14052001

// Status represents the current state of the SWUpdate daemon.
type Status int32

const (
	StatusIdle    Status = iota // No update in progress.
	StatusStart                // Update is starting.
	StatusRun                  // Update is running.
	StatusSuccess              // Update completed successfully.
	StatusFailure              // Update failed.
	StatusDownload             // Download in progress.
	StatusDone                 // Post-update actions completed.
	StatusSubProgress          // Sub-step progress notification.
)

// RunType controls whether the update performs actual changes.
type RunType int32

const (
	RunDefault RunType = iota // Normal installation.
	RunDryRun                 // Simulate the installation without writing.
	RunInstall                // Force install even if same version.
)

// SourceType identifies the origin of the update image.
type SourceType int32

const (
	SourceUnknown           SourceType = iota // Unknown source.
	SourceWebserver                           // Update from the embedded webserver.
	SourceSuricatta                           // Update from a suricatta server (e.g. hawkBit).
	SourceDownloader                          // Update from the downloader interface.
	SourceLocal                               // Update from a local file.
	SourceChunksDownloader                    // Update from the chunks downloader.
)
