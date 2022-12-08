// Package yaml provides functions for reading and writing pakman yaml files.
package yaml

import (
	"fmt"
	"io"

	"github.com/WithoutPants/pakman/pkg/pak"
	"gopkg.in/yaml.v3"
)

func readYaml(f io.Reader, v interface{}) error {
	decoder := yaml.NewDecoder(f)

	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("failed to decode yaml: %w", err)
	}

	return nil
}

func writeYaml(out io.Writer, v interface{}) error {
	encoder := yaml.NewEncoder(out)
	defer encoder.Close()

	if err := encoder.Encode(v); err != nil {
		return fmt.Errorf("failed to encode yaml: %w", err)
	}

	return nil
}

// ReadSpec reads a spec from the given reader parsing it as yaml.
func ReadSpec(f io.Reader) (*pak.Spec, error) {
	var spec pak.Spec
	if err := readYaml(f, &spec); err != nil {
		return nil, err
	}

	return &spec, nil
}

// ReadManifest reads a manifest from the given reader parsing it as yaml.
func ReadManifest(f io.Reader) (*pak.Manifest, error) {
	var manifest pak.Manifest
	if err := readYaml(f, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// WriteManifest writes the given manifest to the given writer as yaml.
func WriteManifest(out io.Writer, manifest pak.Manifest) error {
	return writeYaml(out, manifest)
}

// ReadSpecIndex reads a spec index from the given reader parsing it as yaml.
func ReadSpecIndex(f io.Reader) (*pak.SpecIndex, error) {
	var index pak.SpecIndex
	if err := readYaml(f, &index); err != nil {
		return nil, err
	}

	return &index, nil
}
