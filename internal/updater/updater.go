//go:build windows

// Package updater queries the latest GitHub release of TurboKey and applies
// in-place self-updates (download the new exe, swap, relaunch).
package updater

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"turbokey/internal/buildinfo"
)

const (
	apiURL    = "https://api.github.com/repos/houko/turbokey/releases/latest"
	assetName = "turbokey.exe"
)

// userAgent identifies the client to GitHub's API. GitHub returns 403 for the
// default Go-http-client UA, so we must send something app-specific.
func userAgent() string { return "TurboKey/" + buildinfo.Version }

// Release is the trimmed view of a GitHub release we care about.
type Release struct {
	Tag      string // e.g. "v1.0.9"
	URL      string // HTML release page
	AssetURL string // direct download URL for turbokey.exe (empty if not found)
	Size     int64  // asset size in bytes (0 if unknown)
}

// Latest fetches the latest GitHub release.
func Latest() (*Release, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent())
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
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
			Size int64  `json:"size"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, err
	}
	r := &Release{Tag: v.TagName, URL: v.HTMLURL}
	for _, a := range v.Assets {
		if strings.EqualFold(a.Name, assetName) {
			r.AssetURL = a.URL
			r.Size = a.Size
			break
		}
	}
	return r, nil
}

// progressReader wraps an io.Reader to report bytes read to a callback.
type progressReader struct {
	r        io.Reader
	total    int64
	read     int64
	onChange func(read, total int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.read += int64(n)
		if p.onChange != nil {
			p.onChange(p.read, p.total)
		}
	}
	return n, err
}

// Download streams the asset to dest. onProgress (if non-nil) is called from
// the download goroutine with the running byte count; it must be safe to call
// from a non-UI thread.
func Download(url, dest string, onProgress func(read, total int64)) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent())
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download: %s", resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	pr := &progressReader{r: resp.Body, total: resp.ContentLength, onChange: onProgress}
	if _, err := io.Copy(f, pr); err != nil {
		return err
	}
	return f.Sync()
}

// SwapAndRelaunch installs the freshly downloaded exe at newExe over the
// running executable and starts the new instance. The current process must
// exit immediately after this returns nil — the caller does that.
//
// The swap uses two renames on the same volume:
//  1. running turbokey.exe → turbokey.exe.old (allowed for a running image)
//  2. newExe → turbokey.exe
//
// CleanupOld() removes the .old file once the parent has fully exited.
//
// preExit (optional) runs after the swap succeeds but before the new process
// starts. Use it to release process-wide locks (e.g. the singleton mutex).
func SwapAndRelaunch(newExe string, preExit func()) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	old := self + ".old"
	// Best-effort: remove any leftover .old from a prior update before renaming.
	_ = os.Remove(old)
	if err := os.Rename(self, old); err != nil {
		return fmt.Errorf("rename self: %w", err)
	}
	if err := os.Rename(newExe, self); err != nil {
		// Try to roll back so the user isn't left with no exe at all.
		_ = os.Rename(old, self)
		return fmt.Errorf("install new: %w", err)
	}
	if preExit != nil {
		preExit()
	}
	cmd := exec.Command(self)
	cmd.Dir = filepath.Dir(self)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch new: %w", err)
	}
	return nil
}

// CleanupOld removes the leftover *.old file from a previous self-update.
// Runs asynchronously with a short retry loop because the previous process
// may still be exiting and holding the image open for a few ms.
func CleanupOld() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	old := exe + ".old"
	go func() {
		for i := 0; i < 20; i++ {
			if err := os.Remove(old); err == nil || os.IsNotExist(err) {
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()
}
