// NOTICE: This is a derivative work of https://github.com/cloudfoundry/cli/blob/master/cf/manifest/manifest_disk_repository.go.
package manifest

import (
	"code.cloudfoundry.org/cli/plugin/models"
	"errors"
	"fmt"
	"github.com/cloudfoundry-incubator/candiedyaml"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

//go:generate counterfeiter . Repository

type Repository interface {
	ReadManifest(string) (*Manifest, error)
}

type DiskRepository struct{}

type FakeRepo struct {
	err  error
	path string
	yaml string
}

func NewEmptyFakeRepo() *FakeRepo {
	return &FakeRepo{}
}

func NewFakeRepo(yaml string) *FakeRepo {
	return &FakeRepo{
		yaml: yaml,
	}
}

func (r *FakeRepo) ReadManifest(path string) (*Manifest, error) {
	r.path = path
	yamlMap := make(map[string]interface{})
	candiedyaml.Unmarshal([]byte(r.yaml), &yamlMap)
	return &Manifest{Data: yamlMap}, r.err
}

func NewDiskRepository() (repo Repository) {
	return DiskRepository{}
}

func (repo DiskRepository) ReadManifest(inputPath string) (*Manifest, error) {

	m := NewEmptyManifest()
	manifestPath, err := repo.manifestPath(inputPath)

	if err != nil {
		return m, errors.New("Error finding manifest")
	}

	m.Path = manifestPath

	mapp, err := repo.readAllYAMLFiles(manifestPath)

	if err != nil {
		return m, err
	}

	m.Data = mapp

	return m, nil
}

func (repo DiskRepository) readAllYAMLFiles(path string) (mergedMap map[string]interface{}, err error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return
	}
	defer file.Close()

	mapp, err := parseManifest(file)
	if err != nil {
		return
	}

	if _, ok := mapp["inherit"]; !ok {
		mergedMap = mapp
		return
	}

	inheritedPath, ok := mapp["inherit"].(string)
	if !ok {
		err = errors.New("invalid inherit path in manifest")
		return
	}

	if !filepath.IsAbs(inheritedPath) {
		inheritedPath = filepath.Join(filepath.Dir(path), inheritedPath)
	}

	inheritedMap, err := repo.readAllYAMLFiles(inheritedPath)
	if err != nil {
		return
	}

	mergedMap, err = DeepMerge(inheritedMap, mapp)
	if err != nil {
		return
	}
	return
}

func parseManifest(file io.Reader) (yamlMap map[string]interface{}, err error) {
	manifest, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}

	mmap := make(map[interface{}]interface{})
	err = yaml.Unmarshal(manifest, &mmap)
	if err != nil {
		return
	}

	if !IsMappable(mmap) || len(mmap) == 0 {
		err = errors.New("Invalid manifest. Expected a map")
		return
	}

	yamlMap = make(map[string]interface{})

	return
}

func (repo DiskRepository) manifestPath(userSpecifiedPath string) (string, error) {
	fileInfo, err := os.Stat(userSpecifiedPath)
	if err != nil {
		return "", err
	}

	// If we've been given a directory, check inside it for manifest.yml/manifest.yaml files.
	if fileInfo.IsDir() {
		manifestPaths := []string{
			filepath.Join(userSpecifiedPath, "manifest.yml"),
			filepath.Join(userSpecifiedPath, "manifest.yaml"),
		}
		var err error
		for _, manifestPath := range manifestPaths {
			if _, err = os.Stat(manifestPath); err == nil {
				return manifestPath, err
			}
		}
		return "", err
	}
	// If we didn't get a directory, assume we've been passed the file we want, so
	// just give that back.
	return userSpecifiedPath, nil
}

type ManifestReader func(Repository, string) *plugin_models.GetAppModel

type ManifestAppFinder struct {
	Repo          Repository
	ManifestPath  string
	AppName       string
	DefaultDomain string
}

func (f *ManifestAppFinder) AppParams() *plugin_models.GetAppModel {
	var manifest *Manifest
	var err error
	if f.ManifestPath == "" {
		manifest, err = f.Repo.ReadManifest("./")
	} else {
		manifest, err = f.Repo.ReadManifest(f.ManifestPath)
	}

	if err != nil {
		fmt.Println(err)
		return nil
	}

	apps, err := manifest.Applications(f.DefaultDomain)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	for index, app := range apps {
		if IsHostEmpty(app) {
			continue
		}

		if app.Name != "" && app.Name != f.AppName {
			continue
		}

		return &apps[index]
	}
	return nil
}
