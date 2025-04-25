package artistlist

import (
    "encoding/json"
    "os"
    "path/filepath"
    "strings"
    "sync"

    "go.uber.org/zap"
)

// ArtistList manages the female and male whitelists
type ArtistList struct {
    female map[string]struct{}
    male   map[string]struct{}
    united map[string]struct{}
    mu     sync.RWMutex
    dir    string
    logger *zap.Logger
}

// NewArtistList creates a new ArtistList instance, loading whitelists from JSON files
func NewArtistList(dir string, logger *zap.Logger) (*ArtistList, error) {
    al := &ArtistList{
        female:  make(map[string]struct{}),
        male:    make(map[string]struct{}),
        united:  make(map[string]struct{}),
        dir:     dir,
        logger:  logger,
    }

    if err := al.loadFemaleWhitelist(); err != nil {
        return nil, err
    }
    if err := al.loadMaleWhitelist(); err != nil {
        return nil, err
    }
    al.updateUnited()
    return al, nil
}

// loadFemaleWhitelist loads the female whitelist from JSON file
func (al *ArtistList) loadFemaleWhitelist() error {
    path := filepath.Join(al.dir, "female_whitelist.json")
    data, err := os.ReadFile(path)
    if err != nil {
        al.logger.Error("Failed to read female whitelist", zap.Error(err))
        return err
    }

    var items []string
    if err := json.Unmarshal(data, &items); err != nil {
        al.logger.Error("Failed to unmarshal female whitelist", zap.Error(err))
        return err
    }

    al.mu.Lock()
    defer al.mu.Unlock()
    for _, item := range items {
        key := strings.ToLower(strings.TrimSpace(item))
        if key != "" {
            al.female[key] = struct{}{}
            al.united[key] = struct{}{}
        }
    }
    return nil
}

// loadMaleWhitelist loads the male whitelist from JSON file
func (al *ArtistList) loadMaleWhitelist() error {
    path := filepath.Join(al.dir, "male_whitelist.json")
    data, err := os.ReadFile(path)
    if err != nil {
        al.logger.Error("Failed to read male whitelist", zap.Error(err))
        return err
    }

    var items []string
    if err := json.Unmarshal(data, &items); err != nil {
        al.logger.Error("Failed to unmarshal male whitelist", zap.Error(err))
        return err
    }

    al.mu.Lock()
    defer al.mu.Unlock()
    for _, item := range items {
        key := strings.ToLower(strings.TrimSpace(item))
        if key != "" {
            al.male[key] = struct{}{}
            al.united[key] = struct{}{}
        }
    }
    return nil
}

// updateUnited updates the united whitelist
func (al *ArtistList) updateUnited() {
    al.mu.Lock()
    defer al.mu.Unlock()

    al.united = make(map[string]struct{})
    for k := range al.female {
        al.united[k] = struct{}{}
    }
    for k := range al.male {
        al.united[k] = struct{}{}
    }
}

// GetFemaleWhitelist returns the female whitelist
func (al *ArtistList) GetFemaleWhitelist() map[string]struct{} {
    al.mu.RLock()
    defer al.mu.RUnlock()

    result := make(map[string]struct{})
    for k := range al.female {
        result[k] = struct{}{}
    }
    return result
}

// GetMaleWhitelist returns the male whitelist
func (al *ArtistList) GetMaleWhitelist() map[string]struct{} {
    al.mu.RLock()
    defer al.mu.RUnlock()

    result := make(map[string]struct{})
    for k := range al.male {
        result[k] = struct{}{}
    }
    return result
}

// GetUnitedWhitelist returns the united whitelist
func (al *ArtistList) GetUnitedWhitelist() map[string]struct{} {
    al.mu.RLock()
    defer al.mu.RUnlock()

    result := make(map[string]struct{})
    for k := range al.united {
        result[k] = struct{}{}
    }
    return result
}

// AddArtist adds an artist to the specified whitelist and saves to JSON
func (al *ArtistList) AddArtist(artist string, isFemale bool) error {
    artist = strings.ToLower(strings.TrimSpace(artist))
    if artist == "" {
        return nil
    }

    al.mu.Lock()
    defer al.mu.Unlock()

    if isFemale {
        al.female[artist] = struct{}{}
    } else {
        al.male[artist] = struct{}{}
    }
    al.united[artist] = struct{}{}

    if err := al.saveFemaleWhitelist(); err != nil {
        return err
    }
    if err := al.saveMaleWhitelist(); err != nil {
        return err
    }
    al.logger.Info("Added artist", zap.String("artist", artist), zap.Bool("isFemale", isFemale))
    return nil
}

