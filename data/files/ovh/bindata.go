// Code generated by go-bindata.
// sources:
// cloud.json
// DO NOT EDIT!

package ovh

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

var _cloudJson = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xbc\x98\xdf\x6f\xe2\x38\x10\xc7\xdf\xfb\x57\x58\x79\xa6\x1c\x09\x94\x45\x7d\xa3\x2d\x0b\x27\xdd\xb6\x51\xdd\xde\x3e\x9c\x56\x95\x9b\xcc\x96\x5c\x83\x1d\xd9\x0e\x77\xdd\x8a\xff\xfd\x64\x03\x09\x38\xbf\x0c\x89\xee\xa5\x29\x1e\x67\xe6\x93\x99\xf9\xca\xf2\x7c\x5e\x20\xe4\x50\xb2\x02\xe7\x1a\x39\x6c\xbd\x74\x7a\x6a\x01\xe8\xda\xb9\x46\x7f\x5d\x20\x84\x90\x13\xc2\xda\xb9\x40\xe8\x87\xb6\x70\x78\x8b\x18\x15\x99\xf5\x53\xff\x45\xc8\x89\x59\x40\x64\xc4\xa8\xf2\x73\x03\x24\x5d\x12\x4e\x59\x24\x7a\xe8\x96\x50\x12\x12\xed\x57\x6f\xdc\x7a\x50\xdb\xee\x19\x97\x4b\x34\x5d\x01\x8f\x82\x83\x0d\xbf\x18\x85\x3c\x80\x5e\xba\x59\x60\xd7\xd9\xfd\xfc\xa1\x9f\x9b\x5e\x75\xf8\x39\x27\x6b\x88\x23\x0a\xa2\x87\xbe\x72\x42\x03\x28\x8b\xfe\x1d\x84\x04\x4e\xd1\x2c\xe5\x2c\x81\xda\xf0\xf3\xc7\xe9\x09\xe1\xb1\xe4\x44\xbc\xb2\x94\xbf\xd5\x85\xbf\x05\x2a\x39\x89\x6d\xc2\xe3\x9b\xb9\x19\x3e\xab\x47\x44\x85\x54\x21\x9e\x3e\x12\x28\xa9\x8a\x78\x4f\x55\xb0\x3f\x7d\x8c\x6e\xff\x78\x78\xbe\x43\x6e\x1e\x29\x04\x11\xf0\x28\xd9\x63\xbb\x68\x7d\xcb\xb8\xca\xd9\xb0\xef\xce\x97\xbf\xf2\x8d\x01\x91\xf0\xc6\xf8\x47\xe6\x29\x66\x69\x78\x60\x4e\x54\x0c\x37\xff\x46\xb2\x72\xae\x91\x97\xc7\x89\xc4\xbb\x5a\xb8\x2a\xcd\x5c\x81\xd0\xab\x24\xf4\xda\x11\x7a\x06\xe1\xc8\x24\xbc\x1a\xd8\x11\x0e\x2b\x09\x47\xed\x08\x47\x06\xe1\xc4\x24\x74\x07\x8d\x88\x18\xdb\x15\xd9\xeb\x8f\x6a\x01\x31\xbe\x3b\xb9\xc4\xae\x15\x5d\x75\x81\x3b\xa5\x2b\x94\xd7\xb3\xa2\xab\x2e\xae\xd7\x86\xce\x6c\xbe\x42\x69\x47\x96\xcd\xf7\x38\xfd\xd6\x49\x7d\x75\x03\x2a\x6f\x4d\x79\x1c\x9f\x2b\x64\x45\x6a\x23\xe6\xb3\x49\xcd\x9c\xba\x85\x86\xb4\x55\xb4\x42\xb5\x51\xf5\xd9\xa8\xa6\xb2\xbd\x42\x77\x36\x48\x7b\x36\xbf\xfc\x62\x95\xcb\x61\x25\xa0\x9f\xbe\xc6\x51\xb0\x63\xc4\xc0\xd7\xc0\x45\x53\x4a\xbf\x14\x45\xd4\x84\xe9\x5e\x59\x25\xb2\x1d\xa7\x99\x4f\xf7\xaa\xa8\xa7\x26\xd0\xe1\xa0\x12\x74\xd2\x19\xe8\xc4\x00\x1d\x0e\x4c\xd0\x49\x23\xe8\xb8\x1a\xd4\x1d\x77\x46\xea\x8e\x4d\xe1\x17\x50\xdd\x71\x73\xf5\xbd\x6a\xd8\x61\x77\x7d\x3a\x2c\x6a\xff\x64\xda\x85\x9d\xa4\xea\x4e\xf2\xff\x43\x52\x0b\x4b\x49\xb5\xe5\x6c\x2d\xa9\x85\xa5\xa4\xda\x82\xb6\x96\xd4\xc2\x56\x52\x6d\x49\x3b\x90\xd4\xc2\x5a\x52\x6d\x61\xbb\x90\x14\xf6\xeb\x5a\xc0\xee\xc8\x3f\x47\x53\xc5\x16\x68\x10\x15\xf6\xeb\x5a\xc0\xee\xc0\x3f\x47\x54\xc5\x06\x68\x10\x15\xf6\x6b\xeb\x3f\xe9\x8c\xd4\x54\x55\x49\xf5\x1b\x64\x85\xfd\x4b\x6f\x64\x79\x54\x75\xcb\xea\x8d\x6a\x3a\x35\xbb\x2a\x07\x1c\x42\xa0\x32\x22\x71\xc9\x45\x39\xe1\x6c\x1d\x85\xc0\x15\xc2\xc3\x6e\x0c\xb2\xf7\x97\xc4\xe4\xe3\x2b\xe3\x2b\x22\x95\xf5\x67\x04\xf1\xc1\xcd\x8d\x50\xca\xa4\xbe\xfb\x2b\xaf\x7b\x7f\xca\xe3\x92\xf0\x15\xf0\x3e\x49\x12\x11\xb0\x10\xfa\x01\x5b\xfd\x16\xc4\xa9\x90\xc0\x2f\x73\x16\xe5\x72\x7f\xb9\xdf\x64\x5e\x75\x90\xe3\x31\x40\xee\x7a\x3b\xa0\x09\x18\xfd\x19\xbd\x69\x5e\xfc\xf2\x8c\x67\x8f\xf7\xd3\x6f\xb3\x8c\x6b\xeb\x85\xf1\x95\x9e\xeb\x88\x97\x54\x00\xd7\x63\x9e\xa3\x0d\x7f\x8b\x6d\x6d\xca\xad\x31\x79\x05\xcd\xf7\x5c\x6a\x8e\x68\x92\xea\x8c\x48\xf8\x57\x3a\x99\x65\xd3\xb3\x43\xf6\xa7\x18\x7f\x7f\x78\xbc\xab\x44\x4e\x88\x10\xff\x30\x1e\x96\x23\x97\x5b\x33\x64\xbf\xd4\x9c\x21\x67\x6f\x9f\x8c\xfd\x34\xbb\x9f\xde\x3f\xbd\xfc\x5e\xcd\x2d\x81\x12\x2a\x5f\xa2\x0a\xf0\xad\xd9\x7c\x3f\x03\x7f\xd2\x66\x64\xda\xab\x92\xbd\xfb\xcf\x1c\x0b\xbd\xa7\xaf\xc0\x29\x48\xc8\x9b\xd2\x51\x4a\x3a\x9a\xdd\x1d\x7e\xe9\xde\xaa\x95\xda\x9f\xf4\x07\x07\xe1\x0b\x4a\x36\xed\xdb\x71\xe1\x51\xda\x42\x50\x4b\x92\xa7\x50\x60\xdd\x3e\x15\xf1\xe6\x62\xf3\x5f\x00\x00\x00\xff\xff\x66\x5c\xc9\x61\x80\x14\x00\x00")

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

	info := bindataFileInfo{name: "cloud.json", size: 5248, mode: os.FileMode(420), modTime: time.Unix(1453795200, 0)}
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
