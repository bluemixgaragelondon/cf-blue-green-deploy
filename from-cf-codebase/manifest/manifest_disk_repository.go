// NOTICE: This is a derivative work of https://github.com/cloudfoundry/cli/blob/master/cf/manifest/manifest_disk_repository.go.
package manifest

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

<<<<<<< HEAD
=======
	"code.cloudfoundry.org/cli/cf/errors"
	"github.com/bluemixgaragelondon/cf-blue-green-deploy/from-cf-codebase/utils/generic"
>>>>>>> copy generic package from CF code base
	"gopkg.in/yaml.v2"
)

//go:generate counterfeiter . Repository

type Repository interface {
	ReadManifest(string) (*Manifest, error)
}

type DiskRepository struct{}

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

	mergedMap = DeepMerge(inheritedMap, mapp)
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
	return userSpecifiedPath, nil
}
