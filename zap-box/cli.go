// Copyright 2019 dfuse Platform Inc.
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

package zapbox

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// CLILogger wraps a `zap.Logger` pointer and offers only a printf like interface
type CLILogger struct {
	base *zap.Logger
}

func NewCLILogger(base *zap.Logger) *CLILogger {
	return &CLILogger{base}
}

func (l *CLILogger) LoggerReference() **zap.Logger {
	return &l.base
}

func (l *CLILogger) ReconfigureReference() {
	l.base = l.base.WithOptions(zap.AddCallerSkip(1))
}

func (l *CLILogger) Printf(template string, args ...interface{}) {
	if l.base.Core().Enabled(zapcore.InfoLevel) {
		l.base.Check(zap.InfoLevel, fmt.Sprintf(template, args...)).Write()
	}
}

func (l *CLILogger) Debug(msg string, fields ...zapcore.Field) {
	l.base.Check(zap.DebugLevel, msg).Write(fields...)
}

func (l *CLILogger) Warn(msg string, fields ...zapcore.Field) {
	l.base.Check(zap.WarnLevel, msg).Write(fields...)
}

func (l *CLILogger) Error(msg string, fields ...zapcore.Field) {
	l.base.Check(zap.ErrorLevel, msg).Write(fields...)
}

func (l *CLILogger) FatalAppError(app string, err error) {
	msg := fmt.Sprintf("\n################################################################\n"+
		"Fatal error in app %s:\n\n%s"+
		"\n################################################################\n", app, err)
	l.base.Check(zap.ErrorLevel, msg).Write()
}
