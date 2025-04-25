package html

import (
    "html"
)

// Escape escapes special HTML characters in a string.
func Escape(s string) string {
    return html.EscapeString(s)
}