package pak

import (
	"context"
	"fmt"
)

var (
	ErrInvalidInstallSpec = fmt.Errorf("invalid install spec")
	ErrSpecNotFound       = fmt.Errorf("not found")
)

type ManifestNotFoundError struct {
	Version string
}

func (e ManifestNotFoundError) Error() string {
	return fmt.Sprintf("manifest not found for version %s", e.Version)
}

type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
}

// Manager manages the installation of paks.
type Manager struct {
	local  WritableRepository
	remote SourceRepository

	logger Logger
	// TODO: progress
}

type ManagerOptions struct {
	Local  WritableRepository
	Remote SourceRepository

	Logger Logger
}

type noopLogger struct{}

func (l noopLogger) Debugf(format string, args ...interface{}) {
}

func (l noopLogger) Infof(format string, args ...interface{}) {
}

// NewManager creates a new Manager. It panics if the local or remote repositories are nil.
func NewManager(options ManagerOptions) *Manager {
	if options.Local == nil {
		panic("local repository is required")
	}
	if options.Remote == nil {
		panic("remote repository is required")
	}
	if options.Logger == nil {
		options.Logger = noopLogger{}
	}

	return &Manager{
		local:  options.Local,
		remote: options.Remote,
		logger: options.Logger,
	}
}

type InstallSpec struct {
	ID string
	// Version is the version to install. If empty, the latest version is installed.
	Version string
}

// Install installs the given paks.
// If the pak is already installed, it will be upgraded to the applicable version,
// unless the already installed version is the same as the requested version, in which
// case the function will return with no changes.
func (m *Manager) Install(ctx context.Context, specs ...InstallSpec) error {
	const upgrade = false
	for _, spec := range specs {
		m.logger.Infof("Installing %s@%s", spec.ID, spec.Version)
		if err := m.install(ctx, spec, upgrade); err != nil {
			return fmt.Errorf("installing pak %s@%s: %w", spec.ID, spec.Version, err)
		}
	}

	return nil
}

func (m *Manager) install(ctx context.Context, toInstall InstallSpec, upgrade bool) error {
	if toInstall.ID == "" {
		return ErrInvalidInstallSpec
	}

	// check if pak already installed
	existing, err := m.local.GetInstalledManifest(ctx, toInstall.ID)
	if err != nil {
		return fmt.Errorf("getting local pak spec: %w", err)
	}

	if toInstall.Version == "" {
		spec, err := m.remote.GetSpec(ctx, toInstall.ID)
		if err != nil {
			return fmt.Errorf("getting spec: %w", err)
		}

		if spec == nil {
			return ErrSpecNotFound
		}

		toInstall.Version = spec.CurrentVersion
	}

	if existing != nil {
		// check if version is already installed
		if existing.Version == toInstall.Version {
			m.logger.Debugf("pak %s@%s already installed", toInstall.ID, toInstall.Version)
			return nil
		}
	}

	if upgrade {
		m.logger.Infof("Upgrading %s from %s to %s", toInstall.ID, existing.Version, toInstall.Version)
	}

	// get pak manifest for latest version/selected version
	manifest, err := m.remote.GetManifest(ctx, toInstall.ID, toInstall.Version)
	if err != nil {
		return fmt.Errorf("getting remote pak manifest: %w", err)
	}

	if manifest == nil {
		return ManifestNotFoundError{Version: toInstall.Version}
	}

	if existing != nil {
		// uninstall the existing version
		if err := m.uninstall(ctx, toInstall.ID); err != nil {
			return fmt.Errorf("uninstalling existing version: %w", err)
		}
	}

	// download pak files sending to store
	for _, file := range manifest.Files {
		if err := m.downloadFile(ctx, toInstall.ID, toInstall.Version, file); err != nil {
			return fmt.Errorf("downloading file %q: %w", file, err)
		}
	}

	if err := m.local.WriteManifest(ctx, *manifest); err != nil {
		return fmt.Errorf("writing local pak manifest: %w", err)
	}

	return nil
}

func (m *Manager) downloadFile(ctx context.Context, id string, version string, file string) error {
	rc, err := m.remote.GetFile(ctx, id, version, file)
	if err != nil {
		return fmt.Errorf("getting remote pak file: %w", err)
	}

	defer rc.Close()

	if err := m.local.Write(ctx, id, version, file, rc); err != nil {
		return fmt.Errorf("writing local pak file: %w", err)
	}

	return nil
}

// Uninstall uninstalls the given paks.
func (m *Manager) Uninstall(ctx context.Context, ids ...string) error {
	for _, id := range ids {
		m.logger.Infof("Uninstalling %s", id)
		if err := m.uninstall(ctx, id); err != nil {
			return fmt.Errorf("uninstalling pak %s: %w", id, err)
		}
	}

	return nil
}

func (m *Manager) uninstall(ctx context.Context, id string) error {
	if err := m.local.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting local pak: %w", err)
	}

	return nil
}

// Upgrade upgrades the given paks to the version specified in the spec.
// If no specs are given then all paks are upgraded to the latest version.
func (m *Manager) Upgrade(ctx context.Context, specs ...InstallSpec) error {
	if len(specs) == 0 {
		// get all installed paks
		installed, err := m.local.ListInstalled(ctx)
		if err != nil {
			return fmt.Errorf("listing local paks: %w", err)
		}

		for _, pak := range installed {
			specs = append(specs, InstallSpec{
				ID: pak.ID,
			})
		}
	}

	const upgrade = true

	for _, spec := range specs {
		if err := m.install(ctx, spec, upgrade); err != nil {
			return fmt.Errorf("upgrading pak %s: %w", spec.ID, err)
		}
	}

	return nil
}

// Upgradable returns a list of paks that can be upgraded.
func (m *Manager) Upgradable(ctx context.Context) ([]UpgradableSpec, error) {
	// get all installed paks
	installed, err := m.local.ListInstalled(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing local paks: %w", err)
	}

	var upgradable []UpgradableSpec
	for _, pak := range installed {
		spec, err := m.remote.GetSpec(ctx, pak.ID)
		if err != nil {
			return nil, fmt.Errorf("getting latest version: %w", err)
		}

		if spec.CurrentVersion != pak.Version {
			upgradable = append(upgradable, UpgradableSpec{
				Spec: Spec{
					ID:          pak.ID,
					Description: spec.Description,
					// get the version and date from the installed pak
					CurrentVersion: pak.Version,
					Updated:        pak.Date,
				},
				LatestVersion: spec.CurrentVersion,
				LastUpdated:   spec.Updated,
			})
		}
	}

	return upgradable, nil
}

// List lists all paks in the remote repository.
func (m *Manager) List(ctx context.Context) (SpecIndex, error) {
	specs, err := m.remote.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing remote paks: %w", err)
	}

	return specs, nil
}

// ListInstalled lists all installed paks in the local repository.
func (m *Manager) ListInstalled(ctx context.Context) ([]Manifest, error) {
	installed, err := m.local.ListInstalled(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing local paks: %w", err)
	}

	return installed, nil
}
