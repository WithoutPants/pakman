package pak

import (
	"context"
	"io"
)

// SourceRepository is a repository that can be used to get paks from.
type SourceRepository interface {
	ManifestGetter
	SpecGetter
	FileGetter
}

// WritableRepository is a repository that can be used to store paks in.
type WritableRepository interface {
	InstalledManifestGetter
	InstalledLister
	FileWriter
	ManifestWriter
	Deleter
}

type ManifestWriter interface {
	WriteManifest(ctx context.Context, manifest Manifest) error
}

type FileWriter interface {
	Write(ctx context.Context, id string, version string, file string, data io.Reader) error
}

type Deleter interface {
	Delete(ctx context.Context, id string) error
}

type SpecGetter interface {
	// GetSpec gets the spec for the given id.
	GetSpec(ctx context.Context, id string) (*Spec, error)

	// List returns all specs in the repository.
	List(ctx context.Context) (SpecIndex, error)
}

type ManifestGetter interface {
	// GetManifest gets the manifest for the given id and version.
	// If version is empty then the latest version is returned.
	GetManifest(ctx context.Context, id string, version string) (*Manifest, error)
}

type InstalledManifestGetter interface {
	// GetInstalledManifest gets the manifest for the given id.
	GetInstalledManifest(ctx context.Context, id string) (*Manifest, error)
}

type InstalledLister interface {
	ListInstalled(ctx context.Context) ([]Manifest, error)
}

type FileGetter interface {
	GetFile(ctx context.Context, id string, version string, file string) (io.ReadCloser, error)
}
