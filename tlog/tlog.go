package tlog

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	isInitialied bool
	// Sugar represents a zap logger
	Sugar *zap.SugaredLogger
	// SugarFile represents a zap logger file
	SugarFile *zap.SugaredLogger
)

// Init creates and initializes the logging
func Init(fileWriter io.Writer, consoleWriter io.Writer) {
	if isInitialied {
		return
	}

	isInitialied = true
	//pe := zap.NewProductionEncoderConfig()

	consoleConfig := zap.NewDevelopmentConfig()
	consoleConfig.EncoderConfig.EncodeLevel = shortLevelEncoder
	if runtime.GOOS != "windows" {
		consoleConfig.EncoderConfig.EncodeLevel = shortColorLevelEncoder
	}
	consoleConfig.EncoderConfig.ConsoleSeparator = " "
	consoleConfig.EncoderConfig.TimeKey = ""
	consoleEncoder := zapcore.NewConsoleEncoder(consoleConfig.EncoderConfig)

	level := zap.DebugLevel
	if consoleWriter == nil {
		consoleWriter = os.Stdout
	}
	core := zapcore.NewCore(consoleEncoder, zapcore.AddSync(consoleWriter), level)
	Sugar = zap.New(core).Sugar()

	if fileWriter != nil {
		fileConfig := zap.NewDevelopmentConfig()
		fileConfig.EncoderConfig.LevelKey = "L"
		fileConfig.EncoderConfig.EncodeLevel = shortLevelEncoder
		fileConfig.EncoderConfig.ConsoleSeparator = " "
		fileConfig.EncoderConfig.FunctionKey = "F"
		opts := []zap.Option{
			zap.AddCallerSkip(1), // traverse call depth for more useful log lines
			zap.AddCaller(),
		}

		fileEncoder := zapcore.NewConsoleEncoder(fileConfig.EncoderConfig)
		core = zapcore.NewTee(
			zapcore.NewCore(fileEncoder, zapcore.AddSync(fileWriter), level),
		)
		SugarFile = zap.New(core, opts...).Sugar()
	}
}

// Debug uses fmt.Sprint to construct and log a message.
func Debug(args ...interface{}) {
	Init(nil, nil)
	Sugar.Debug(args)
	if SugarFile != nil {
		SugarFile.Debug(args)
	}
}

// Info uses fmt.Sprint to construct and log a message.
func Info(args ...interface{}) {
	Init(nil, nil)
	Sugar.Info(args)
	if SugarFile != nil {
		SugarFile.Info(args)
	}
}

// Warn uses fmt.Sprint to construct and log a message.
func Warn(args ...interface{}) {
	Init(nil, nil)
	Sugar.Warn(args)
	if SugarFile != nil {
		SugarFile.Warn(args)
	}
}

// Error uses fmt.Sprint to construct and log a message.
func Error(args ...interface{}) {
	Init(nil, nil)
	Sugar.Error(args)
	if SugarFile != nil {
		SugarFile.Error(args)
	}
}

// DPanic uses fmt.Sprint to construct and log a message. In development, the
// logger then panics. (See DPanicLevel for details.)
func DPanic(args ...interface{}) {
	Init(nil, nil)
	Sugar.DPanic(args)
	if SugarFile != nil {
		SugarFile.DPanic(args)
	}
}

// Panic uses fmt.Sprint to construct and log a message, then panics.
func Panic(args ...interface{}) {
	Init(nil, nil)
	Sugar.Panic(args)
	if SugarFile != nil {
		SugarFile.Panic(args)
	}
}

// Fatal uses fmt.Sprint to construct and log a message, then calls os.Exit.
func Fatal(args ...interface{}) {
	Init(nil, nil)
	Sugar.Fatal(args)
	if SugarFile != nil {
		SugarFile.Fatal(args)
	}
}

