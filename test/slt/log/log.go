/*
Copyright 2021 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

package web

import (
	"fmt"
	"github.com/labstack/gommon/log"
	"os"
)

// Logger is a custom object that overwrites Fatal() to log the error to Kubernetes
// error logs file
type Logger struct {
	*log.Logger
	filePath string
}

// Fatal ...
func (l *Logger) Fatal(i ...interface{}) {
	err := os.WriteFile(l.filePath, []byte(fmt.Sprint(i...)), 0777)
	if err != nil {
		l.Logger.Warnf("Failed to open and write to the logging file: %s", err)
	}
	l.Logger.Fatal(i...)
}

// Fatalf ...
func (l *Logger) Fatalf(format string, args ...interface{}) {
	err := os.WriteFile(l.filePath, []byte(fmt.Sprintf(format, args...)), 0777)
	if err != nil {
		l.Logger.Errorf("Failed to open and write to the logging file: %s", err)
	}
	l.Logger.Fatalf(format, args...)
}

// NewLogger ..
func NewLogger(name string) *Logger {
	l := log.New(name)
	// l.SetOutput(NewWriter(l.Output(), getLock()))
	return &Logger{
		l,
		"/dev/termination-log",
	}
}
