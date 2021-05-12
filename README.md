This repository is a mirror of a private GitLab instance. All changes will be overwritten.

# golog

_golog_ is a simple package for logging.

```bash
go get -u github.com/magnetde/log
```

The following types of outputs (_transports_) are so far possible:

## 1. Console

The log entry is printed to stdout. It is also possible to create a colored output or concat the date to the log entry.

## 2. File

Analogous to the console, log entries can be written to a file.
Log files can be rotated by compressing the current log file and adding a sequential suffix (e.g. _example.log.1.gz_).

## 3. Log server

JSON packets are sent to an URL via HTTP POST calls. Packets have the following format:

```json
{
  "type": "my-go-binary",
  "level": "info",
  "date": "2021-02-22T16:11:20+01:00",
  "message": "This is an example log entry"
}
```

HTTP calls are executed in a separate go routine, so calls to log functions do not block until the packet is sent.

Thereby, the correct order is guaranteed.

## Example

```golang
package main

import "github.com/magnetde/log"

func main() {
	log.Init(
		&log.ConsoleTransporter{
			Date:   true,
			Colors: true,
		},
		&log.FileTransporter{
			Path:        "/var/log/test.log",
			Date:        true,
			RotateLines: 50_000, // Rotate after 50,000 lines
			Rotations:   12,     // Keep 12 rotations
		},
		&log.ServerTransporter{
			Type:     "example",
			URL:      "http://localhost/log",
			Secret:   "logging",
			MinLevel: "info",
		},
	)

	log.Info("Example info")
	log.Error("Example error")
	log.Close()
}
```

Example output:

```txt
 [info] [2021-02-2216:21:56+01:00] Example info
[error] [2021-02-2216:21:56+01:00] Example error 0.2 ms
```
