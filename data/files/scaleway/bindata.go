// Code generated by go-bindata.
// sources:
// cloud.json
// DO NOT EDIT!

package scaleway

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _cloudJson = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xa4\x96\xdf\x4f\xe3\x3e\x0c\xc0\xdf\xf7\x57\x58\x7d\x2e\xfb\xb6\xa5\xf4\x3b\xed\x6d\x8c\x1f\x42\x07\xec\x74\x9b\xb8\x5f\x42\x28\xb4\x66\xe4\xd6\x26\xbd\x24\x1b\x0c\xb4\xff\xfd\x94\x0c\xda\xae\x6b\x3a\xc1\x5e\x36\xc5\x76\xec\x8f\x53\xc7\xce\x6b\x07\xc0\x61\x24\x43\xa7\x0f\x8e\x8c\x49\x8a\x4f\x64\xe9\xb8\x5a\x8a\x6c\xe1\xf4\xe1\x77\x07\x00\xc0\x49\x70\x61\xa4\x00\xce\x5f\xe2\x74\x00\x6e\x8d\x8d\xc0\x29\xe5\x4c\x16\x76\xaf\xe6\x17\xc0\x49\x79\x4c\x14\xe5\x4c\xbb\xfd\x4a\x04\x95\x2e\x9c\x09\xc2\x62\x7c\xf3\x52\xec\xd5\x06\x39\x11\x7e\x29\x7f\xe1\x0c\x4b\x8f\x46\x64\x0c\xde\x96\xb7\xe6\x7f\xe5\xda\xe3\x0d\x32\xa9\x50\x24\x24\x73\xe1\x1a\xd5\x23\x8a\x94\xb0\x44\x36\x05\x26\x99\x6c\x0f\x6c\x0c\x36\x03\x17\xb9\x53\x26\x95\xce\x68\xb2\xcc\xb1\xe1\x04\xe4\x6c\xae\x43\xdc\x0c\xfd\x71\x19\x22\x41\x19\x0b\x9a\xbf\x93\x06\xf0\xdc\x8b\x20\x0a\xef\xa9\x82\x21\x17\x28\x5d\x08\xce\x8f\x21\xc3\x8c\x8b\x65\xb9\x2b\x26\x0a\xa7\x5a\xd2\x07\x67\x98\xf2\x79\x02\x63\x14\x0b\x14\x95\x9c\xe2\x5c\x47\x0b\xca\x1c\x49\xb6\xb1\x4e\xa8\x9c\x39\x7d\x38\xf2\x1a\x4f\xaf\x64\xbd\xb2\xb2\x86\xdb\xac\xe1\x3e\xac\x61\x8d\x35\xac\xb3\xfa\xde\x2e\xd8\x4b\x2b\x6c\xb4\x0d\xdb\xdb\x07\x36\xaa\xc1\xf6\xea\xb0\x41\x3b\xec\x8f\x28\x3c\x88\xbc\xf3\x63\x2b\xb0\xef\x6d\x13\xeb\x0d\x9f\x47\xf6\xbd\x1a\x73\xe4\xd5\xa1\xff\xdf\x0d\xed\x07\xad\xd4\x0d\x05\x6c\x76\xec\x81\x5d\xaf\x61\x3f\xd8\xe2\xf6\xbd\x76\xf0\x61\x60\xbf\x72\x21\x9c\x60\x42\x35\x4e\xf2\xf1\x1a\x39\x26\x02\xaf\x50\x91\x74\x57\x31\x6f\xd5\x47\xfb\xc5\x1b\x06\xf6\x7b\xd7\x6b\x03\xf6\xa3\xcf\x12\xf7\xea\xc7\x1c\x7d\x14\xd9\x7e\xfb\x5a\x91\x0f\xcb\x0e\xe7\x42\x70\xa4\x8b\xe5\x84\x0a\x8c\x15\x8c\xc7\x27\x7b\x26\x71\x68\x6d\x78\x45\xd7\x8e\x05\x26\xc8\x14\x25\x69\x43\xcf\xce\x05\x5f\xd0\x04\x85\x0e\x3a\xae\x0e\xc3\x77\x97\x79\x4a\x96\x67\x5c\x64\x44\x69\x93\x07\x8a\x69\x52\xea\x09\x63\x5c\x99\x29\xa4\x5d\xbf\x56\xc6\xd7\x23\x11\x19\x8a\x2e\xc9\x73\x19\xf3\x04\xbb\x31\xcf\xfe\x8b\xd3\xb9\x1e\x53\x07\x25\x90\x76\xf9\x3e\x6c\x56\x85\x57\x13\x64\x73\x2c\x95\xae\xd7\x63\x3a\xe6\xec\x81\x4e\x0d\xf4\x70\x70\x79\xfa\x7d\xf0\xf3\x6e\xf4\xed\x7c\x70\x7d\xf1\x6b\x30\xb9\x18\x5d\x17\x84\x6b\x7f\x5c\x64\xd5\x61\x7f\xc7\xc5\x94\x30\xfa\xb2\x1e\x9f\x1b\xa6\x7f\xe4\xfa\x7b\xda\x2d\x52\x72\x8f\x86\x7b\x64\x35\xa1\x2c\x9f\x9b\xd3\x52\xf8\xac\x9c\x42\xb3\x72\x3f\x92\xce\x64\xf4\xe5\x74\x57\x1e\x8a\xcf\xd0\x92\x40\x83\xaa\x20\x9f\x6c\xeb\x0a\xe4\x9c\x48\xf9\xc4\x45\x52\xc1\xb6\x3c\x06\x66\xf3\x7b\x14\x0c\x15\xca\x1b\x14\xb2\xf9\x4d\xb4\x58\x6b\x4c\xe3\xec\xf6\xba\x9e\xbd\xad\x6e\x6a\xd7\x0f\xb1\x4a\x3d\xe9\xc7\x58\x1f\x94\x98\x63\xa7\x4a\x65\x68\x3a\xab\x7f\x01\x00\x00\xff\xff\x8c\xe7\x37\x22\xd3\x09\x00\x00")

func cloudJsonBytes() ([]byte, error) {
	return bindataRead(
		_cloudJson,
		"cloud.json",
	)
}

func cloudJson() (*asset, error) {
	bytes, err := cloudJsonBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "cloud.json", size: 2515, mode: os.FileMode(420), modTime: time.Unix(1453795200, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"cloud.json": cloudJson,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"cloud.json": {cloudJson, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
