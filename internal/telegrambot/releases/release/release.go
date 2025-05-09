package release

import "time"

// CurrentYear returns the current year
func CurrentYear() string {
	return time.Now().Format("2006")
}
