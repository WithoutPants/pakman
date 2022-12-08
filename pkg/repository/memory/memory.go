// Package memory provides a memory based repository implementation.
// Suitable for testing.
package memory

import (
	"bytes"
	"context"
	"io"

	"github.com/WithoutPants/pakman/pkg/pak"
)

type FileSpec struct {
	pak.InstallSpec
	File string
}

type Repository struct {
	Index     pak.SpecIndex
	Manifests map[pak.InstallSpec]pak.Manifest
	Files     map[FileSpec][]byte
}

func New() *Repository {
	return &Repository{
		Index:     make(pak.SpecIndex),
		Manifests: make(map[pak.InstallSpec]pak.Manifest),
		Files:     make(map[FileSpec][]byte),
	}
}

// GetSpec gets the spec for the given id and version.
// If version is empty then the latest version is returned.
func (r *Repository) GetSpec(ctx context.Context, id string) (*pak.Spec, error) {
	s, ok := r.Index[id]
	if !ok {
		return nil, nil
	}

	return &s, nil
}

// List returns all specs in the repository.
func (r *Repository) List(ctx context.Context) (pak.SpecIndex, error) {
	ret := make(pak.SpecIndex, len(r.Index))
	for k, v := range r.Index {
		ret[k] = v
	}
	return ret, nil
}

// GetManifest gets the manifest for the given id and version.
func (r *Repository) GetManifest(ctx context.Context, id string, version string) (*pak.Manifest, error) {
	s, ok := r.Manifests[pak.InstallSpec{ID: id, Version: version}]
	if !ok {
		return nil, nil
	}

	return &s, nil
}

func (r *Repository) GetFile(ctx context.Context, id string, version string, file string) (io.ReadCloser, error) {
	s, ok := r.Files[FileSpec{InstallSpec: pak.InstallSpec{ID: id, Version: version}, File: file}]
	if !ok {
		return nil, nil
	}

	return io.NopCloser(bytes.NewReader(s)), nil
}

func (r *Repository) Write(ctx context.Context, id string, version string, file string, data io.Reader) error {
	buf := bytes.Buffer{}
	_, err := io.Copy(&buf, data)
	if err != nil {
		return err
	}

	r.Files[FileSpec{InstallSpec: pak.InstallSpec{ID: id, Version: version}, File: file}] = buf.Bytes()
	return nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	delete(r.Index, id)
	for k := range r.Manifests {
		if k.ID == id {
			delete(r.Manifests, k)
		}
	}
	for k := range r.Files {
		if k.ID == id {
			delete(r.Files, k)
		}
	}
	return nil
}
