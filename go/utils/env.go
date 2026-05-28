package utils

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

var ErrVarNotSet = errors.New("environment variable not set")

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

func GetDuration(name string) (time.Duration, error) {
	s, err := Get(name)
	if err != nil {
		return 0, err
	}
	return time.ParseDuration(strings.TrimSpace(s))
}

func GetDurationWithDefault(name string, defaultValue time.Duration) time.Duration {
	if v, err := GetDuration(name); err == nil {
		return v
	}
	return defaultValue
}

func GetBoolWithDefault(name string, defaultValue bool) bool {
	v, err := Get(name)
	if err != nil {
		return defaultValue
	}
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return defaultValue
	}
	return b
}
