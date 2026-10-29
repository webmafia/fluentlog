package forward

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/webmafia/fluentlog"
	"github.com/webmafia/fluentlog/fallback"
	"github.com/webmafia/fluentlog/forward/transport"
	"github.com/webmafia/fluentlog/pkg/msgpack/types"
)

func TestBufferedReplayUsesForwardProtocol(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	accepted, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	allow := func() { once.Do(func() { close(release) }) }
	defer allow()
	records := make(chan int, 201)
	serverErrors := make(chan error, 1)
	server := NewServer(ServerOptions{Auth: StaticAuthServer(Credentials{SharedKey: "secret"})})
	go func() {
		first, err := ln.Accept()
		if err != nil {
			return
		}
		close(accepted)
		<-release
		first.Close()
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				ss := newServerSession(server, conn)
				defer ss.Close()
				if err := ss.authenticate(context.Background()); err != nil {
					return
				}
				ss.initTransportPhase()
				var entry transport.Entry
				for {
					if err := ss.Next(&entry); err != nil {
						return
					}
					n := entry.Record.Items()
					seq := -1
					for i := 0; i < n; i++ {
						if err := entry.Record.NextExpectedType(types.Str); err != nil {
							select {
							case serverErrors <- err:
							default:
							}
							return
						}
						key := string(entry.Record.Str())
						if !entry.Record.Next() {
							return
						}
						if key == "seq" {
							seq = int(entry.Record.Int())
						} else {
							entry.Record.Skip()
						}
					}
					if entry.Tag != "auth" || seq < 0 {
						select {
						case serverErrors <- fmt.Errorf("tag=%q sequence=%d", entry.Tag, seq):
						default:
						}
						return
					}
					records <- seq
				}
			}()
		}
	}()
	cli := NewClient(ln.Addr().String(), ClientOptions{Auth: StaticAuthClient(Credentials{SharedKey: "secret"})})
	inst, err := fluentlog.NewInstance(cli, fluentlog.Options{Tag: "auth", BufferSize: 2, WriteBehavior: fluentlog.Fallback, Fallback: fallback.NewDirBuffer(t.TempDir())})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		allow()
		if err := inst.Close(); err != nil {
			t.Error(err)
		}
	}()
	logger := fluentlog.NewLogger(inst)
	logger.Info("initial", "seq", 0)
	<-accepted
	for i := 1; i <= 200; i++ {
		logger.Info("during outage", "seq", i)
	}
	allow()
	seen := map[int]bool{}
	timer := time.NewTimer(20 * time.Second)
	defer timer.Stop()
	for len(seen) < 201 {
		select {
		case seq := <-records:
			if seen[seq] {
				t.Fatalf("duplicate record %d", seq)
			}
			seen[seq] = true
		case err := <-serverErrors:
			t.Fatal(err)
		case <-timer.C:
			t.Fatalf("recovered %d of 201 records through Forward protocol", len(seen))
		}
	}
}
