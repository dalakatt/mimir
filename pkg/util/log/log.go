// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/cortexproject/cortex/blob/master/pkg/util/log/log.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Cortex Authors.

package log

import (
	"fmt"
	"os"

	"github.com/go-kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/weaveworks/common/logging"
	"github.com/weaveworks/common/server"
)

var (
	// Logger is a shared go-kit logger.
	// TODO: Change all components to take a non-global logger via their constructors.
	// Prefer accepting a non-global logger as an argument.
	Logger = kitlog.NewNopLogger()
)

// InitLogger initialises the global gokit logger (util_log.Logger) and overrides the
// default logger for the server.
func InitLogger(cfg *server.Config) {
	l, err := NewPrometheusLogger(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		panic(err)
	}

	// when use util_log.Logger, skip 3 stack frames.
	Logger = log.With(l, "caller", log.Caller(3))

	// cfg.Log wraps log function, skip 4 stack frames to get caller information.
	// this works in go 1.12, but doesn't work in versions earlier.
	// it will always shows the wrapper function generated by compiler
	// marked <autogenerated> in old versions.
	cfg.Log = logging.GoKit(log.With(l, "caller", log.Caller(4)))
}

// PrometheusLogger exposes Prometheus counters for each of go-kit's log levels.
type PrometheusLogger struct {
	logger log.Logger
}

// NewPrometheusLogger creates a new instance of PrometheusLogger which exposes
// Prometheus counters for various log levels.
func NewPrometheusLogger(l logging.Level, format logging.Format) (log.Logger, error) {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	if format.String() == "json" {
		logger = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
	}
	logger = level.NewFilter(logger, logFilter(l.String()))

	logger = &PrometheusLogger{
		logger: logger,
	}

	// return a Logger without caller information, shouldn't use directly
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	return logger, nil
}

// Log increments the appropriate Prometheus counter depending on the log level.
func (pl *PrometheusLogger) Log(kv ...interface{}) error {
	return pl.logger.Log(kv...)
}

// CheckFatal prints an error and exits with error code 1 if err is non-nil
func CheckFatal(location string, err error) {
	if err != nil {
		logger := level.Error(Logger)
		if location != "" {
			logger = log.With(logger, "msg", "error "+location)
		}
		// %+v gets the stack trace from errors using github.com/pkg/errors
		logger.Log("err", fmt.Sprintf("%+v", err))
		os.Exit(1)
	}
}

// TODO(dannyk): remove once weaveworks/common updates to go-kit/log
// 					-> we can then revert to using Level.Gokit
func logFilter(l string) level.Option {
	switch l {
	case "debug":
		return level.AllowDebug()
	case "info":
		return level.AllowInfo()
	case "warn":
		return level.AllowWarn()
	case "error":
		return level.AllowError()
	default:
		return level.AllowAll()
	}
}
