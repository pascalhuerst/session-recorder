package utils

import (
	"os"
	"testing"
)

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
