package models

import "time"

// Release represents a K-pop release event
type Release struct {
	Date       string `json:"release_date"`
	TimeMSK    string `json:"time_msk"`
	Artist     string `json:"artist"`
	AlbumName  string `json:"album_name"`
	TitleTrack string `json:"title_track"`
	MV         string `json:"mv"`
}

// Months is a list of month names
var Months = []string{
	"january", "february", "march", "april", "may", "june",
	"july", "august", "september", "october", "november", "december",
}

// MonthToNumber maps month names to numbers
var MonthToNumber = map[string]string{
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
}

// DateFormat is the output format for dates
const DateFormat = "02.01.06"

// TimeFormat is the output format for times
const TimeFormat = "15:04"

// CurrentYear returns the current year
func CurrentYear() string {
	return time.Now().Format("2006")
}
