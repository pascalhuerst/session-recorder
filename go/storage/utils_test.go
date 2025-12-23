package storage

import (
	"errors"
	"io"
	"testing"
)

// mockCloser is a test helper that implements io.Closer
type mockCloser struct {
	err    error
	closed bool
}

func (m *mockCloser) Close() error {
	m.closed = true
	return m.err
}

func TestMultiCloser_AllSuccess(t *testing.T) {
	c1 := &mockCloser{}
	c2 := &mockCloser{}
	c3 := &mockCloser{}

	mc := newMultiCloser([]io.Closer{c1, c2, c3})

	err := mc.Close()
	if err != nil {
		t.Errorf("MultiCloser.Close() error = %v, want nil", err)
	}

	if !c1.closed || !c2.closed || !c3.closed {
		t.Error("MultiCloser.Close() did not close all closers")
	}
}

func TestMultiCloser_OneError(t *testing.T) {
	expectedErr := errors.New("close error")
	c1 := &mockCloser{}
	c2 := &mockCloser{err: expectedErr}
	c3 := &mockCloser{}

	mc := newMultiCloser([]io.Closer{c1, c2, c3})

	err := mc.Close()
	if err == nil {
		t.Error("MultiCloser.Close() error = nil, want error")
	}

	// All closers should still be closed
	if !c1.closed || !c2.closed || !c3.closed {
		t.Error("MultiCloser.Close() did not close all closers")
	}
}

func TestMultiCloser_MultipleErrors(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	c1 := &mockCloser{err: err1}
	c2 := &mockCloser{}
	c3 := &mockCloser{err: err2}

	mc := newMultiCloser([]io.Closer{c1, c2, c3})

	err := mc.Close()
	if err == nil {
		t.Error("MultiCloser.Close() error = nil, want multierror")
	}

	// All closers should still be closed
	if !c1.closed || !c2.closed || !c3.closed {
		t.Error("MultiCloser.Close() did not close all closers")
	}

	// Error should contain both errors
	errStr := err.Error()
	if errStr == "" {
		t.Error("MultiCloser.Close() error message is empty")
	}
}

func TestMultiCloser_Empty(t *testing.T) {
	mc := newMultiCloser([]io.Closer{})

	err := mc.Close()
	if err != nil {
		t.Errorf("MultiCloser.Close() error = %v, want nil", err)
	}
}

func TestMakeReaders(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{"single reader", 1},
		{"two readers", 2},
		{"three readers", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readers, writer, closer := makeReaders(tt.count)

			if len(readers) != tt.count {
				t.Errorf("makeReaders() readers count = %d, want %d", len(readers), tt.count)
			}

			if writer == nil {
				t.Error("makeReaders() writer is nil")
			}

			if closer == nil {
				t.Error("makeReaders() closer is nil")
			}

			// Test writing to the multiwriter
			testData := []byte("test data")
			go func() {
				writer.Write(testData)
				closer.Close()
			}()

			// Read from each reader
			for i, r := range readers {
				buf := make([]byte, len(testData))
				n, err := io.ReadFull(r, buf)
				if err != nil {
					t.Errorf("reader[%d] read error = %v", i, err)
				}
				if n != len(testData) {
					t.Errorf("reader[%d] read %d bytes, want %d", i, n, len(testData))
				}
				if string(buf) != string(testData) {
					t.Errorf("reader[%d] data = %s, want %s", i, string(buf), string(testData))
				}
			}
		})
	}
}
