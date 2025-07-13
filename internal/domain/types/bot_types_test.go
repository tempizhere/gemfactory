package types

import (
	"errors"
	"testing"
)

func TestBotError(t *testing.T) {
	err := NewBotError("TEST_CODE", "test message", errors.New("cause"))

	if err.Code != "TEST_CODE" {
		t.Errorf("Expected code 'TEST_CODE', got '%s'", err.Code)
	}

	if err.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", err.Message)
	}

	if err.Unwrap() == nil {
		t.Error("Expected unwrapped error, got nil")
	}

	expected := "TEST_CODE: test message (cause)"
	if err.Error() != expected {
		t.Errorf("Expected error string '%s', got '%s'", expected, err.Error())
	}
}

func TestCommandError(t *testing.T) {
	err := NewCommandError("test_command", 123, 456, errors.New("test error"))

	if err.Command != "test_command" {
		t.Errorf("Expected command 'test_command', got '%s'", err.Command)
	}

	if err.UserID != 123 {
		t.Errorf("Expected user ID 123, got %d", err.UserID)
	}

	if err.ChatID != 456 {
		t.Errorf("Expected chat ID 456, got %d", err.ChatID)
	}

	expected := "command test_command failed for user 123 in chat 456: test error"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestIsCommandError(t *testing.T) {
	err := NewCommandError("test", 123, 456, errors.New("test"))

	if !IsCommandError(err) {
		t.Error("Expected IsCommandError to return true for CommandError")
	}

	if IsCommandError(errors.New("regular error")) {
		t.Error("Expected IsCommandError to return false for regular error")
	}
}

func TestIsBotError(t *testing.T) {
	err := NewBotError("TEST", "test", nil)

	if !IsBotError(err) {
		t.Error("Expected IsBotError to return true for BotError")
	}

	if IsBotError(errors.New("regular error")) {
		t.Error("Expected IsBotError to return false for regular error")
	}
}
