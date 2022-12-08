# Pakman - an addon manager for Go projects

**This is a prototype and is not ready for production use. It is subject to breaking changes.**

Pakman is a library for managing external addons for Go projects. It is designed to be used by Go projects that need to manage external addons, such as plugins, themes, and other assets.

# Usage

Construct a new manager with the desired options, then use the manager object to install, update, list and remove addons.

```
import (
    "net/url"

    "github.com/WithoutPants/pakman/pkg/repository/fs"
	"github.com/WithoutPants/pakman/pkg/repository/http"
)

// Local repository - stores addons on the local filesystem
local := &fs.Repository{
    BaseDir: localPath,
}

// Remote repository - reads addons using a http URL
u, _ := url.Parse("https://example.com/addons")
remote := http.New(*u, nil)

manager := pak.NewManager(pak.ManagerOptions{
    Local: local,
    Remote: remote,
    Logger: logger{},
})

err := manager.Install("widget", "1.0.0")
```

The `Local` repository is a `pak.WritableRepository` and is used to store addons. An example implementation is provided in the `fs` package. 

The `Remote` repository is a `pak.SourceRepository` and is used to retrieve addons. An example implementation is provided in the `http` package.

# Example CLI client

An example CLI client is provided in the `cmd/pakman` directory. It is a simple command line client that can be used to install, update, list and remove addons.

It can be built with `go build ./cmd/pakman`.

For testing purposes, an example http repository is provided at: `https://withoutpants.github.io/CommunityScrapers/`

For further details, run `pakman` to see usage information and the configuration format.
