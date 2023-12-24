package utils

import (
	"errors"
	"os"
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