// Debugf uses fmt.Sprintf to log a templated message.
func Debugf(template string, args ...interface{}) {
	Init(nil, nil)
	Sugar.Debugf(template, args...)
	if SugarFile != nil {
		SugarFile.Debugf(template, args...)
	}
}

// Infof uses fmt.Sprintf to log a templated message.
func Infof(template string, args ...interface{}) {
	Init(nil, nil)
	Sugar.Infof(template, args...)
	if SugarFile != nil {
		SugarFile.Infof(template, args...)
	}
}

// Warnf uses fmt.Sprintf to log a templated message.
func Warnf(template string, args ...interface{}) {
	Init(nil, nil)
	Sugar.Warnf(template, args...)
	if SugarFile != nil {
		SugarFile.Warnf(template, args...)
	}
}

// Errorf uses fmt.Sprintf to log a templated message.
func Errorf(template string, args ...interface{}) {
	Init(nil, nil)
	Sugar.Errorf(template, args...)
	if SugarFile != nil {
		SugarFile.Errorf(template, args...)
	}
}

// DPanicf uses fmt.Sprintf to log a templated message. In development, the
// logger then panics. (See DPanicLevel for details.)
func DPanicf(template string, args ...interface{}) {
	Init(nil, nil)
	Sugar.DPanicf(template, args...)
	if SugarFile != nil {
		SugarFile.DPanicf(template, args...)
	}
}

// Panicf uses fmt.Sprintf to log a templated message, then panics.
func Panicf(template string, args ...interface{}) {
	Init(nil, nil)
	Sugar.Panicf(template, args...)
	if SugarFile != nil {
		SugarFile.Panicf(template, args...)
	}
}

// Fatalf uses fmt.Sprintf to log a templated message, then calls os.Exit.
func Fatalf(template string, args ...interface{}) {
	Init(nil, nil)
	Sugar.Fatalf(template, args...)
	if SugarFile != nil {
		SugarFile.Fatalf(template, args...)
	}
}

// Debugw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
//
// When debug-level logging is disabled, this is much faster than
//
//	s.With(keysAndValues).Debug(msg)
func Debugw(msg string, keysAndValues ...interface{}) {
	Init(nil, nil)
	Sugar.Debugw(msg, keysAndValues)
	if SugarFile != nil {
		SugarFile.Debugw(msg, keysAndValues)
	}
}

// Infow logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func Infow(msg string, keysAndValues ...interface{}) {
	Init(nil, nil)
	Sugar.Infow(msg, keysAndValues)
	if SugarFile != nil {
		SugarFile.Infow(msg, keysAndValues)
	}
}

// Warnw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func Warnw(msg string, keysAndValues ...interface{}) {
	Init(nil, nil)
	Sugar.Warnw(msg, keysAndValues)
	if SugarFile != nil {
		SugarFile.Warnw(msg, keysAndValues)
	}
}

// Errorw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func Errorw(msg string, keysAndValues ...interface{}) {
	Init(nil, nil)
	Sugar.Errorw(msg, keysAndValues)
	if SugarFile != nil {
		SugarFile.Errorw(msg, keysAndValues)
	}
}

// DPanicw logs a message with some additional context. In development, the
// logger then panics. (See DPanicLevel for details.) The variadic key-value
// pairs are treated as they are in With.
func DPanicw(msg string, keysAndValues ...interface{}) {
	Init(nil, nil)
	Sugar.DPanicw(msg, keysAndValues)
	if SugarFile != nil {
		SugarFile.DPanicw(msg, keysAndValues)
	}
}

// Panicw logs a message with some additional context, then panics. The
// variadic key-value pairs are treated as they are in With.
func Panicw(msg string, keysAndValues ...interface{}) {
	Init(nil, nil)
	Sugar.Panicw(msg, keysAndValues)
	if SugarFile != nil {
		SugarFile.Panicw(msg, keysAndValues)
	}
}

