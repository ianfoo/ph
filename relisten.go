package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const relistenArtistsCacheFile = "relisten-artists.json"

// relistenArtist describes part of the entries that are returned
// from Relisten's artists API. There is much more data contained
// in the response, but we are only concerned with the artist name
// and the "slug" which is used in building a URL for a particular
// artist on Relisten. E.g., For the artist "Umphrey's McGee" the
// slug is "umphreys", and the resultant Relisten URL would be
// https://relisten.net/umphreys/...
type relistenArtist struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// relistenGetArtists fetches the list of artists available on Relisten from
// either a local cache or the Relisten artists API and returns a map from the
// readable name to the "slug" used in the Relisten URL.
func relistenGetArtists(client *http.Client) (map[string]string, error) {
	var artistsList []relistenArtist
	cachePath, err := relistenArtistsCachePath()
	if err != nil {
		// TODO Fall through to API fetch
		return nil, err
	}
	cacheFile, err := relistenGetArtistsCache(cachePath)
	if err != nil {
		return nil, err
	}
	if cacheFile != nil {
		defer cacheFile.Close()
		if err := json.NewDecoder(cacheFile).Decode(&artistsList); err != nil {
			log.Printf("warning: cannot decode Relisten artists cache: %v", err)
		}
		if len(artistsList) > 0 {
			return relistenMakeArtistsMap(artistsList), nil
		}
	}
	apiRespBody, err := relistenFetchArtists(client)
	if err != nil {
		return nil, err
	}
	defer apiRespBody.Close()
	if err := json.NewDecoder(apiRespBody).Decode(&artistsList); err != nil {
		return nil, err
	}
	if err := relistenWriteAristsCache(cachePath, artistsList); err != nil {
		log.Printf("warning: could not write Relisten artists cache: %v", err)
	}
	return relistenMakeArtistsMap(artistsList), nil
}

// relistenFetchArtists gets the list of artists that Relisten supports from
// the Relisten artists API.
func relistenFetchArtists(client *http.Client) (io.ReadCloser, error) {
	const relistenArtistsAPI = "https://api.relisten.net/api/v2/artists"
	resp, err := client.Get(relistenArtistsAPI)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// relistenGetArtistsCache returns an io.ReadCloser for the local Relisten
// artists cache, if it exists and if it has been modified within the last
// week. If it doesn't exist or is older than one week, a nil ReadCloser is
// returned. This is simpler than creating a sentinel error that must be
// interpreted by the caller, rather allowing it to just check for nil and look
// elsewhere for Relisten artists.
func relistenGetArtistsCache(path string) (io.ReadCloser, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if invalidateBefore := time.Now().Add(-7 * 24 * time.Hour); info.ModTime().Before(invalidateBefore) {
		return nil, nil
	}
	return os.Open(path)
}

func relistenWriteAristsCache(path string, artistsList []relistenArtist) error {
	if err := os.MkdirAll(filepath.Dir(path), os.FileMode(0777)); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(artistsList)
}

func relistenArtistsCachePath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(cacheDir, "ph", relistenArtistsCacheFile)
	return p, nil
}

func relistenMakeArtistsMap(artistsList []relistenArtist) map[string]string {
	artists := make(map[string]string, len(artistsList))
	for _, a := range artistsList {
		artists[a.Name] = a.Slug
	}
	return artists
}
