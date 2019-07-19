package log

import (
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/cog"
	"github.com/reconquest/karma-go"
)

var (
	logger *cog.Logger
	stderr *lorg.Log
)

func init() {
	stderr = lorg.NewLog()
	stderr.SetIndentLines(true)
	stderr.SetFormat(
		lorg.NewFormat("${time} ${level:[%s]:right:short} ${prefix}%s"),
	)

	logger = cog.NewLogger(stderr)

	logger.SetLevel(lorg.LevelDebug)
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
	logger.Fatalf(err, message, args...)
}

func Errorf(
	err error,
	message string,
	args ...interface{},
) {
	logger.Errorf(err, message, args...)
}

func Warningf(
	err error,
	message string,
	args ...interface{},
) {
	logger.Warningf(err, message, args...)
}

func Infof(
	context *karma.Context,
	message string,
	args ...interface{},
) {
	logger.Infof(context, message, args...)
}

func Debugf(
	context *karma.Context,
	message string,
	args ...interface{},
) {
	logger.Debugf(context, message, args...)
}

func Tracef(
	context *karma.Context,
	message string,
	args ...interface{},
) {
	logger.Tracef(context, message, args...)
}

func Fatal(values ...interface{}) {
	logger.Fatal(values...)
}

func Error(values ...interface{}) {
	logger.Error(values...)
}

func Warning(values ...interface{}) {
	logger.Warning(values...)
}

func Info(values ...interface{}) {
	logger.Info(values...)
}

func Debug(values ...interface{}) {
	logger.Debug(values...)
}

func Trace(values ...interface{}) {
	logger.Trace(values...)
}

func TraceJSON(obj interface{}) string {
	return logger.TraceJSON(obj)
}
