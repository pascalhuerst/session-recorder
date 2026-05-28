package storage

import (
	"errors"
	"io"

	multierror "github.com/hashicorp/go-multierror"
)

// closeReader closes r if it implements io.Closer. Used to unblock a fan-out
// MultiWriter when a consumer stops reading early (e.g. on encode error).
func closeReader(r io.Reader) {
	if c, ok := r.(io.Closer); ok {
		_ = c.Close()
	}
}

// isPipeClosed reports whether err is the result of a closed io.Pipe — expected
// when a consumer finishes/aborts before the fan-out copy has drained the rest.
func isPipeClosed(err error) bool {
	return errors.Is(err, io.ErrClosedPipe)
}

type MultiCloser struct {
	closers []io.Closer
}

func newMultiCloser(closers []io.Closer) *MultiCloser {
	return &MultiCloser{
		closers: closers,
	}
}

func (m *MultiCloser) Close() error {
	var err error
	for _, c := range m.closers {
		if e := c.Close(); e != nil {
			err = multierror.Append(err, e)
		}
	}
	return err
}

func makeReaders(count int) ([]io.Reader, io.Writer, io.Closer) {
	readers := make([]io.Reader, count)
	pipeWriters := make([]io.Writer, count)
	pipeClosers := make([]io.Closer, count)

	for i := 0; i < count; i++ {
		pr, pw := io.Pipe()
		readers[i] = pr
		pipeWriters[i] = pw
		pipeClosers[i] = pw
	}

	return readers, io.MultiWriter(pipeWriters...), newMultiCloser(pipeClosers)
}
