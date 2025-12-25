package utils

import (
	"os"
	"testing"
)

/**
 * Test Plan: Environment Variable Utilities
 *
 * Scenario: Get existing environment variable
 *   Given an environment variable is set with a value
 *   When Get() is called with that variable name
 *   Then the value is returned with no error
 *
 * Scenario: Get non-existing environment variable
 *   Given an environment variable is not set
 *   When Get() is called with that variable name
 *   Then an empty string and ErrVarNotSet error are returned
 *
 * Scenario: Get empty environment variable
 *   Given an environment variable is set to empty string
 *   When Get() is called with that variable name
 *   Then an empty string and ErrVarNotSet error are returned
 *
 * Scenario: MustGet existing variable
 *   Given an environment variable is set with a value
 *   When MustGet() is called with that variable name
 *   Then the value is returned (no panic)
 */

func TestGet_ExistingVar(t *testing.T) {
	const testVar = "TEST_GET_EXISTING_VAR"
	const testValue = "test_value"

	os.Setenv(testVar, testValue)
	defer os.Unsetenv(testVar)

	got, err := Get(testVar)
	if err != nil {
		t.Errorf("Get() error = %v, want nil", err)
	}
	if got != testValue {
		t.Errorf("Get() = %v, want %v", got, testValue)
	}
}

func TestGet_NonExistingVar(t *testing.T) {
	const testVar = "TEST_GET_NON_EXISTING_VAR"

	// Ensure the variable is not set
	os.Unsetenv(testVar)

	got, err := Get(testVar)
	if err != ErrVarNotSet {
		t.Errorf("Get() error = %v, want %v", err, ErrVarNotSet)
	}
	if got != "" {
		t.Errorf("Get() = %v, want empty string", got)
	}
}

func TestGet_EmptyVar(t *testing.T) {
	const testVar = "TEST_GET_EMPTY_VAR"

	os.Setenv(testVar, "")
	defer os.Unsetenv(testVar)

	got, err := Get(testVar)
	if err != ErrVarNotSet {
		t.Errorf("Get() error = %v, want %v", err, ErrVarNotSet)
	}
	if got != "" {
		t.Errorf("Get() = %v, want empty string", got)
	}
}

func TestMustGet_ExistingVar(t *testing.T) {
	const testVar = "TEST_MUST_GET_EXISTING_VAR"
	const testValue = "test_value"

	os.Setenv(testVar, testValue)
	defer os.Unsetenv(testVar)

	got := MustGet(testVar)
	if got != testValue {
		t.Errorf("MustGet() = %v, want %v", got, testValue)
	}
}