// AddArtists adds multiple artists to the specified whitelist and saves to JSON
func (al *ArtistList) AddArtists(artists []string, isFemale bool) (int, error) {
    al.mu.Lock()
    defer al.mu.Unlock()

    addedCount := 0
    for _, artist := range artists {
        artist = strings.ToLower(strings.TrimSpace(artist))
        if artist == "" {
            continue
        }

        if isFemale {
            if _, exists := al.female[artist]; !exists {
                al.female[artist] = struct{}{}
                al.united[artist] = struct{}{}
                addedCount++
            }
        } else {
            if _, exists := al.male[artist]; !exists {
                al.male[artist] = struct{}{}
                al.united[artist] = struct{}{}
                addedCount++
            }
        }
    }

    if addedCount == 0 {
        return 0, nil
    }

    if err := al.saveFemaleWhitelist(); err != nil {
        return addedCount, err
    }
    if err := al.saveMaleWhitelist(); err != nil {
        return addedCount, err
    }
    al.logger.Info("Added artists", zap.Int("count", addedCount), zap.Bool("isFemale", isFemale))
    return addedCount, nil
}

// RemoveArtist removes an artist from both whitelists and saves to JSON
func (al *ArtistList) RemoveArtist(artist string) error {
    artist = strings.ToLower(strings.TrimSpace(artist))
    if artist == "" {
        return nil
    }

    al.mu.Lock()
    defer al.mu.Unlock()

    deletedFemale := false
    deletedMale := false

    if _, exists := al.female[artist]; exists {
        delete(al.female, artist)
        deletedFemale = true
    }
    if _, exists := al.male[artist]; exists {
        delete(al.male, artist)
        deletedMale = true
    }
    delete(al.united, artist)

    if deletedFemale {
        if err := al.saveFemaleWhitelist(); err != nil {
            return err
        }
    }
    if deletedMale {
        if err := al.saveMaleWhitelist(); err != nil {
            return err
        }
    }
    al.logger.Info("Removed artist", zap.String("artist", artist))
    return nil
}

// RemoveArtists removes multiple artists from both whitelists and saves to JSON
func (al *ArtistList) RemoveArtists(artists []string) (int, error) {
    al.mu.Lock()
    defer al.mu.Unlock()

    removedCount := 0
    for _, artist := range artists {
        artist = strings.ToLower(strings.TrimSpace(artist))
        if artist == "" {
            continue
        }

        deletedFemale := false
        deletedMale := false

        if _, exists := al.female[artist]; exists {
            delete(al.female, artist)
            deletedFemale = true
            removedCount++
        }
        if _, exists := al.male[artist]; exists {
            delete(al.male, artist)
            deletedMale = true
            removedCount++
        }
        if deletedFemale || deletedMale {
            delete(al.united, artist)
        }
    }

    if removedCount == 0 {
        return 0, nil
    }

    if err := al.saveFemaleWhitelist(); err != nil {
        return removedCount, err
    }
    if err := al.saveMaleWhitelist(); err != nil {
        return removedCount, err
    }
    al.logger.Info("Removed artists", zap.Int("count", removedCount))
    return removedCount, nil
}

// ClearWhitelists clears both whitelists and saves to JSON
func (al *ArtistList) ClearWhitelists() error {
    al.mu.Lock()
    defer al.mu.Unlock()

    al.female = make(map[string]struct{})
    al.male = make(map[string]struct{})
    al.united = make(map[string]struct{})

    if err := al.saveFemaleWhitelist(); err != nil {
        return err
    }
    if err := al.saveMaleWhitelist(); err != nil {
        return err
    }
    al.logger.Info("Cleared whitelists")
    return nil
}

// saveFemaleWhitelist saves the female whitelist to JSON
func (al *ArtistList) saveFemaleWhitelist() error {
    path := filepath.Join(al.dir, "female_whitelist.json")
    var items []string
    for k := range al.female {
        items = append(items, k)
    }

    data, err := json.MarshalIndent(items, "", "  ")
    if err != nil {
        al.logger.Error("Failed to marshal female whitelist", zap.Error(err))
        return err
    }

    if err := os.WriteFile(path, data, 0644); err != nil {
        al.logger.Error("Failed to write female whitelist", zap.Error(err))
        return err
    }
    return nil
}

// saveMaleWhitelist saves the male whitelist to JSON
func (al *ArtistList) saveMaleWhitelist() error {
    path := filepath.Join(al.dir, "male_whitelist.json")
    var items []string
    for k := range al.male {
        items = append(items, k)
    }

    data, err := json.MarshalIndent(items, "", "  ")
    if err != nil {
        al.logger.Error("Failed to marshal male whitelist", zap.Error(err))
        return err
    }

    if err := os.WriteFile(path, data, 0644); err != nil {
        al.logger.Error("Failed to write male whitelist", zap.Error(err))
        return err
    }
    return nil
}