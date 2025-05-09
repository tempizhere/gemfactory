package release

import (
	"strings"
	"time"
)

// Config holds configuration constants for release processing
type Config struct {
	months          []string
	monthToNumber   map[string]string
	dateFormat      string
	dateParseFormat string
	timeFormat      string
	timeParseFormat string
	kstToMSKDiff    time.Duration
}

// NewConfig creates a new Config instance
func NewConfig() *Config {
	return &Config{
		months: []string{
			"january", "february", "march", "april", "may", "june",
			"july", "august", "september", "october", "november", "december",
		},
		monthToNumber: map[string]string{
			"january":   "01",
			"february":  "02",
			"march":     "03",
			"april":     "04",
			"may":       "05",
			"june":      "06",
			"july":      "07",
			"august":    "08",
			"september": "09",
			"october":   "10",
			"november":  "11",
			"december":  "12",
		},
		dateFormat:      "02.01.06",
		dateParseFormat: "January 2, 2006",
		timeFormat:      "15:04",
		timeParseFormat: "3 PM",
		kstToMSKDiff:    -6 * time.Hour,
	}
}

// Months returns the list of month names
func (c *Config) Months() []string {
	result := make([]string, len(c.months))
	copy(result, c.months)
	return result
}

// MonthToNumber returns the numeric representation of a month
func (c *Config) MonthToNumber(month string) (string, bool) {
	num, ok := c.monthToNumber[strings.ToLower(month)]
	return num, ok
}

// DateFormat returns the output format for dates
func (c *Config) DateFormat() string {
	return c.dateFormat
}

// DateParseFormat returns the format for parsing dates
func (c *Config) DateParseFormat() string {
	return c.dateParseFormat
}

// TimeFormat returns the output format for times
func (c *Config) TimeFormat() string {
	return c.timeFormat
}

// TimeParseFormat returns the format for parsing times
func (c *Config) TimeParseFormat() string {
	return c.timeParseFormat
}

// KSTToMSKDiff returns the time difference between KST and MSK
func (c *Config) KSTToMSKDiff() time.Duration {
	return c.kstToMSKDiff
}
