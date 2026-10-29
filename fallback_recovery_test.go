package fluentlog

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/webmafia/fluentlog/fallback"
	"github.com/webmafia/fluentlog/pkg/msgpack"
	"github.com/webmafia/fluentlog/pkg/msgpack/types"
)

type stalledCollector struct {
	entered, release chan struct{}
	once             sync.Once
	mu               sync.Mutex
	seen             map[int]int
}

func (c *stalledCollector) record(iter *msgpack.Iterator) error {
	if err := iter.NextExpectedType(types.Map); err != nil {
		return err
	}
	n := iter.Items()
	seq := -1
	for i := 0; i < n; i++ {
		if err := iter.NextExpectedType(types.Str); err != nil {
			return err
		}
		key := string(iter.Str())
		if !iter.Next() {
			return iter.Error()
		}
		if key == "seq" {
			seq = int(iter.Int())
		} else {
			iter.Skip()
		}
	}
	if seq < 0 {
		return errors.New("missing sequence")
	}
	c.mu.Lock()
	c.seen[seq]++
	c.mu.Unlock()
	return nil
}
func (c *stalledCollector) Write(p []byte) (int, error) {
	c.once.Do(func() { close(c.entered); <-c.release })
	iter := msgpack.NewIterator(strings.NewReader(string(p)))
	if err := iter.NextExpectedType(types.Array); err != nil {
		return 0, err
	}
	if err := iter.NextExpectedType(types.Str); err != nil {
		return 0, err
	}
	iter.Str()
	if !iter.Next() {
		return 0, iter.Error()
	}
	iter.Skip()
	if err := c.record(&iter); err != nil {
		return 0, err
	}
	return len(p), nil
}
func (c *stalledCollector) WriteBatch(_ string, _ int, r io.Reader) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()
	iter := msgpack.NewIterator(gz)
	for iter.Next() {
		if iter.Type() != types.Array || iter.Items() != 2 {
			return fmt.Errorf("invalid disk record")
		}
		if !iter.Next() {
			return iter.Error()
		}
		iter.Skip()
		if err := c.record(&iter); err != nil {
			return err
		}
		iter.Flush()
	}
	if err := iter.Error(); err != io.EOF {
		return err
	}
	return nil
}
func TestCollectorStallSpillsAndReplaysEveryRecord(t *testing.T) {
	collector := &stalledCollector{entered: make(chan struct{}), release: make(chan struct{}), seen: map[int]int{}}
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(collector.release) }) }
	t.Cleanup(release)
	disk := fallback.NewDirBuffer(t.TempDir())
	inst, err := NewInstance(collector, Options{Tag: strings.Repeat("a", 64), BufferSize: 2, WriteBehavior: Fallback, Fallback: disk})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		release()
		if err := inst.Close(); err != nil {
			t.Error(err)
		}
	})
	logger := NewLogger(inst)
	logger.Info("initial", "seq", 0)
	<-collector.entered
	done := make(chan struct{})
	go func() {
		var wg sync.WaitGroup
		for i := 1; i <= 80; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); logger.Info("queued", "seq", i) }()
		}
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("network stall blocked log producers")
	}
	if has, err := disk.HasData(); err != nil || !has {
		t.Fatalf("queue overflow was not buffered: %v %v", has, err)
	}
	release()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		collector.mu.Lock()
		n := len(collector.seen)
		collector.mu.Unlock()
		if n == 81 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	collector.mu.Lock()
	defer collector.mu.Unlock()
	if len(collector.seen) != 81 {
		t.Fatalf("received %d of 81 records", len(collector.seen))
	}
	for seq, n := range collector.seen {
		if n != 1 {
			t.Errorf("sequence %d received %d times", seq, n)
		}
	}
}
