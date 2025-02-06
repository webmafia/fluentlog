package fluentlog

type Options struct {
	Tag        string
	BufferSize int
}

func (opt *Options) setDefaults() {
	if opt.Tag == "" {
		opt.Tag = "fluentlog"
	}

	if opt.BufferSize <= 0 {
		opt.BufferSize = 16
	}
}
