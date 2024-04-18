package storage

import (
	"io"

	multierror "github.com/hashicorp/go-multierror"
)

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
