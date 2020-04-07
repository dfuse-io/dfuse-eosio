package core

import (
	"go.uber.org/zap/zapcore"
)

type AppDef struct {
	ID          string
	Title       string
	Description string
	MetricsID   string
	Logger      *LoggingDef
	InitFunc    func(config *RuntimeConfig, modules *RuntimeModules) error
	FactoryFunc func(config *RuntimeConfig, modules *RuntimeModules) App
}

type LoggingDef struct {
	Title  string
	Levels []zapcore.Level
	Regex  string
}

type App interface {
	Terminating() <-chan struct{}
	Terminated() <-chan struct{}
	Shutdown(err error)
	Err() error
	Run() error
}

type readiable interface {
	IsReady() bool
}
