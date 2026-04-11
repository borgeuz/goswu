# goswu

A Go client library for [SWUpdate](https://swupdate.org/), the software update framework for embedded Linux.

It speaks the SWUpdate IPC protocol over Unix domain sockets, letting you trigger updates and monitor progress from Go code running on the same device.

## Install

```
go get github.com/Borgeouzz/goswu
```

## Usage

### Local update from a file path

```go
sock := goswu.NewSocket(goswu.WithImagePath("/tmp/update.swu"))
client := goswu.NewClient(sock, goswu.ParseSelection("stable,main"))

if err := client.Install(goswu.SourceLocal); err != nil {
    log.Fatal(err)
}
```

### Local update from an io.Reader

```go
f, err := os.Open("/tmp/update.swu")
if err != nil {
    log.Fatal(err)
}
defer f.Close()

sock := goswu.NewSocket(goswu.WithImageReader(f))
client := goswu.NewClient(sock, nil)

if err := client.Install(goswu.SourceLocal); err != nil {
    log.Fatal(err)
}
```

### Dry-run mode

```go
client.SetDryRun(true)
err := client.Install(goswu.SourceLocal) // simulates the update, no changes written
```

### Reading progress

```go
progress, err := client.Progress()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("status=%d step=%d/%d percent=%d%%\n",
    progress.Status, progress.CurStep, progress.NSteps, progress.CurPercent)
```

### Custom socket paths

```go
sock := goswu.NewSocket(
    goswu.WithControlPath("/run/swupdate/ctrl"),
    goswu.WithProgressPath("/run/swupdate/progress"),
    goswu.WithImagePath("/tmp/update.swu"),
)
```

## How it works

SWUpdate exposes two Unix sockets:

- **Control socket** (`/tmp/sockinstctrl`) -- send install requests, receive ACK/NACK
- **Progress socket** (`/tmp/swupdateprog`) -- read `progress_msg` structs with status, percentage, current step, etc.

The client serializes requests and messages matching the C structs from SWUpdate's `network_ipc.h`, so it can talk directly to the daemon without CGo.

## Project structure

```
types.go      -- shared types and constants (Status, RunType, SourceType)
errors.go     -- sentinel errors
ipc.go        -- IPC message format and progress_msg deserialization
request.go    -- Request struct and binary serialization
transport.go  -- Transport interface
socket.go     -- Unix socket Transport implementation
client.go     -- high-level Client API
```

## License

MIT
