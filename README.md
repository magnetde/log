This repository is a mirror of a private GitLab instance. All changes will be overwritten.

# serverhook

Serverhook is a hook for [logrus](https://github.com/sirupsen/logrus) to send log entries to a private log server.

```bash
go get -u github.com/magnetde/serverhook
```

JSON packets are sent to an URL via HTTP POST calls. Packets have the following format:

```json
{
  "type": "my-go-binary",
  "level": "info",
  "date": "2021-02-22T16:11:20+01:00",
  "message": "This is an example log entry"
}
```

## Available Options

- `serverhook.WithSecret("...")`: secret required by the server
- `serverhook.KeepColors(true)`: keep or strip ANSI colors from the log message
- `serverhook.SuppressErrors(true)`: suppress errors when sending to the server failed
- `serverhook.Synchronous(true)`: log entries are sent synchronously to the server

## Example

Entries can be sent synchronously or asynchronously.

```go
package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/magnetde/serverhook"
)

func main() {
	hook, err := serverhook.NewServerHook("example", "https://example.org/log", serverhook.WithSecret("example"))
	if err != nil {
		// ...
	}

	defer hook.Flush()
	log.AddHook(hook)
}
```