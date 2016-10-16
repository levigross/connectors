//    Copyright 2016 Levi Gross
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package connectors

import (
	"io"
	"net"
	"time"
)

// LatencyConn adds artificial latency to a net.Conn connection
type LatencyConn struct {
	lineRate                       int64
	quantum                        time.Duration
	wrappedConnection              net.Conn
	internalReader, externalReader *io.PipeReader
	internalWriter, externalWriter *io.PipeWriter
}

// NewLatencyConn returns a new instance of NewLatencyConn – please remember to close the connection
// ensure that the background goroutine exits
func NewLatencyConn(LineRate int64, Quantum time.Duration, existingConn net.Conn) *LatencyConn {
	internalReader, internalWriter := io.Pipe()
	externalReader, externalWriter := io.Pipe()
	lc := &LatencyConn{
		lineRate:          LineRate,
		quantum:           Quantum,
		wrappedConnection: existingConn,
		internalReader:    internalReader,
		externalReader:    externalReader,
		internalWriter:    internalWriter,
		externalWriter:    externalWriter,
	}
	go lc.backgroundWriter()
	go lc.backgroundReader()
	return lc
}

var _ net.Conn = &LatencyConn{}

func (lc *LatencyConn) Read(b []byte) (n int, err error) {
	return lc.internalReader.Read(b)
}

func (lc *LatencyConn) Write(b []byte) (n int, err error) {
	return lc.externalWriter.Write(b)
}

func (lc *LatencyConn) backgroundWriter() {
	ticker := time.NewTicker(lc.quantum)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			readableBytes := lc.lineRate

		stillHaveBytes:
			byteCount, err := io.Copy(lc.wrappedConnection, io.LimitReader(lc.externalReader, readableBytes))
			if err != nil {
				lc.externalReader.CloseWithError(err)
				return
			}

			readableBytes -= byteCount

			if readableBytes != int64(0) {
				goto stillHaveBytes
			}
			continue
		}
	}
}

func (lc *LatencyConn) backgroundReader() {
	ticker := time.NewTicker(lc.quantum)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			readableBytes := lc.lineRate

		stillHaveBytes:
			byteCount, err := io.Copy(lc.internalWriter, io.LimitReader(lc.wrappedConnection, readableBytes))
			if err != nil {
				lc.internalReader.CloseWithError(err)
				return
			}

			readableBytes -= byteCount

			if readableBytes != int64(0) {
				goto stillHaveBytes
			}
			continue
		}
	}
}

// Close ...
func (lc *LatencyConn) Close() error {
	return anyError(
		lc.externalWriter.Close(),
		lc.internalWriter.Close(),
		lc.internalReader.Close(),
		lc.externalReader.Close(),
		lc.wrappedConnection.Close(),
	)
}

// LocalAddr calls the LocalAddr method on the wrapped net.Conn
func (lc LatencyConn) LocalAddr() net.Addr {
	return lc.wrappedConnection.LocalAddr()
}

// RemoteAddr calls the RemoteAddr on the wrapped net.Conn
func (lc LatencyConn) RemoteAddr() net.Addr {
	return lc.wrappedConnection.RemoteAddr()
}

// SetDeadline calls the SetDeadline method on the wrapped net.Conn
func (lc *LatencyConn) SetDeadline(t time.Time) error {
	return lc.wrappedConnection.SetDeadline(t)
}

// SetReadDeadline calls the SetReadDeadline method on the wrapped net.Conn
func (lc *LatencyConn) SetReadDeadline(t time.Time) error {
	return lc.wrappedConnection.SetReadDeadline(t)
}

// SetWriteDeadline calls the SetWriteDeadline method on the wrapped net.Conn
func (lc *LatencyConn) SetWriteDeadline(t time.Time) error {
	return lc.wrappedConnection.SetWriteDeadline(t)
}

// anyError returns the first error that isn't nil
func anyError(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}
