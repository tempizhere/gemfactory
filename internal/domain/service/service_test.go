package service

import (
	"testing"

	"go.uber.org/zap"
)

func TestFormatDate(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name        string
		input       string
		expected    string
		shouldError bool
	}{
		{
			name:        "Дата с временем KST",
			input:       "June 16, 2025 at 0 AM KST",
			expected:    "16.06.25",
			shouldError: false,
		},
		{
			name:        "Дата с временем KST (другой формат)",
			input:       "January 15, 2025 at 11 PM KST",
			expected:    "15.01.25",
			shouldError: false,
		},
		{
			name:        "Обычная дата с запятой",
			input:       "June 16, 2025",
			expected:    "16.06.25",
			shouldError: false,
		},
		{
			name:        "Дата без года",
			input:       "June 16",
			expected:    "16.06.25", // Текущий год
			shouldError: false,
		},
		{
			name:        "Пустая строка",
			input:       "",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "Невалидная дата",
			input:       "Not a date",
			expected:    "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatDate(tt.input, logger)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for input %q, but got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for input %q, but got: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("Expected %q for input %q, but got %q", tt.expected, tt.input, result)
				}
			}
		})
	}
}
