package release

// Release represents a K-pop release event
type Release struct {
	Date       string `json:"release_date"`
	TimeMSK    string `json:"time_msk"`
	Artist     string `json:"artist"`
	AlbumName  string `json:"album_name"`
	TitleTrack string `json:"title_track"`
	MV         string `json:"mv"`
}
