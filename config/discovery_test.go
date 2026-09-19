package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types/image"
)

type fakeLister struct {
	images []image.Summary
	err    error
}

func (f fakeLister) ImageList(context.Context, image.ListOptions) ([]image.Summary, error) {
	return f.images, f.err
}

func img(repoTag string, labels map[string]string) image.Summary {
	return image.Summary{RepoTags: []string{repoTag}, Labels: labels}
}

func TestDiscoverBuildsCatalogFromLabels(t *testing.T) {
	d := NewDockerDiscoverer(fakeLister{images: []image.Summary{
		img("websummoner/chrome:153.0", map[string]string{LabelBrowser: "chrome", LabelVersion: "153.0"}),
		img("websummoner/chrome:152.0", map[string]string{LabelBrowser: "chrome", LabelVersion: "152.0"}),
		img("acme/firefox:154.0", map[string]string{
			LabelBrowser: "firefox", LabelVersion: "154.0", LabelPort: "4445", LabelPath: "/wd/hub"}),
	}})

	got, err := d()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want chrome and firefox, got %d browsers", len(got))
	}
	if got["chrome"].Default != "153.0" {
		t.Errorf("default should be the newest version, got %q", got["chrome"].Default)
	}
	if b := got["chrome"].Versions["152.0"]; b == nil || b.Image != "websummoner/chrome:152.0" {
		t.Errorf("older version must stay available: %+v", b)
	}
	// A browser that is not ours must work identically: the labels are the
	// contract, not the repository name.
	ff := got["firefox"].Versions["154.0"]
	if ff.Image != "acme/firefox:154.0" || ff.Port != "4445" || ff.Path != "/wd/hub" {
		t.Errorf("labels not honoured: %+v", ff)
	}
}

func TestDiscoverDefaultsPortAndPath(t *testing.T) {
	d := NewDockerDiscoverer(fakeLister{images: []image.Summary{
		img("websummoner/edge:152.0", map[string]string{LabelBrowser: "edge", LabelVersion: "152.0"}),
	}})
	got, _ := d()
	b := got["edge"].Versions["152.0"]
	if b.Port != "4444" || b.Path != "/" {
		t.Fatalf("want 4444 and /, got %q and %q", b.Port, b.Path)
	}
}

func TestDiscoverSkipsIncompleteImages(t *testing.T) {
	d := NewDockerDiscoverer(fakeLister{images: []image.Summary{
		img("x/no-version:1", map[string]string{LabelBrowser: "chrome"}),
		img("x/no-name:1", map[string]string{LabelVersion: "153.0"}),
		{RepoTags: nil, Labels: map[string]string{LabelBrowser: "chrome", LabelVersion: "153.0"}},
	}})
	got, _ := d()
	if len(got) != 0 {
		t.Fatalf("half-labelled or untagged images must be ignored, got %v", got)
	}
}

func TestNewerComparesNumerically(t *testing.T) {
	// The case string ordering gets wrong, and browsers reach it every year.
	if !newer("10.0", "9.0") {
		t.Error("10.0 must beat 9.0")
	}
	if newer("152.0", "153.0") {
		t.Error("152.0 must not beat 153.0")
	}
	if !newer("153.0.1", "153.0") {
		t.Error("more specific patch must win")
	}
}

func TestFileWinsOverDiscovery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "browsers.json")
	err := os.WriteFile(path, []byte(`{
      "chrome": {"default": "152.0", "versions": {
        "152.0": {"image": "pinned/chrome:152.0", "port": "5555", "path": "/"}}}}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	c := NewConfig()
	c.Discover = NewDockerDiscoverer(fakeLister{images: []image.Summary{
		img("websummoner/chrome:152.0", map[string]string{LabelBrowser: "chrome", LabelVersion: "152.0"}),
		img("websummoner/chrome:153.0", map[string]string{LabelBrowser: "chrome", LabelVersion: "153.0"}),
	}})
	if err := c.Load(path, ""); err != nil {
		t.Fatal(err)
	}

	// The file pins 152.0 to its own image and default; discovery may only add.
	b, version, ok := c.Find("chrome", "152.0")
	if !ok || b.Image != "pinned/chrome:152.0" || b.Port != "5555" {
		t.Fatalf("file entry must win, got %+v %q", b, version)
	}
	if c.Browsers["chrome"].Default != "152.0" {
		t.Errorf("file default must win, got %q", c.Browsers["chrome"].Default)
	}
	if _, _, ok := c.Find("chrome", "153.0"); !ok {
		t.Error("discovery must still contribute versions the file does not mention")
	}
}

func TestLoadSurvivesMissingFileWhenDiscovering(t *testing.T) {
	c := NewConfig()
	c.Discover = NewDockerDiscoverer(fakeLister{images: []image.Summary{
		img("websummoner/chrome:153.0", map[string]string{LabelBrowser: "chrome", LabelVersion: "153.0"}),
	}})
	if err := c.Load(filepath.Join(t.TempDir(), "absent.json"), ""); err != nil {
		t.Fatalf("discovery alone must be a valid configuration: %v", err)
	}
	if _, _, ok := c.Find("chrome", ""); !ok {
		t.Error("discovered browser should be servable")
	}
}

func TestFailedScanKeepsFileCatalog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "browsers.json")
	_ = os.WriteFile(path, []byte(`{"chrome":{"default":"152.0","versions":{
      "152.0":{"image":"pinned/chrome:152.0","port":"4444","path":"/"}}}}`), 0o644)

	c := NewConfig()
	c.Discover = func() (map[string]Versions, error) { return nil, errors.New("docker is down") }
	if err := c.Load(path, ""); err != nil {
		t.Fatalf("a broken scan must not fail the load: %v", err)
	}
	if _, _, ok := c.Find("chrome", "152.0"); !ok {
		t.Error("file catalog must survive a failed scan")
	}
}

func TestLoadFailsWhenNothingConfigured(t *testing.T) {
	c := NewConfig()
	c.Discover = func() (map[string]Versions, error) { return nil, nil }
	if err := c.Load(filepath.Join(t.TempDir(), "absent.json"), ""); err == nil {
		t.Fatal("no file and no images is a misconfiguration, not an empty grid")
	}
}

func TestEmptyConfigFileStaysValid(t *testing.T) {
	// An explicitly empty catalog is a choice, not an error, and predates
	// discovery. Only a *missing* file with nothing discovered is fatal.
	path := filepath.Join(t.TempDir(), "browsers.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewConfig()
	c.Discover = func() (map[string]Versions, error) { return nil, nil }
	if err := c.Load(path, ""); err != nil {
		t.Fatalf("empty file must remain valid: %v", err)
	}
}
