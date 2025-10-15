# Fluentlog

**Fluentlog** is a high-performance, asynchronous logging library for Go that implements the Fluent Forward protocol (used by [FluentBit](https://fluentbit.io/) and [Fluentd](https://www.fluentd.org/), hence the name). It is designed for minimal overhead and maximum flexibility, with zero allocations during logging operations.

> **Note:** While this package integrates with the awesome work of the Fluentd team, this is **not** an official package of theirs.

## Features

- **Asynchronous Logging:**  
  Log entries are queued and processed in background workers to keep logging operations fast and non-blocking.

- **Multiple Write Modes:**
  - **Block:** Log calls block when the internal queue is full, ensuring that no message is lost.
  - **Loose:** Log messages are dropped immediately if the queue is full, ensuring that the application is never blocked.
  - **Fallback:** When the queue is full, log messages are written to a disk-based fallback buffer. *(Requires that the client implements the `BatchWriter` interface.)*

- **Structured Logging:**  
  Each log entry is a structured message that includes a tag, timestamp, and key-value pairs. All logging operations perform zero allocations.  
  See [Structured Logging](#structured-logging) for details on how to pass metadata as key-value pairs.

- **Severity Levels:**  
  Log messages can be emitted with different syslog severity levels (e.g., DEBUG, INFO, WARN, ERROR, CRIT).

- **Formatted Logging:**  
  Support for both plain string messages and formatted (printf-style) messages.

- **Sub-loggers with Inherited Metadata:**  
  Create child loggers that automatically include additional context. The metadata for both logging operations and sub-loggers is provided as key-value pairs.  
  For details, see [Structured Logging](#structured-logging).

- **Panic Recovery:**  
  Helper functions allow you to recover from panics and log them as critical errors.

- **Fluent Forward Protocol:**  
  The Forward client implements the [Fluent Forward protocol](https://github.com/fluent/fluentd/wiki/Forward-Protocol-Specification-v1.5), making it compatible with popular log collectors like FluentBit and Fluentd.

## Installation

Install Fluentlog and its dependencies using `go get`:

```sh
go get github.com/webmafia/fluentlog
```

When choosing between FluentBit and Fluentd, always pick the lighter FluentBit unless you need Fluentd's additional features.

## Getting Started

Below is an example demonstrating how to set up a Fluentlog instance, create a logger, use sub-loggers with metadata, and log messages with different severity levels.

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/webmafia/fluentlog"
	"github.com/webmafia/fluentlog/fallback"
	"github.com/webmafia/fluentlog/forward"
)

func main() {
	// Listen for interrupt signals to allow graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := startClient(ctx); err != nil {
		log.Println(err)
	}
}

func startClient(ctx context.Context) error {
	// Configure the log client.
	// For example, using the forward client to send logs to a remote endpoint
	// that implements the Fluent Forward protocol.
	addr := "localhost:24284"
	cli := forward.NewClient(addr, forward.ClientOptions{
		SharedKey: []byte("secret"),
	})

	// Create a new Fluentlog instance with desired options.
	inst, err := fluentlog.NewInstance(cli, fluentlog.Options{
		WriteBehavior:       fluentlog.Fallback,               // Use fallback when the log queue is full.
		Fallback:            fallback.NewDirBuffer("fluentlog"),   // Disk-based fallback buffer using "fluentlog" directory.
		StackTraceThreshold: fluentlog.NOTICE,                 // Threshold for including a stack trace.
	})
	if err != nil {
		return err
	}
	defer inst.Close()

	// Acquire a new logger.
	l := fluentlog.NewLogger(inst)

	// Create a sub-logger with additional metadata.
	// Pass metadata as key-value pairs: the odd arguments are the keys and the even arguments are the values.
	sub := l.With(
		"component", "database",
		"operation", "query",
	)
	defer sub.Release()

	// Log several informational messages.
	for i := 0; i < 10; i++ {
		sub.Infof("message %d", i+1)
	}

	return nil
}
```

## Structured Logging

Fluentlog supports structured logging by allowing you to pass key-value pairs as arguments to log messages and sub-loggers. The logging methods accept an initial message (or format) string followed by a variadic list of arguments. These arguments must be provided in pairs:

- **Odd-indexed arguments:** Keys (must be strings according to the Fluent Forward protocol)
- **Even-indexed arguments:** Corresponding values

For example, the following call:

```go
l.Info("Server started",
	"port", 8080,
	"env", "production",
)
```

logs a message with metadata where `"port"` is paired with `8080` and `"env"` is paired with `"production"`.

Similarly, when creating sub-loggers with the `With` method:

```go
sub := l.With(
	"component", "database",
	"operation", "query",
)

sub.Info("Query executed successfully",
	"rows", 42,
)
```

the sub-logger automatically includes the metadata (`"component": "database"`, `"operation": "query"`) in every log entry, and additional key-value pairs (like `"rows": 42`) can be provided during each logging call.

## API Overview

### Creating a Logger Instance

The logging system is built around the `Instance` type. Create a new instance using `NewInstance`:

```go
inst, err := fluentlog.NewInstance(cli, fluentlog.Options{
    WriteBehavior:       fluentlog.Fallback,  // Choose Block, Loose, or Fallback.
    Fallback:            fallback.NewDirBuffer("fluentlog"),
    BufferSize:          16,                  // Default is 16 if not specified.
    StackTraceThreshold: fluentlog.NOTICE,
})
if err != nil {
    // Handle error.
}
defer inst.Close()

l := inst.Logger()
// Start logging with `l`.
```

### Logging with the Logger

The `Logger` type provides several methods to log messages at different severity levels. Each method returns a [hexid](https://github.com/webmafia/hexid) that can be used for tracing. For details on providing metadata, see [Structured Logging](#structured-logging).

#### Plain Messages

- `l.Debug(msg string, args ...any) hexid.ID`
- `l.Info(msg string, args ...any) hexid.ID`
- `l.Warn(msg string, args ...any) hexid.ID`
- `l.Error(msg string, args ...any) hexid.ID`

Usage:

```go
l.Info("Server started", "port", 8080)
l.Error("Failed to connect", "reason", err)
```

#### Formatted Messages

- `l.Debugf(format string, args ...any) hexid.ID`
- `l.Infof(format string, args ...any) hexid.ID`
- `l.Warnf(format string, args ...any) hexid.ID`
- `l.Errorf(format string, args ...any) hexid.ID`

Usage:

```go
l.Infof("Listening on port %d", 8080)
l.Errorf("Error: %v", err)
```

### Creating Sub-Loggers with Metadata

Sub-loggers allow you to attach metadata that will be included with every log entry. Pass metadata as key-value pairs (see [Structured Logging](#structured-logging)):

```go
sub := l.With("component", "database", "operation", "query")
defer sub.Release()

sub.Info("Query executed successfully", "rows", 42)
```

### Panic Recovery

To ensure that panics are logged instead of crashing the application, use the `Recover` helper in a deferred call within your goroutine:

```go
go func() {
    defer l.Recover() // Logs the panic as a critical error.
    // Code that might panic.
}()
```

## Write Behavior Modes

Fluentlog supports three write behavior modes via the `Options.WriteBehavior` setting:

- **Block:**  
  If the log queue is full, the log call will block until space becomes available. This ensures that no messages are lost but may cause delays.

- **Loose:**  
  Log messages are dropped if the queue is full. This prevents blocking but may lead to data loss during peak logging periods.

- **Fallback:**  
  When the queue is full, log messages are written to a fallback buffer on disk. This mode prevents both blocking and data loss. *(Requires that the client implements the `BatchWriter` interface.)*

Set the mode when creating the logger instance:

```go
inst, err := fluentlog.NewInstance(cli, fluentlog.Options{
    WriteBehavior: fluentlog.Fallback,
    Fallback:      fallback.NewDirBuffer("fluentlog"),
})
```

## Fallback Buffer

When using the fallback write behavior, Fluentlog uses a disk-based fallback mechanism (e.g., `DirBuffer`) to temporarily store log messages. Key points include:

- **Fallback Queue:**  
  An unbuffered channel to queue messages when the main queue is full.

- **Fallback Worker:**  
  A dedicated goroutine that writes queued messages to disk and later attempts to flush them to the logging destination.

This design helps ensure that no log messages are lost even if the primary logging destination is temporarily unreachable.

## Contributing

Contributions are welcome, but please open an issue first that describes your use case.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

By following the guidelines and examples above, you can integrate Fluentlog into your application, take advantage of its zero-allocation logging operations, and ensure reliable logging using the Fluent Forward protocol. Happy coding!
