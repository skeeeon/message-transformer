//file: internal/api/writer.go

package api

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
)

// ResponseWriter extends http.ResponseWriter with additional metrics
type ResponseWriter interface {
	http.ResponseWriter
	Status() int
	BytesWritten() int
	Flush()
}

// bufferedResponseWriter enhances http.ResponseWriter with buffering
type bufferedResponseWriter struct {
	orig        http.ResponseWriter
	buffer      *bytes.Buffer
	status      int
	bytesWritten int
}

// newBufferedResponseWriter creates a new buffered response writer
func newBufferedResponseWriter(w http.ResponseWriter, buf []byte) *bufferedResponseWriter {
	return &bufferedResponseWriter{
		orig:   w,
		buffer: bytes.NewBuffer(buf[:0]),
		status: http.StatusOK,
	}
}

// Header returns the header map
func (w *bufferedResponseWriter) Header() http.Header {
	return w.orig.Header()
}

// WriteHeader captures the status code
func (w *bufferedResponseWriter) WriteHeader(status int) {
	w.status = status
}

// Write buffers the response
func (w *bufferedResponseWriter) Write(p []byte) (int, error) {
	n, err := w.buffer.Write(p)
	w.bytesWritten += n
	return n, err
}

// Status returns the HTTP status code
func (w *bufferedResponseWriter) Status() int {
	return w.status
}

// BytesWritten returns the number of bytes written
func (w *bufferedResponseWriter) BytesWritten() int {
	return w.bytesWritten
}

// Flush writes the buffered data to the original response writer
func (w *bufferedResponseWriter) Flush() {
	if w.status != 0 {
		w.orig.WriteHeader(w.status)
	}
	if w.buffer.Len() > 0 {
		w.orig.Write(w.buffer.Bytes())
		w.buffer.Reset()
	}
}

// Hijack implements the http.Hijacker interface
func (w *bufferedResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.orig.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// CloseNotify implements the http.CloseNotifier interface
func (w *bufferedResponseWriter) CloseNotify() <-chan bool {
	if cn, ok := w.orig.(http.CloseNotifier); ok {
		return cn.CloseNotify()
	}
	return nil
}

// Unwrap returns the original ResponseWriter
func (w *bufferedResponseWriter) Unwrap() http.ResponseWriter {
	return w.orig
}
