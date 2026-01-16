package utils

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

var ErrVarNotSet = errors.New("environment variable not set")

func MustGet(name string) string {
	var (
		err error
		v   string
	)

	if v, err = Get(name); err != nil {
		log.Fatal().Str("variable", name).Err(err).Msg("variable must be set")
	}

	return v
}

func Get(name string) (string, error) {
	s := ""
	set := false

	if s, set = os.LookupEnv(name); !set || s == "" {
		return s, ErrVarNotSet
	}

	return strings.Clone(s), nil
}

func GetWithDefault(name, defaultValue string) string {
	if v, err := Get(name); err == nil {
		return v
	}
	return defaultValue
}

// GetEnvOrDefault is an alias for GetWithDefault for backwards compatibility
func GetEnvOrDefault(name string, defaultValue string) string {
	return GetWithDefault(name, defaultValue)
}

func GetInt(name string) (int, error) {
	s, err := Get(name)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(s)
}

func GetIntWithDefault(name string, defaultValue int) int {
	if v, err := GetInt(name); err == nil {
		return v
	}
	return defaultValue
}
