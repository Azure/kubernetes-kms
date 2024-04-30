// Package mlog implements a thin layer over logr to help enforce mo's logging convention.
// Logs are always structured as a constant message with key and value pairs of related metadata.
//
// The logging levels in order of increasing verbosity are:
// error, warning, info, debug, trace and all.
//
// error and warning logs are always emitted (there is no way for the end user to disable them),
// and thus should be used sparingly.  Ideally, logs at these levels should be actionable.
//
// info should be reserved for "nice to know" information.  It should be possible to run a production
// server at the info log level with no performance degradation due to high log volume.
// debug should be used for information targeted at developers and to aid in support cases.  Care must
// be taken at this level to not leak any secrets into the log stream.  That is, even though debug may
// cause performance issues in production, it must not cause security issues in production.
//
// trace should be used to log information related to timing (i.e. the time it took a controller to sync).
// Just like debug, trace should not leak secrets into the log stream.  trace will likely leak information
// about the current state of the process, but that, along with performance degradation, is expected.
//
// all is reserved for the most verbose and security sensitive information.  At this level, full request
// metadata such as headers and parameters along with the body may be logged.  This level is completely
// unfit for production use both from a performance and security standpoint.  Using it is generally an
// act of desperation to determine why the system is broken.
package mlog

import (
	"os"

	"github.com/go-logr/logr"
)

const errorKey = "error" // this matches zapr's default for .Error calls (which is asserted via tests)

