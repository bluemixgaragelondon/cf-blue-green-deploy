// NOTICE: This is a derivative work of https://github.com/cloudfoundry/cli/blob/master/cf/manifest/manifest_disk_repository.go.
package manifest

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type ManifestReader interface {
	Read() *Manifest
}

type FileManifestReader struct {
	ManifestPath string
}

func (manifestReader FileManifestReader) Read() *Manifest {
	var manifest *Manifest
	var err error
	if path := manifestReader.ManifestPath; path == "" {
		manifest, err = manifestReader.readManifest("./")
	} else {
		manifest, err = manifestReader.readManifest(path)
	}

	if err != nil {
		fmt.Println(err)
		return nil
	}
	return manifest
}

func (manifestReader *FileManifestReader) readManifest(inputPath string) (*Manifest, error) {

	m := &Manifest{}
	manifestPath, err := manifestReader.interpetManifestPath(inputPath)

	if err != nil {
		return m, errors.New("Error finding manifest")
	}

	m.Path = manifestPath

	mapp, err := manifestReader.readAllYAMLFiles(manifestPath)

	if err != nil {
		return m, err
	}

	m.Data = mapp

	return m, nil
}

func (manifestReader *FileManifestReader) readAllYAMLFiles(path string) (mergedMap map[string]interface{}, err error) {
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

	inheritedMap, err := manifestReader.readAllYAMLFiles(inheritedPath)
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

	yamlMap = make(map[string]interface{})
	err = yaml.Unmarshal(manifest, &yamlMap)

	if err != nil {
		return
	}

	if !IsMappable(yamlMap) || len(yamlMap) == 0 {
		err = errors.New("Invalid manifest. Expected a map")
		return
	}

	return
}

func (manifestReader *FileManifestReader) interpetManifestPath(userSpecifiedPath string) (string, error) {
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
