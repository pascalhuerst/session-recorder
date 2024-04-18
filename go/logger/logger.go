package logger

import (
	"os"
	"strconv"
	"strings"
	"time"

	colorable "github.com/mattn/go-colorable"
	isatty "github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

const callerMarshallerPathParts = 2

// Setup initializes the zerolog Logger based on environment variables
//
// `RECORDER_LOG_LEVEL` sets the log level. Supported values are `debug`, `info`,
// `warn`, `error`, `panic` and `fatal`. Defaults to `info`.
//
// If `RECORDER_LOG_FORMAT` is set to `json`, the output format will be JSON, and the
// timestamp is reported in Unix epoch with microsecond precision.
//
// For other output formats than JSON, the variable `RECORDER_LOG_COLOR` is parsed.
// If it is set to `on`, colors are enabled. If set to `auto`, colors are enabled in
// case stdout is connected to a TTY. Defaults to `auto`.
func Setup() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro

	if strings.ToLower(os.Getenv("RECORDER_LOG_FORMAT")) != "json" {
		consoleWriter := zerolog.ConsoleWriter{
			TimeFormat: time.StampMicro,
			Out:        os.Stdout,
		}

		s := strings.ToLower(os.Getenv("RECORDER_LOG_COLOR"))
		if s == "" {
			s = "auto"
		}

		if s == "on" || (s == "auto" && isatty.IsTerminal(os.Stdout.Fd())) {
			consoleWriter.Out = colorable.NewColorableStdout()
		} else {
			consoleWriter.NoColor = true
		}

		log.Logger = log.Output(consoleWriter)
	}

	level := zerolog.InfoLevel
	s := strings.ToLower(os.Getenv("RECORDER_LOG_LEVEL"))

	switch s {
	case "debug":
		level = zerolog.DebugLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	case "fatal":
		level = zerolog.FatalLevel
	case "panic":
		level = zerolog.PanicLevel
	}

	zerolog.SetGlobalLevel(level)

	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		parts := strings.Split(file, "/")

		if len(parts) > callerMarshallerPathParts {
			parts = parts[len(parts)-callerMarshallerPathParts:]
		}

		return strings.Join(parts, "/") + ":" + strconv.Itoa(line)
	}

	log.Logger = log.With().Caller().Logger()

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack //nolint:reassign
}
