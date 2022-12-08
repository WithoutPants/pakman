// Package fs implements a file system based repository.
package fs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/WithoutPants/pakman/pkg/pak"
	"github.com/WithoutPants/pakman/pkg/pak/yaml"
)

const (
	IndexPath          = "index.yml"
	ManifestPath       = "manifest"
	RemoteManifestPath = "manifest.yml"
)

// Repository is a writable file system based repository.
// Pak files are stored in the following directory structure:
//
//	<BaseDir>/<id>
//
// The manifest is stored in manifest in the same directory.
type Repository struct {
	BaseDir string
}

// GetInstalledManifest gets the manifest for the given id.
func (r *Repository) GetInstalledManifest(ctx context.Context, id string) (*pak.Manifest, error) {
	manifest, err := r.getManifest(id)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return manifest, nil
}

func (r *Repository) manifestPath(id string) string {
	return filepath.Join(r.BaseDir, id, ManifestPath)
}

func (r *Repository) filePath(id string, name string) string {
	return filepath.Join(r.BaseDir, id, name)
}

func (r *Repository) getManifest(id string) (*pak.Manifest, error) {
	f, err := os.Open(r.manifestPath(id))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return yaml.ReadManifest(f)
}

// ListInstalled returns all specs in the repository.
func (r *Repository) ListInstalled(ctx context.Context) ([]pak.Manifest, error) {
	var ret []pak.Manifest

	if err := filepath.Walk(r.BaseDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if info.Name() != ManifestPath {
			return nil
		}

		// found a manifest file
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		manifest, err := yaml.ReadManifest(f)
		if err != nil {
			// ignore manifests with errors
			return nil
		}

		// manifest must be in the correct directory for it to be returned
		if path != r.manifestPath(manifest.ID) {
			return nil
		}

		ret = append(ret, *manifest)

		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to walk repository: %w", err)
	}

	return ret, nil
}

// Write writes the given file to the repository, in the following location: <BaseDir>/<id>/<file>
func (r *Repository) Write(ctx context.Context, id string, version string, file string, data io.Reader) error {
	path := r.filePath(id, file)

	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", dir, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", path, err)
	}

	defer f.Close()

	if _, err := io.Copy(f, data); err != nil {
		return fmt.Errorf("failed to write file %q: %w", path, err)
	}

	return nil
}

// WriteManifest writes the given manifest to the repository. The manifest file is stored in <BaseDir>/<id>/manifest.
func (r *Repository) WriteManifest(ctx context.Context, manifest pak.Manifest) error {
	path := r.manifestPath(manifest.ID)

	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", dir, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", path, err)
	}

	defer f.Close()

	if err := yaml.WriteManifest(f, manifest); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// Delete deletes the pak with the given id from the repository.
// It will remove the manifest file and all files listed in the manifest.
// If the directory is empty after the files are removed, it will also be removed.
func (r *Repository) Delete(ctx context.Context, id string) error {
	manifest, err := r.GetInstalledManifest(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get manifest: %w", err)
	}

	if manifest == nil {
		// nothing to delete
		return nil
	}

	// only remove the files listed by the manifest
	for _, f := range manifest.Files {
		path := r.filePath(id, f)
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove file %q: %w", path, err)
		}
	}

	// remove the manifest
	if err := os.Remove(r.manifestPath(id)); err != nil {
		return fmt.Errorf("failed to remove manifest: %w", err)
	}

	// remove the directory if it is empty - ignore errors
	_ = os.Remove(filepath.Dir(r.manifestPath(id)))

	return nil
}

// GetManifest gets the manifest for the given id and version.
// This method is used when the Repository is being used as a SourceRepository.
// If version is empty then the latest version is returned.
func (r *Repository) GetManifest(ctx context.Context, id string, version string) (*pak.Manifest, error) {
	f, err := os.Open(r.remoteManifestPath(id, version))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	manifest, err := yaml.ReadManifest(f)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

func (r *Repository) remoteManifestPath(id string, version string) string {
	return filepath.Join(r.BaseDir, id, version, RemoteManifestPath)
}

// GetSpec gets the spec for the given id.
// This method is used when the Repository is being used as a SourceRepository.
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

func (r *Repository) getIndex(ctx context.Context) (pak.SpecIndex, error) {
	path := filepath.Join(r.BaseDir, IndexPath)

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get index file: %w", err)
	}

	defer f.Close()

	index, err := yaml.ReadSpecIndex(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read index file: %w", err)
	}

	return *index, nil
}

// List returns all specs in the repository.
// This method is used when the Repository is being used as a SourceRepository.
// This method will return an error for Repositories used as local storage.
func (r *Repository) List(ctx context.Context) (pak.SpecIndex, error) {
	return r.getIndex(ctx)
}

// GetFile gets the file with the given name for the given id and version.
// This method is used when the Repository is being used as a SourceRepository.
// This method will return an error for Repositories used as local storage.
func (r *Repository) GetFile(ctx context.Context, id string, version string, file string) (io.ReadCloser, error) {
	f, err := os.Open(r.remoteFilePath(id, version, file))
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	return f, nil
}

func (r *Repository) remoteFilePath(id string, version string, file string) string {
	return filepath.Join(r.BaseDir, id, version, file)
}
