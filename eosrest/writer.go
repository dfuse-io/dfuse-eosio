// Copyright 2020 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eosrest

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

type StatusAwareResponseWriter struct {
	http.ResponseWriter
	Status int
}

func (w *StatusAwareResponseWriter) WriteHeader(status int) {
	w.Status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *StatusAwareResponseWriter) Write(b []byte) (int, error) {
	if w.Status == 0 {
		w.Status = http.StatusOK
	}

	return w.ResponseWriter.Write(b)
}

func (w *StatusAwareResponseWriter) Hijack() (rwc net.Conn, buf *bufio.ReadWriter, err error) {
	hijackableWriter, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("wrapped writer should be implementing http.Hijacker but is not")
	}

	return hijackableWriter.Hijack()
}

func WriterStatus(w http.ResponseWriter) (int, error) {
	if wrappedWriter, ok := w.(*StatusAwareResponseWriter); ok {
		return wrappedWriter.Status, nil
	}

	return 0, errors.New("not a status aware response writer")
}

func TurnIntoStatusAwareResponseWriter(w http.ResponseWriter) *StatusAwareResponseWriter {
	return &StatusAwareResponseWriter{ResponseWriter: w}
}
