package release

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

// DateParseFormat is the format for parsing dates from the website
const DateParseFormat = "January 2, 2006"

// TimeFormat is the output format for times
const TimeFormat = "15:04"

// TimeParseFormat is the format for parsing times from the website
const TimeParseFormat = "3 PM"

// KSTToMSKDiff is the time difference between KST and MSK (KST is 6 hours ahead)
const KSTToMSKDiff = -6 * time.Hour

// CurrentYear returns the current year
func CurrentYear() string {
	return time.Now().Format("2006")
}
