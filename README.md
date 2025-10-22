# minivalkey

A **zero-dependency, in-memory Valkey server** for Go.
Useful for unit and integration testing without running a real Valkey or Redis instance.

---

## Features

* **Zero dependencies** — pure Go standard library only
* **Persistent in-memory data store** with TTL support
* **Implements a subset of Valkey/Redis commands** (`PING`, `SET`, `GET`, `DEL`, `EXPIRE`, `TTL`, etc.)
* **Virtual clock** via `FastForward(duration)` for time-travel testing
* Tested against [`valkey-go`](https://github.com/valkey-io/valkey-go)

---

## Installation

```bash
go get github.com/mickamy/minivalkey
```

> No external dependencies will be added to your project.

---

## Example

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/valkey-io/valkey-go"
    "github.com/mickamy/minivalkey"
)

func main() {
    s, _ := minivalkey.Run()
    defer s.Close()

    client, _ := valkey.NewClient(valkey.ClientOption{
        InitAddress:  []string{s.Addr()},
    })
    defer client.Close()

    ctx := context.Background()

    // Basic usage
    _ = client.Do(ctx, client.B().Set().Key("hello").Value("world").Build())
    resp := client.Do(ctx, client.B().Get().Key("hello").Build())
    val, _ := resp.ToString()
    fmt.Println(val) // world

    // Fast-forward time to test expiration
    _ = client.Do(ctx, client.B().Expire().Key("hello").Seconds(3).Build())
    s.FastForward(4 * time.Second)
    expired := client.Do(ctx, client.B().Get().Key("hello").Build())
    _, err := expired.ToString()
    fmt.Println(err) // valkey nil message
}
```

---

## Supported Commands

| Category             | Commands                                            |
| -------------------- | --------------------------------------------------- |
| **Connection**       | `PING`, `ECHO`, `HELLO`                             |
| **Keys**             | `DEL`, `EXISTS`                                     |
| **Strings**          | `SET`, `GET`, `MSET`, `MGET`, `INCR`, `DECR`        |
| **TTL / Expiration** | `EXPIRE`, `PEXPIRE`, `TTL`, `PTTL`                  |
| **Server / Info**    | `INFO`, `CLIENT` (stub), `FASTFORWARD` (Go API)     |
| **Planned**          | `HSET`, `HGET`, `LPUSH`, `LRANGE`, `SCAN`, `PUBSUB` |

---

## Testing

Unit tests live in the main module (`go test ./...`).

End-to-end tests using [`valkey-go`](https://github.com/valkey-io/valkey-go) are isolated under `e2e/`
as a separate Go module, so production builds remain zero-dependency.

```bash
cd e2e
go test ./...
```

---

## Go Compatibility

* **Minimum:** Go 1.24
* **Tested on:** Go 1.24 - 1.25
* No third-party dependencies at runtime.

---

## License

MIT[./LICENSE]
