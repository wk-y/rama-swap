package util

import "io"

type ReadCloserWrapper struct {
	io.Reader
	Closer func() error
}

// Close implements io.ReadCloser.
func (r ReadCloserWrapper) Close() error {
	return r.Closer()
}

var _ io.ReadCloser = ReadCloserWrapper{}
