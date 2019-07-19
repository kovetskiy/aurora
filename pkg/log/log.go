package log

import (
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/cog"
	"github.com/reconquest/karma-go"
)

var (
	Logger *cog.Logger
	stderr *lorg.Log
)

func init() {
	stderr = lorg.NewLog()
	stderr.SetIndentLines(true)
	stderr.SetFormat(
		lorg.NewFormat("${time} ${level:[%s]:right:short} ${prefix}%s"),
	)

	Logger = cog.NewLogger(stderr)

	Logger.SetLevel(lorg.LevelDebug)
}

func SetDebug(enabled bool) {
	if enabled {
		stderr.SetLevel(lorg.LevelDebug)
	} else {
		stderr.SetLevel(lorg.LevelInfo)
	}
}

func SetLevel(level lorg.Level) {
	stderr.SetLevel(level)
}

func Fatalf(
	err error,
	message string,
	args ...interface{},
) {
	Logger.Fatalf(err, message, args...)
}

func Errorf(
	err error,
	message string,
	args ...interface{},
) {
	Logger.Errorf(err, message, args...)
}

func Warningf(
	err error,
	message string,
	args ...interface{},
) {
	Logger.Warningf(err, message, args...)
}

func Infof(
	context *karma.Context,
	message string,
	args ...interface{},
) {
	Logger.Infof(context, message, args...)
}

func Debugf(
	context *karma.Context,
	message string,
	args ...interface{},
) {
	Logger.Debugf(context, message, args...)
}

func Tracef(
	context *karma.Context,
	message string,
	args ...interface{},
) {
	Logger.Tracef(context, message, args...)
}

func Fatal(values ...interface{}) {
	Logger.Fatal(values...)
}

func Error(values ...interface{}) {
	Logger.Error(values...)
}

func Warning(values ...interface{}) {
	Logger.Warning(values...)
}

func Info(values ...interface{}) {
	Logger.Info(values...)
}

func Debug(values ...interface{}) {
	Logger.Debug(values...)
}

func Trace(values ...interface{}) {
	Logger.Trace(values...)
}

func TraceJSON(obj interface{}) string {
	return Logger.TraceJSON(obj)
}
