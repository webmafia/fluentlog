package main

import (
	"io"

	_ "unsafe"

	"github.com/webmafia/fluentlog/forward"
)

//go:linkname newServerSession forward.newServerSession
func newServerSession(s *forward.Server, conn io.Writer) forward.ServerSession
