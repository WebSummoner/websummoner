package config

import (
	"context"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
)

// Labels a browser image declares. Only the first two are required.
const (
	LabelBrowser = "org.websummoner.browser"
	LabelVersion = "org.websummoner.version"
	LabelPort    = "org.websummoner.port"
	LabelPath    = "org.websummoner.path"
)

// Discoverer returns browsers found in the environment. Nil disables discovery.
type Discoverer func() (map[string]Versions, error)

type imageLister interface {
	ImageList(context.Context, image.ListOptions) ([]image.Summary, error)
}

// NewDockerDiscoverer scans images on this host, not a registry: every entry it
// returns is backed by an image already pulled, so it cannot produce a browser
// that fails at session start with "image not found".
func NewDockerDiscoverer(cli imageLister) Discoverer {
	return func() (map[string]Versions, error) {
		images, err := cli.ImageList(context.Background(), image.ListOptions{
			Filters: filters.NewArgs(filters.Arg("label", LabelBrowser)),
		})
		if err != nil {
			return nil, err
		}
		return browsersFromImages(images), nil
	}
}

func browsersFromImages(images []image.Summary) map[string]Versions {
	out := map[string]Versions{}
	for _, img := range images {
		name, version := img.Labels[LabelBrowser], img.Labels[LabelVersion]
		if name == "" || version == "" || len(img.RepoTags) == 0 {
			continue
		}
		versions, ok := out[name]
		if !ok {
			versions = Versions{Versions: map[string]*Browser{}}
		}
		versions.Versions[version] = &Browser{
			Image: img.RepoTags[0],
			Port:  label(img.Labels, LabelPort, "4444"),
			Path:  label(img.Labels, LabelPath, "/"),
		}
		if newer(version, versions.Default) {
			versions.Default = version
		}
		out[name] = versions
	}
	return out
}

func label(labels map[string]string, name, fallback string) string {
	if v := labels[name]; v != "" {
		return v
	}
	return fallback
}

// newer compares dotted versions numerically, so 10.0 beats 9.0.
func newer(a, b string) bool {
	if b == "" {
		return true
	}
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		if x, y := segment(as, i), segment(bs, i); x != y {
			return x > y
		}
	}
	return false
}

func segment(parts []string, i int) int {
	if i >= len(parts) {
		return 0
	}
	n, _ := strconv.Atoi(parts[i])
	return n
}

// merge overlays the file on discovery, so browsers.json always wins.
func merge(discovered, file map[string]Versions) map[string]Versions {
	if len(discovered) == 0 {
		return file
	}
	for name, fromFile := range file {
		found, ok := discovered[name]
		if !ok {
			discovered[name] = fromFile
			continue
		}
		for version, browser := range fromFile.Versions {
			found.Versions[version] = browser
		}
		if fromFile.Default != "" {
			found.Default = fromFile.Default
		}
		discovered[name] = found
	}
	return discovered
}
