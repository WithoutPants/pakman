package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/WithoutPants/pakman/pkg/pak"
	"github.com/WithoutPants/pakman/pkg/repository/fs"
	"github.com/WithoutPants/pakman/pkg/repository/http"
	"gopkg.in/yaml.v3"
)

var (
	cfg     *config
	manager *pak.Manager
	ctx     = context.Background()
)

type logger struct{}

func (l logger) Debugf(format string, args ...interface{}) {
	if cfg.Debug {
		l.Infof(format, args...)
	}
}

func (l logger) Infof(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func main() {
	if len(os.Args[1:]) == 0 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	if err := loadConfig(); err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// initialise manager
	initManager()

	switch cmd {
	case "install":
		install()
	case "uninstall":
		uninstall()
	case "upgrade":
		upgrade()
	case "upgradable":
		upgradable()
	case "list":
		list()
	case "installed":
		installed()
	case "search":
		search()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func initManager() {
	var remote pak.SourceRepository
	if strings.HasPrefix(cfg.RemotePath, "http://") || strings.HasPrefix(cfg.RemotePath, "https://") {
		u, err := url.Parse(cfg.RemotePath)
		if err != nil {
			fmt.Printf("Error parsing remote URL: %v\n", err)
			os.Exit(1)
		}

		remote = http.New(*u, nil)
	} else {
		remote = &fs.Repository{
			BaseDir: cfg.RemotePath,
		}
	}

	manager = pak.NewManager(pak.ManagerOptions{
		Local: &fs.Repository{
			BaseDir: cfg.LocalPath,
		},
		Remote: remote,
		Logger: logger{},
	})
}

func usage() {
	fmt.Print(`Usage: pakman <command> [args...]
Pakman is a package manager for the Pak package format.

Pakman will look for a configuration file "pakman.yml" in the current working directory. It will output an error if it cannot find the file.

The format of pakman.yml is as follows:

local: /path/to/local/repository
remote: /path/to/remote/repository
debug: true|false (optional)

local must be a path to a directory where packages will be installed to.
remote must be a path to a directory where packages will be downloaded from, or a URL to a remote repository. If it is a URL, it must be a valid HTTP or HTTPS URL.

debug is optional. If set to true, pakman will output debug messages.

Commands:
  install <package ID>...	Install one or more packages
  uninstall <package ID>...	Uninstall one or more packages
  upgrade <package ID>...	Upgrade one or more packages. If no package ID is specified, all eligible packages will be upgraded.
  upgradable			    List upgradable packages
  list				        List all packages
  installed			        List installed packages
  search <query>			Search for packages
	`)
}

func install() {
	if len(os.Args[1:]) < 2 {
		fmt.Println("Missing package IDs")
		usage()
		os.Exit(1)
	}

	var specs []pak.InstallSpec
	for _, id := range os.Args[2:] {
		specs = append(specs, pak.InstallSpec{
			ID: id,
		})
	}

	err := manager.Install(ctx, specs...)
	if err != nil {
		fmt.Printf("Error installing packages: %v\n", err)
		os.Exit(1)
	}
}

func uninstall() {
	if len(os.Args[1:]) < 2 {
		fmt.Println("Missing package IDs")
		usage()
		os.Exit(1)
	}

	err := manager.Uninstall(ctx, os.Args[2:]...)
	if err != nil {
		fmt.Printf("Error uninstalling packages: %v\n", err)
		os.Exit(1)
	}
}

func upgrade() {
	var specs []pak.InstallSpec
	for _, id := range os.Args[2:] {
		specs = append(specs, pak.InstallSpec{
			ID: id,
		})
	}

	err := manager.Upgrade(ctx, specs...)
	if err != nil {
		fmt.Printf("Error upgrading packages: %v\n", err)
		os.Exit(1)
	}
}

func upgradable() {
	u, err := manager.Upgradable(ctx)

	if err != nil {
		fmt.Printf("Error listing upgradable packages: %v\n", err)
		os.Exit(1)
	}

	for _, v := range u {
		fmt.Printf("%s %s -> %s\n", v.ID, v.CurrentVersion, v.LatestVersion)
	}
}

func list() {
	index, err := manager.List(ctx)
	if err != nil {
		fmt.Printf("Error listing packages: %v\n", err)
		os.Exit(1)
	}

	keys := sortedKeys(index)

	for _, k := range keys {
		v := index[k]
		for _, vv := range v.Versions {
			fmt.Printf("%s %s %s\n", v.ID, vv, v.Description)
		}
	}
}

func sortedKeys(index pak.SpecIndex) []string {
	// sort keys
	var keys []string
	for k := range index {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		if strings.EqualFold(keys[i], keys[j]) {
			return keys[i] < keys[j]
		}

		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})

	return keys
}

func installed() {
	installed, err := manager.ListInstalled(ctx)
	if err != nil {
		fmt.Printf("Error listing installed packages: %v\n", err)
		os.Exit(1)
	}

	for _, v := range installed {
		fmt.Printf("%s %s\n", v.ID, v.Version)
	}
}
func search() {
	if len(os.Args[1:]) < 2 {
		fmt.Println("Missing search term")
		usage()
		os.Exit(1)
	}

	index, err := manager.List(ctx)
	if err != nil {
		fmt.Printf("Error listing packages: %v\n", err)
		os.Exit(1)
	}

	keys := sortedKeys(index)
	for _, k := range keys {
		if strings.Contains(strings.ToLower(k), strings.ToLower(os.Args[2])) {
			v := index[k]
			for _, vv := range v.Versions {
				fmt.Printf("%s %s %s\n", v.ID, vv, v.Description)
			}
		}
	}
}

type config struct {
	LocalPath  string `yaml:"localPath"`
	RemotePath string `yaml:"remotePath"`
	Debug      bool   `yaml:"debug"`
}

func loadConfig() error {
	f, err := os.Open("pakman.yml")
	if err != nil {
		return fmt.Errorf("opening config file: %w", err)
	}

	defer f.Close()

	d := yaml.NewDecoder(f)
	return d.Decode(&cfg)
}
