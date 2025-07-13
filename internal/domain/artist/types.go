// Package artist содержит типы и логику для работы с артистами.
package artist

// WhitelistManager defines the interface for managing artist whitelists
type WhitelistManager interface {
	// GetFemaleWhitelist returns the list of female artists
	GetFemaleWhitelist() []string
	// GetMaleWhitelist returns the list of male artists
	GetMaleWhitelist() []string
	// GetUnitedWhitelist returns the combined list of all artists
	GetUnitedWhitelist() []string
	// AddArtist adds a single artist to the specified whitelist
	AddArtist(artist string, isFemale bool) error
	// AddArtists adds multiple artists to the specified whitelist
	AddArtists(artists []string, isFemale bool) (int, error)
	// RemoveArtist removes a single artist from both whitelists
	RemoveArtist(artist string) error
	// RemoveArtists removes multiple artists from both whitelists
	RemoveArtists(artists []string) (int, error)
	// ClearWhitelists clears both whitelists
	ClearWhitelists() error
}
