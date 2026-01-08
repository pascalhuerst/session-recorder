package logger

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
)

/**
 * Test Plan: Logger Setup
 *
 * Scenario: Default log level is info
 *   Given no RECORDER_LOG_LEVEL environment variable
 *   When Setup() is called
 *   Then global log level is set to InfoLevel
 *
 * Scenario: Log level from environment variable
 *   Given RECORDER_LOG_LEVEL is set to "debug"/"warn"/"error"
 *   When Setup() is called
 *   Then global log level is set to the corresponding level
 *
 * Scenario: JSON format configuration
 *   Given RECORDER_LOG_FORMAT is set to "json"
 *   When Setup() is called
 *   Then time format is set for JSON output
 *
 * Scenario: Console format configuration
 *   Given RECORDER_LOG_FORMAT is set to "console"
 *   When Setup() is called
 *   Then console writer is configured (no panic)
 *
 * Scenario: Color configuration options
 *   Given RECORDER_LOG_COLOR is set to "on"/"off"/"auto"
 *   When Setup() is called
 *   Then appropriate color settings are applied (no panic)
 */

func clearEnvVars() {
	os.Unsetenv("RECORDER_LOG_LEVEL")
	os.Unsetenv("RECORDER_LOG_FORMAT")
	os.Unsetenv("RECORDER_LOG_COLOR")
}

func TestSetup_DefaultLevel(t *testing.T) {
	clearEnvVars()
	defer clearEnvVars()

	Setup()

	if zerolog.GlobalLevel() != zerolog.InfoLevel {
		t.Errorf("Setup() default level = %v, want %v", zerolog.GlobalLevel(), zerolog.InfoLevel)
	}
}

func TestSetup_DebugLevel(t *testing.T) {
	clearEnvVars()
	defer clearEnvVars()

	os.Setenv("RECORDER_LOG_LEVEL", "debug")

	Setup()

	if zerolog.GlobalLevel() != zerolog.DebugLevel {
		t.Errorf("Setup() level = %v, want %v", zerolog.GlobalLevel(), zerolog.DebugLevel)
	}
}

func TestSetup_WarnLevel(t *testing.T) {
	clearEnvVars()
	defer clearEnvVars()

	os.Setenv("RECORDER_LOG_LEVEL", "warn")

	Setup()

	if zerolog.GlobalLevel() != zerolog.WarnLevel {
		t.Errorf("Setup() level = %v, want %v", zerolog.GlobalLevel(), zerolog.WarnLevel)
	}
}

func TestSetup_ErrorLevel(t *testing.T) {
	clearEnvVars()
	defer clearEnvVars()

	os.Setenv("RECORDER_LOG_LEVEL", "error")

	Setup()

	if zerolog.GlobalLevel() != zerolog.ErrorLevel {
		t.Errorf("Setup() level = %v, want %v", zerolog.GlobalLevel(), zerolog.ErrorLevel)
	}
}

func TestSetup_JSONFormat(t *testing.T) {
	clearEnvVars()
	defer clearEnvVars()

	os.Setenv("RECORDER_LOG_FORMAT", "json")

	// Should not panic
	Setup()

	// Verify time format is set for JSON
	if zerolog.TimeFieldFormat != zerolog.TimeFormatUnixMicro {
		t.Errorf("Setup() TimeFieldFormat = %v, want %v", zerolog.TimeFieldFormat, zerolog.TimeFormatUnixMicro)
	}
}

func TestSetup_ConsoleFormat(t *testing.T) {
	clearEnvVars()
	defer clearEnvVars()

	os.Setenv("RECORDER_LOG_FORMAT", "console")

	// Should not panic
	Setup()

	if zerolog.GlobalLevel() != zerolog.InfoLevel {
		t.Errorf("Setup() level = %v, want %v", zerolog.GlobalLevel(), zerolog.InfoLevel)
	}
}

func TestSetup_ColorOff(t *testing.T) {
	clearEnvVars()
	defer clearEnvVars()

	os.Setenv("RECORDER_LOG_COLOR", "off")

	// Should not panic
	Setup()
}

func TestSetup_ColorOn(t *testing.T) {
	clearEnvVars()
	defer clearEnvVars()

	os.Setenv("RECORDER_LOG_COLOR", "on")

	// Should not panic
	Setup()
}

func TestSetup_ColorAuto(t *testing.T) {
	clearEnvVars()
	defer clearEnvVars()

	os.Setenv("RECORDER_LOG_COLOR", "auto")

	// Should not panic
	Setup()
}
