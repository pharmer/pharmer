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

var _cloudJson = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xa4\x96\xdd\x4f\xeb\x3a\x0c\xc0\xdf\xf7\x57\x58\x7d\x1e\xbb\x6d\x19\xbd\xd3\xde\xc6\xf8\x10\xba\xc0\xae\xee\x26\xee\xf9\x10\x42\xa1\x31\x23\x67\x6d\xd2\x93\x64\x83\x81\xf6\xbf\x1f\x25\x65\x6d\xd7\xaf\x09\xf6\x02\xd4\x76\xec\x9f\x83\x63\xfb\xbd\x03\xe0\x70\x12\xa3\x33\x04\x47\x85\x24\xc2\x17\xb2\x76\xba\x46\x8a\x7c\xe5\x0c\xe1\x67\x07\x00\xc0\xa1\xb8\xb2\x52\x00\xe7\x37\xd9\xfe\x95\x48\x41\x9d\x0e\xc0\xbd\xb5\x97\x38\x67\x82\xab\xec\xcc\xbb\xfd\x09\xe0\x44\x22\x24\x9a\x09\x6e\x42\xfc\x4b\x24\x53\x5d\xb8\x90\x84\x87\xf8\xe1\x27\x3b\x6b\x0c\x12\x22\xbd\x5c\xfe\x26\x38\xe6\x1e\xd3\xa0\xc6\xe0\xe3\xf3\xde\xfe\xde\x74\x9b\xe3\x8d\x62\xa5\x51\x52\x12\x77\xe1\x16\xf5\x33\xca\x88\x70\xaa\xea\x02\x93\x58\xb5\x07\xb6\x06\xbb\x81\xb3\xdc\x19\x57\xda\x64\x34\x5b\x27\x58\x73\x03\x6a\xb1\x34\x21\xee\xc6\xde\x34\x0f\x41\x51\x85\x92\x25\x5b\x52\x1f\x5e\x07\x01\x04\xfd\x47\xa6\x61\x2c\x24\xaa\x2e\xf8\x97\xa7\x10\x63\x2c\xe4\x3a\x3f\x15\x12\x8d\x73\x23\x19\x82\x33\x8e\xc4\x92\xc2\x14\xe5\x0a\x65\x21\xa7\x30\x31\xd1\xfc\x3c\x47\x12\xef\x7c\x53\xa6\x16\xce\x10\x4e\xdc\xda\xdb\xcb\x59\x6f\x1a\x59\xfb\x55\xd6\xfe\x21\xac\xfd\x12\x6b\xbf\xcc\xea\xb9\xfb\x60\xaf\x1b\x61\x83\x2a\xec\xe0\x10\xd8\xa0\x04\x3b\x28\xc3\xfa\xed\xb0\xdf\x82\xfe\x51\xe0\x5e\x9e\x36\x02\x7b\x6e\x95\xd8\x1c\xf8\x3a\xb2\xe7\x96\x98\x03\xb7\x0c\xfd\xf7\x7e\x68\xcf\x6f\xa5\xae\x29\x60\x7b\xe2\x00\xec\x72\x0d\x7b\x7e\x85\xdb\x73\xdb\xc1\xc7\x7e\xf3\x93\xeb\xc3\x19\x52\x66\x70\xe8\xe7\x6b\xe4\x94\x48\xbc\x41\x4d\xa2\x7d\xc5\x5c\xa9\x8f\xf6\x87\x37\xf6\x9b\xdf\xdd\xa0\x0d\xd8\x0b\xbe\x4a\x3c\x28\x5f\x73\xf0\x59\xe4\xe6\xd7\xd7\x8a\x7c\x9c\x77\xb8\x2e\xf8\x27\xa6\x58\xce\x98\xc4\x50\xc3\x74\x7a\x76\x60\x12\xc7\x8d\x0d\x2f\xeb\xda\xa1\x44\x8a\x5c\x33\x12\xd5\xf4\xec\x44\x8a\x15\xa3\x28\x4d\xd0\x69\x71\x30\x6e\x5d\x26\x11\x59\x5f\x08\x19\x13\x6d\x4c\x9e\x18\x46\x34\xd7\x13\xce\x85\xb6\x53\xc8\xb8\x7e\x2f\x8c\xaf\x67\x22\x63\x94\x3d\x92\x24\x2a\x14\x14\x7b\xa1\x88\xff\x0a\xa3\xa5\x19\x53\x47\x39\x90\x71\xb9\x1d\x36\x9b\xcc\xab\x0d\xb2\x3b\x96\x72\xd7\xe9\xc8\x0e\x05\x7f\x62\x73\x0b\x3d\x1e\x5d\x9f\xff\x3f\xfa\xfe\x30\xf9\xef\x72\x74\x7b\xf5\x63\x34\xbb\x9a\xdc\x66\x84\xa9\x3f\x21\xe3\xe2\xe0\x7f\x10\x72\x4e\x38\x7b\x4b\xc7\xe7\x8e\xe9\x2f\x95\xfe\x3f\x9b\x2d\x22\xf2\x88\x96\x7b\xd2\x68\xc2\x78\xb2\xb4\xb7\xa5\xf1\x55\x3b\x99\x66\xd3\xfd\x4c\x3a\xb3\xc9\x3f\xe7\xfb\xf2\xd0\x62\x81\x0d\x09\xd4\xa8\x32\xf2\x59\x55\x97\x21\x27\x44\xa9\x17\x21\x69\x01\xbb\x61\x19\x58\x2c\x1f\x51\x72\xd4\xa8\xee\x50\xaa\xfa\x9d\x68\x95\x6a\x6c\xe3\xec\x0d\x7a\x6e\x73\x5b\xdd\xd5\xa6\x4b\x59\xa1\x9e\xcc\x62\x36\x04\x2d\x97\x98\x63\x9b\x15\xad\x22\xb3\xcb\x5a\x2a\xed\x14\xf9\x2d\x77\x67\xf3\x27\x00\x00\xff\xff\x14\x57\x91\x7c\x09\x0a\x00\x00")

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

	info := bindataFileInfo{name: "cloud.json", size: 2569, mode: os.FileMode(420), modTime: time.Unix(1453795200, 0)}
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
