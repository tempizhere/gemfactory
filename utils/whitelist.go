package utils

import (
	"os"
	"strings"
)

// LoadFemaleWhitelist loads only the female whitelist
func LoadFemaleWhitelist() map[string]struct{} {
	femaleWhitelist := os.Getenv("female_whitelist")
	items := strings.Split(femaleWhitelist, ",")
	whitelist := make(map[string]struct{})
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item))
		if key != "" {
			whitelist[key] = struct{}{}
		}
	}
	return whitelist
}

// LoadMaleWhitelist loads only the male whitelist
func LoadMaleWhitelist() map[string]struct{} {
	maleWhitelist := os.Getenv("male_whitelist")
	items := strings.Split(maleWhitelist, ",")
	whitelist := make(map[string]struct{})
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item))
		if key != "" {
			whitelist[key] = struct{}{}
		}
	}
	return whitelist
}

// LoadWhitelist loads both female and male whitelists
func LoadWhitelist(femaleOnly bool) map[string]struct{} {
	femaleWhitelist := os.Getenv("female_whitelist")
	var raw string
	if femaleOnly {
		raw = femaleWhitelist
	} else {
		maleWhitelist := os.Getenv("male_whitelist")
		if maleWhitelist != "" {
			raw = femaleWhitelist + "," + maleWhitelist
		} else {
			raw = femaleWhitelist
		}
	}
	items := strings.Split(raw, ",")
	whitelist := make(map[string]struct{})
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item))
		if key != "" {
			whitelist[key] = struct{}{}
		}
	}
	return whitelist
}
