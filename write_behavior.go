package fluentlog

type WriteBehavior uint8

const (
	// Any writes to a full buffer will block.
	// This guarantees that no logs are lost, but might block the application
	// until there is room in the buffer. This also means that if the client
	// can't write any logs at all, the application might get locked. For this
	// reason, this option is discouraged for clients dependent on remote hosts.
	Block WriteBehavior = iota

	// Any writes to a full buffer will be dropped immediately.
	// This guarantees that the application will never be blocked by logging,
	// but also means that log messages might be lost during busy times.
	Loose

	// Any writes to a full buffer will fallback to a compressed, disk-based
	// ping-pong buffer and retried later.
	// This guarantees that no logs neither lost nor blocking the application, but
	// also requires that the client implements the BatchWriter interface. This
	// should be the prefered option when possible.
	Fallback
)
