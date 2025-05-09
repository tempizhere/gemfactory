package service

// ReleaseServiceInterface defines the interface for release operations
type ReleaseServiceInterface interface {
	GetReleasesForMonth(month string, femaleOnly, maleOnly bool) (string, error)
	ClearCache()
}

// ArtistServiceInterface defines the interface for artist operations
type ArtistServiceInterface interface {
	ParseArtists(input string) []string
	FormatWhitelists() string
	FormatWhitelistsForExport() string
	AddArtists(artists []string, isFemale bool) (int, error)
	RemoveArtists(artists []string) (int, error)
	ClearWhitelists() error
}
