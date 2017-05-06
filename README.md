# Graceful

Graceful shutdown and restart for Go's `net/http` handlers.

This product is a fork of [miyabi](https://github.com/naoina/miyabi).

## Usage

It's very simple. Use `graceful.ListenAndServe` instead of `http.ListenAndServe`.
You don't have to change other code because `graceful.ListenAndServe` is compatible with `http.ListenAndServe`.

```go
package main

import (
    "io"
    "log"
    "net/http"

    "github.com/kohkimakimoto/graceful"
)

// hello world, the web server
func HelloServer(w http.ResponseWriter, req *http.Request) {
    io.WriteString(w, "hello, world!\n")
}

func main() {
    http.HandleFunc("/hello", HelloServer)
    log.Fatal(graceful.ListenAndServe(":8080", nil))
}
```

**NOTE**: Graceful is using features of Go 1.3, so doesn't work in Go 1.2.x and older versions. Also when using on Windows, it works but graceful shutdown/restart are disabled explicitly.

## Graceful shutdown or restart

By default, send `SIGTERM` or `SIGINT` (Ctrl + c) signal to a process that is using Graceful in order to graceful shutdown and send `SIGHUP` signal in order to graceful restart.
If you want to change the these signal, please set another signal to `graceful.ShutdownSignal` and/or `graceful.RestartSignal`.

In fact, `graceful.ListenAndServe` and `graceful.ListenAndServeTLS` will fork a process that is using Graceful in order to achieve the graceful restart.
This means that you should write code as no side effects until the call of `graceful.ListenAndServe` or `graceful.ListenAndServeTLS`.

## License

Graceful is licensed under the MIT.
