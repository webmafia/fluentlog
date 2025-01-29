module github.com/webmafia/fluentlog

go 1.23

replace github.com/webmafia/fast => ../go-fast

require (
	github.com/webmafia/fast v0.12.0
	github.com/webmafia/identifier v0.2.0
)

require github.com/klauspost/compress v1.17.11 // indirect