// Logger implements the mlog logging convention described above.  The global functions in this package
// such as Info should be used when one does not intend to write tests assertions for specific log messages.
// If test assertions are desired, Logger should be passed in as an input.  New should be used as the
// production implementation and TestLogger should be used to write test assertions.
type Logger interface {
	Error(msg string, err error, keysAndValues ...interface{})
	Warning(msg string, keysAndValues ...interface{})
	WarningErr(msg string, err error, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	InfoErr(msg string, err error, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
	DebugErr(msg string, err error, keysAndValues ...interface{})
	Trace(msg string, keysAndValues ...interface{})
	TraceErr(msg string, err error, keysAndValues ...interface{})
	All(msg string, keysAndValues ...interface{})
	Always(msg string, keysAndValues ...interface{})
	WithValues(keysAndValues ...interface{}) Logger
	WithName(name string) Logger

	// does not include Fatal on purpose because that is not a method you should be using

	// for internal and test use only
	withDepth(d int) Logger
	withLogrMod(mod func(logr.Logger) logr.Logger) Logger
}

// minLogger is the overlap between Logger and logr.Logger.
type minLogger interface {
	Info(msg string, keysAndValues ...interface{})
}

var _ Logger = mLogger{}
var _, _, _ minLogger = mLogger{}, logr.Logger{}, Logger(nil)

type mLogger struct {
	mods  []func(logr.Logger) logr.Logger
	depth int
}

func New() Logger {
	return mLogger{}
}

func (p mLogger) Error(msg string, err error, keysAndValues ...interface{}) {
	p.logr().WithCallDepth(p.depth+1).Error(err, msg, keysAndValues...)
}

func (p mLogger) warningDepth(msg string, depth int, keysAndValues ...interface{}) {
	if p.logr().V(klogLevelWarning).Enabled() {
		// klog's structured logging has no concept of a warning (i.e. no WarningS function)
		// Thus we use info at log level zero as a proxy
		// klog's info logs have an I prefix and its warning logs have a W prefix
		// Since we lose the W prefix by using InfoS, just add a key to make these easier to find
		keysAndValues = append([]interface{}{"warning", true}, keysAndValues...)
		p.logr().V(klogLevelWarning).WithCallDepth(depth+1).Info(msg, keysAndValues...)
	}
}

func (p mLogger) Warning(msg string, keysAndValues ...interface{}) {
	p.warningDepth(msg, p.depth+1, keysAndValues...)
}

func (p mLogger) WarningErr(msg string, err error, keysAndValues ...interface{}) {
	p.warningDepth(msg, p.depth+1, append([]interface{}{errorKey, err}, keysAndValues...)...)
}

func (p mLogger) infoDepth(msg string, depth int, keysAndValues ...interface{}) {
	if p.logr().V(klogLevelInfo).Enabled() {
		p.logr().V(klogLevelInfo).WithCallDepth(depth+1).Info(msg, keysAndValues...)
	}
}

func (p mLogger) Info(msg string, keysAndValues ...interface{}) {
	p.infoDepth(msg, p.depth+1, keysAndValues...)
}

func (p mLogger) InfoErr(msg string, err error, keysAndValues ...interface{}) {
	p.infoDepth(msg, p.depth+1, append([]interface{}{errorKey, err}, keysAndValues...)...)
}

func (p mLogger) debugDepth(msg string, depth int, keysAndValues ...interface{}) {
	if p.logr().V(klogLevelDebug).Enabled() {
		p.logr().V(klogLevelDebug).WithCallDepth(depth+1).Info(msg, keysAndValues...)
	}
}

func (p mLogger) Debug(msg string, keysAndValues ...interface{}) {
	p.debugDepth(msg, p.depth+1, keysAndValues...)
}

func (p mLogger) DebugErr(msg string, err error, keysAndValues ...interface{}) {
	p.debugDepth(msg, p.depth+1, append([]interface{}{errorKey, err}, keysAndValues...)...)
}

func (p mLogger) traceDepth(msg string, depth int, keysAndValues ...interface{}) {
	if p.logr().V(klogLevelTrace).Enabled() {
		p.logr().V(klogLevelTrace).WithCallDepth(depth+1).Info(msg, keysAndValues...)
	}
}

func (p mLogger) Trace(msg string, keysAndValues ...interface{}) {
	p.traceDepth(msg, p.depth+1, keysAndValues...)
}

func (p mLogger) TraceErr(msg string, err error, keysAndValues ...interface{}) {
	p.traceDepth(msg, p.depth+1, append([]interface{}{errorKey, err}, keysAndValues...)...)
}

func (p mLogger) All(msg string, keysAndValues ...interface{}) {
	if p.logr().V(klogLevelAll).Enabled() {
		p.logr().V(klogLevelAll).WithCallDepth(p.depth+1).Info(msg, keysAndValues...)
	}
}

func (p mLogger) Always(msg string, keysAndValues ...interface{}) {
	p.logr().WithCallDepth(p.depth+1).Info(msg, keysAndValues...)
}

func (p mLogger) WithValues(keysAndValues ...interface{}) Logger {
	if len(keysAndValues) == 0 {
		return p
	}

	return p.withLogrMod(func(l logr.Logger) logr.Logger {
		return l.WithValues(keysAndValues...)
	})
}

func (p mLogger) WithName(name string) Logger {
	if len(name) == 0 {
		return p
	}

	return p.withLogrMod(func(l logr.Logger) logr.Logger {
		return l.WithName(name)
	})
}

func (p mLogger) withDepth(d int) Logger {
	out := p
	out.depth += d // out is a copy so this does not mutate p
	return out
}

func (p mLogger) withLogrMod(mod func(logr.Logger) logr.Logger) Logger {
	out := p // make a copy and carefully avoid mutating the mods slice
	mods := make([]func(logr.Logger) logr.Logger, 0, len(out.mods)+1)
	mods = append(mods, out.mods...)
	mods = append(mods, mod)
	out.mods = mods
	return out
}

func (p mLogger) logr() logr.Logger {
	l := Logr() // grab the current global logger and its current config
	for _, mod := range p.mods {
		mod := mod
		l = mod(l) // and then update it with all modifications
	}
	return l // this logger is guaranteed to have the latest config and all modifications
}

var logger = New().withDepth(1) //nolint:gochecknoglobals

func Error(msg string, err error, keysAndValues ...interface{}) {
	logger.Error(msg, err, keysAndValues...)
}

func Warning(msg string, keysAndValues ...interface{}) {
	logger.Warning(msg, keysAndValues...)
}

func WarningErr(msg string, err error, keysAndValues ...interface{}) {
	logger.WarningErr(msg, err, keysAndValues...)
}

func Info(msg string, keysAndValues ...interface{}) {
	logger.Info(msg, keysAndValues...)
}

func InfoErr(msg string, err error, keysAndValues ...interface{}) {
	logger.InfoErr(msg, err, keysAndValues...)
}

func Debug(msg string, keysAndValues ...interface{}) {
	logger.Debug(msg, keysAndValues...)
}

func DebugErr(msg string, err error, keysAndValues ...interface{}) {
	logger.DebugErr(msg, err, keysAndValues...)
}

func Trace(msg string, keysAndValues ...interface{}) {
	logger.Trace(msg, keysAndValues...)
}

func TraceErr(msg string, err error, keysAndValues ...interface{}) {
	logger.TraceErr(msg, err, keysAndValues...)
}

func All(msg string, keysAndValues ...interface{}) {
	logger.All(msg, keysAndValues...)
}

func Always(msg string, keysAndValues ...interface{}) {
	logger.Always(msg, keysAndValues...)
}

func WithValues(keysAndValues ...interface{}) Logger {
	// this looks weird but it is the same as New().WithValues(keysAndValues...) because it returns a new logger rooted at the call site
	return logger.withDepth(-1).WithValues(keysAndValues...)
}

func WithName(name string) Logger {
	// this looks weird but it is the same as New().WithName(name) because it returns a new logger rooted at the call site
	return logger.withDepth(-1).WithName(name)
}

func Fatal(err error, keysAndValues ...interface{}) {
	logger.Error("unrecoverable error encountered", err, keysAndValues...)
	globalFlush()
	os.Exit(1)
}