// Fatalw logs a message with some additional context, then calls os.Exit. The
// variadic key-value pairs are treated as they are in With.
func Fatalw(msg string, keysAndValues ...interface{}) {
	Init(nil, nil)
	Sugar.Fatalw(msg, keysAndValues)
	if SugarFile != nil {
		SugarFile.Fatalw(msg, keysAndValues)
	}
}

// Debugln uses fmt.Sprintln to construct and log a message.
func Debugln(args ...interface{}) {
	Init(nil, nil)
	Sugar.Debugln(args)
	if SugarFile != nil {
		SugarFile.Debugln(args)
	}
}

// Infoln uses fmt.Sprintln to construct and log a message.
func Infoln(args ...interface{}) {
	Init(nil, nil)
	Sugar.Infoln(args)
	if SugarFile != nil {
		SugarFile.Infoln(args)
	}
}

// Warnln uses fmt.Sprintln to construct and log a message.
func Warnln(args ...interface{}) {
	Init(nil, nil)
	Sugar.Warnln(args)
	if SugarFile != nil {
		SugarFile.Warnln(args)
	}
}

// Errorln uses fmt.Sprintln to construct and log a message.
func Errorln(args ...interface{}) {
	Init(nil, nil)
	Sugar.Errorln(args)
	if SugarFile != nil {
		SugarFile.Errorln(args)
	}
}

// DPanicln uses fmt.Sprintln to construct and log a message. In development, the
// logger then panics. (See DPanicLevel for details.)
func DPanicln(args ...interface{}) {
	Init(nil, nil)
	Sugar.DPanicln(args)
	if SugarFile != nil {
		SugarFile.DPanicln(args)
	}
}

// Panicln uses fmt.Sprintln to construct and log a message, then panics.
func Panicln(args ...interface{}) {
	Init(nil, nil)
	Sugar.Panicln(args)
	if SugarFile != nil {
		SugarFile.Panicln(args)
	}
}

// Fatalln uses fmt.Sprintln to construct and log a message, then calls os.Exit.
func Fatalln(args ...interface{}) {
	Init(nil, nil)
	Sugar.Fatalln(args)
	if SugarFile != nil {
		SugarFile.Fatalln(args)
	}
}

// Sync flushes any buffered log entries.
func Sync() error {
	Init(nil, nil)
	if SugarFile != nil {
		err := SugarFile.Sync()
		if err != nil {
			return fmt.Errorf("Sugar.Sync: %w", err)
		}
	}
	return Sugar.Sync()
}

var (
	_levelToCapitalColorString = map[zapcore.Level]string{
		zapcore.DebugLevel:  "\x1b[32mDBG\x1b[0m", //green
		zapcore.InfoLevel:   "\x1b[94mINF\x1b[0m", //bright blue
		zapcore.WarnLevel:   "\x1b[33mWRN\x1b[0m", //yellow
		zapcore.ErrorLevel:  "\x1b[31mERR\x1b[0m", //red
		zapcore.DPanicLevel: "\x1b[36mERR\x1b[0m", //bright red
		zapcore.PanicLevel:  "\x1b[35mERR\x1b[0m", //bright red
		zapcore.FatalLevel:  "\x1b[35mERR\x1b[0m", //brightest red
	}
	_levelToCapitalString = map[zapcore.Level]string{
		zapcore.DebugLevel:  "DBG",
		zapcore.InfoLevel:   "INF",
		zapcore.WarnLevel:   "WRN",
		zapcore.ErrorLevel:  "ERR",
		zapcore.DPanicLevel: "ERR",
		zapcore.PanicLevel:  "ERR",
		zapcore.FatalLevel:  "ERR",
	}
)

func shortColorLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	s, ok := _levelToCapitalColorString[l]
	if !ok {
		s = _levelToCapitalColorString[zapcore.InfoLevel]
	}
	enc.AppendString(s)
}

func shortLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	s, ok := _levelToCapitalString[l]
	if !ok {
		s = _levelToCapitalString[zapcore.InfoLevel]
	}
	enc.AppendString(s)
}
