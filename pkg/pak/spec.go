package pak

import (
	"time"
)

const timeFormat = "2006-01-02 15:04:05 -0700"

type Time struct {
	time.Time
}

func (t *Time) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	parsed, err := time.Parse(timeFormat, s)
	if err != nil {
		return err
	}
	t.Time = parsed
	return nil
}

func (t Time) MarshalYAML() (interface{}, error) {
	return t.Format(timeFormat), nil
}

// Spec is a pak specification.
type Spec struct {
	ID             string   `yaml:"id"`
	Name           string   `yaml:"name"`
	Description    string   `yaml:"description"`
	CurrentVersion string   `yaml:"currentVersion"`
	Updated        Time     `yaml:"updated"`
	Versions       []string `yaml:"versions"`
}

type UpgradableSpec struct {
	Spec
	LatestVersion string `yaml:"latestVersion"`
	LastUpdated   Time   `yaml:"lastUpdated"`
}

// SpecIndex is a map of pak ID to Spec
type SpecIndex map[string]Spec

// Manifest is a pak manifest. It contains the list of files in the pak.
type Manifest struct {
	ID      string   `yaml:"id"`
	Name    string   `yaml:"name"`
	Version string   `yaml:"version"`
	Date    Time     `yaml:"date"`
	Files   []string `yaml:"files"`
}
