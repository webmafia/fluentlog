package main

import (
	"io"

	_ "unsafe"

	"github.com/webmafia/fast/buffer"
	"github.com/webmafia/fluentlog/forward"
	"github.com/webmafia/fluentlog/pkg/msgpack"
)

//go:linkname newServerConn forward.newServerConn
func newServerConn(s *forward.Server, conn io.Writer, iter *msgpack.Iterator, wBuf, state *buffer.Buffer) forward.ServerConn
