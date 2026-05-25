//go:build windows

// Package updater queries the latest GitHub release of TurboKey, used by the
// tray's "Check for updates" action.
package updater

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

const apiURL = "https://api.github.com/repos/houko/turbokey/releases/latest"

// Release is the trimmed view of a GitHub release we care about.
type Release struct {
	Tag string
	URL string
}

// Latest fetches the latest GitHub release. Returns an error if the network
// fails or the repo is unreachable (e.g. private repo with no auth).
func Latest() (*Release, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}
	var v struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, err
	}
	return &Release{Tag: v.TagName, URL: v.HTMLURL}, nil
}
