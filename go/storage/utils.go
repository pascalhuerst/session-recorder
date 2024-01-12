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
	readers := make([]io.Reader, 0, count)
	pipeWriters := make([]io.Writer, 0, count)
	pipeClosers := make([]io.Closer, 0, count)

	for i := 0; i < count; i++ {
		pr, pw := io.Pipe()
		readers = append(readers, pr)
		pipeWriters = append(pipeWriters, pw)
		pipeClosers = append(pipeClosers, pw)
	}

	return readers, io.MultiWriter(pipeWriters...), newMultiCloser(pipeClosers)
}
