// Package http provides a repository implementation for HTTP.
package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/WithoutPants/pakman/pkg/pak"
	"github.com/WithoutPants/pakman/pkg/pak/yaml"
)

const (
	IndexPath    = "index.yml"
	ManifestPath = "manifest.yml"
)

// Repository is a HTTP based repository.
// The index is stored at index.yml in the root of the repository. For example, if the BaseURL is https://example.com/paks, the index file located at https://example.com/paks/index.yml.
//
// Manifest files are stored at <BaseURL>/<id>/<version>/manifest.yml. Using the above BaseURL, the manifest for version "1.0.0" or pak "widget" would be located at https://example.com/paks/widget/1.0.0/manifest.yml.
//
// Pak files are stored in the same location as the manifest file.
//
// The index is cached for the duration of CacheTTL. The first request after the cache expires will cause the index to be reloaded.
type Repository struct {
	BaseURL url.URL
	Client  *http.Client

	// CacheTTL is the time to live for the index cache.
	// The index is cached for this duration. The first request after the cache
	// expires will cause the index to be reloaded.
	CacheTTL time.Duration

	cachedIndex *pak.SpecIndex
	cacheTime   time.Time
}

// New creates a new Repository. If client is nil then http.DefaultClient is used.
func New(baseURL url.URL, client *http.Client) *Repository {
	if client == nil {
		client = http.DefaultClient
	}
	return &Repository{
		BaseURL: baseURL,
		Client:  client,
	}
}

// GetManifest gets the manifest for the given id and version.
func (r *Repository) GetManifest(ctx context.Context, id string, version string) (*pak.Manifest, error) {
	f, err := r.getFile(ctx, r.manifestPath(id, version))
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest file: %w", err)
	}

	defer f.Close()

	manifest, err := yaml.ReadManifest(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	if version != "" && version != manifest.Version {
		return nil, nil
	}

	return manifest, nil
}

func (r *Repository) manifestPath(id string, version string) url.URL {
	u := r.BaseURL
	u.Path, _ = url.JoinPath(u.Path, id, version, ManifestPath)
	return u
}

// GetSpec gets the spec for the given id and version.
// If version is empty then the latest version is returned.
func (r *Repository) GetSpec(ctx context.Context, id string) (*pak.Spec, error) {
	index, err := r.getIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get index: %w", err)
	}

	spec, ok := index[id]
	if !ok {
		return nil, nil
	}

	return &spec, nil
}

// List returns all specs in the repository.
func (r *Repository) List(ctx context.Context) (pak.SpecIndex, error) {
	return r.getIndex(ctx)
}

func (r *Repository) checkCacheExpired() {
	if r.cachedIndex == nil {
		return
	}

	if time.Since(r.cacheTime) > r.CacheTTL {
		r.cachedIndex = nil
	}
}

func (r *Repository) getIndex(ctx context.Context) (pak.SpecIndex, error) {
	r.checkCacheExpired()

	if r.cachedIndex != nil {
		return *r.cachedIndex, nil
	}

	u := r.BaseURL
	var err error
	u.Path, err = url.JoinPath(u.Path, IndexPath)
	if err != nil {
		// shouldn't happen
		return nil, err
	}

	f, err := r.getFile(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("failed to get index file: %w", err)
	}

	defer f.Close()

	index, err := yaml.ReadSpecIndex(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read index file: %w", err)
	}

	r.cachedIndex = index
	r.cacheTime = time.Now()

	return *index, nil
}

func (r *Repository) getFile(ctx context.Context, u url.URL) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		// shouldn't happen
		return nil, err
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote file: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to get remote file: %s", resp.Status)
	}

	return resp.Body, nil
}

// GetFile gets the file for the given id, version and file.
func (r *Repository) GetFile(ctx context.Context, id string, version string, file string) (io.ReadCloser, error) {
	f, err := r.getFile(ctx, r.filePath(id, version, file))
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	return f, nil
}

func (r *Repository) filePath(id string, version string, file string) url.URL {
	u := r.BaseURL
	u.Path, _ = url.JoinPath(u.Path, id, version, file)
	return u
}
