// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"bytes"
	"fmt"
	"log"
)

type Logger interface {
	Info(message string, keyAndValues ...any)
	Error(message string, err error, keyAndValues ...any)
}

type stdlog struct{}

func (l stdlog) Info(message string, keyAndValues ...any) {
	l.log("Info", message, keyAndValues...)
}

func (l stdlog) Error(message string, err error, keyAndValues ...any) {
	if err != nil {
		keyAndValues = append([]any{"error", err.Error()}, keyAndValues...)
	}

	l.log("Error", message, keyAndValues...)
}

func (stdlog) log(level, message string, keyAndValues ...any) {
	buf := new(bytes.Buffer)
	buf.WriteString(level)
	buf.WriteRune(' ')
	buf.WriteString(message)
	for i := 0; i < len(keyAndValues); i += 2 {
		buf.WriteRune(' ')
		buf.WriteString(fmt.Sprintf("%s=%v", keyAndValues[i], keyAndValues[i+1]))
	}

	log.Print(buf)
}
