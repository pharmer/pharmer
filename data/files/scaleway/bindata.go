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

var _cloudJson = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xa4\x96\xdf\x4e\xe3\x38\x14\xc6\xef\x79\x0a\x2b\xd7\xa1\x93\x84\x4c\x06\x71\x17\xd2\x01\x55\x53\xda\x2e\x65\x99\xfd\xa3\x11\x72\x9d\x43\xeb\xad\x63\x67\x6d\xa7\x4c\x41\xbc\xfb\xc8\x4e\x9b\xa4\x6d\x52\xc4\x70\x03\x6a\xfc\xf9\x9c\xdf\xb1\x3f\xfb\xf8\xe5\x04\x21\x87\xe3\x0c\x9c\x0b\xe4\x28\x82\x19\x3c\xe1\xb5\xe3\x9a\xaf\xc0\x57\xce\x05\xfa\xf7\x04\x21\x84\x9c\x14\x56\xf6\x2b\x42\xce\xff\xd8\x39\x41\xe8\x87\xd5\x48\x98\x53\xc1\x55\xa5\x7b\xb1\x7f\x11\x72\x98\x20\x58\x53\xc1\x4d\xd8\x09\x96\x54\xb9\xe8\x4a\x62\x4e\x60\x13\xa5\x9a\x6b\x04\x39\x96\x7e\xfd\xfd\x59\x70\xa8\x23\xda\x4f\x56\xb0\xf9\xf9\xc3\xfe\x7f\x75\xbb\xf3\xc5\x99\xd2\x20\x53\x9c\xb9\x68\x04\x7a\x01\x92\x61\x9e\xaa\xb6\xc4\x38\x53\xc7\x13\x5b\xc1\x6e\xe2\xaa\x76\xca\x95\x36\x15\xdd\xad\x73\x68\x59\x01\xb5\x2c\x4c\x8a\xfb\xc4\x9f\xd6\x29\x52\x50\x44\xd2\x7c\x4b\x1a\xa0\x9f\xe7\x11\x8a\xc2\x19\xd5\x28\x11\x12\x94\x8b\x82\xeb\x4b\x94\x41\x26\xe4\xba\x9e\x45\xb0\x86\xb9\xf9\x72\x81\x9c\x84\x89\x22\x45\x53\x90\x2b\x90\x8d\x9a\x48\x6e\xb2\x05\x75\x8d\x38\xdb\xf9\x9d\x52\xb5\x74\x2e\xd0\x67\xaf\x75\xf5\x6a\xd6\x9b\x4e\xd6\xf0\x90\x35\xfc\x08\x6b\xb8\xc7\x1a\xee\xb3\xfa\xde\x5b\xb0\xc3\x4e\xd8\xe8\x10\xf6\xfc\x23\xb0\xd1\x1e\xec\xf9\x3e\x6c\x70\x1c\xf6\xaf\x28\x3c\x8d\xbc\xeb\xcb\x4e\x60\xdf\x3b\x24\x36\x13\x7e\x1f\xd9\xf7\xf6\x98\x23\x6f\x1f\xfa\xcb\xdb\xd0\x7e\x70\x94\xba\xc5\xc0\x76\xc6\x07\xb0\xf7\x3d\xec\x07\x07\xdc\xbe\x77\x1c\x3c\x09\xba\x8f\x5c\x88\xfa\x90\x52\x83\x93\xbe\xdf\x23\x97\x58\xc2\x0d\x68\xcc\xde\x32\xf3\x81\x3f\x8e\x1f\xbc\x24\xe8\x3e\x77\xe7\xc7\x80\xfd\xe8\x77\x89\xcf\xf7\x97\x39\x7a\x2f\x72\xf7\xe9\x3b\x8a\x7c\x56\xdf\x70\x2e\x0a\x3e\x1b\xb3\xf4\xa9\x04\xa2\xd1\x74\xda\xff\x60\x11\x67\x9d\x17\x5e\x75\x6b\x13\x09\x29\x70\x4d\x31\x6b\xb9\xb3\x73\x29\x56\x34\x05\x69\x92\x4e\x9b\xcd\x70\x1b\x32\x67\x78\x7d\x25\x64\x86\xb5\x91\x3c\x52\x60\x69\x3d\x8e\x39\x17\xda\x76\x21\x13\xfa\xa5\xd1\xbe\x16\x58\x66\x20\x7b\x38\xcf\x15\x11\x29\xf4\x88\xc8\x3e\x11\x56\x98\x36\x75\x5a\x03\x99\x90\xdb\x66\xf3\x5a\x45\xb5\x49\x76\xdb\x52\x1d\xba\x6c\xd3\x44\xf0\x47\x3a\xb7\xd0\x49\x3c\xfc\xfa\x3d\xfe\xfb\x61\x7c\x7b\x1d\x8f\x06\xff\xc4\x77\x83\xf1\xa8\x22\x2c\xe3\x09\x99\x35\x9b\xfd\x83\x90\x73\xcc\xe9\x73\xd9\x3e\x77\xa4\xff\xa9\x72\x3f\xbb\x15\x0c\xcf\xc0\x72\x8f\x3b\x25\x94\xe7\x85\x5d\x2d\x0d\x3f\xb5\x53\x8d\xbc\xba\xef\x29\xe7\x6e\xfc\xed\xeb\x5b\x75\x68\xb1\x84\x8e\x02\x5a\x86\x2a\xf2\xbb\xc3\xb1\x0a\x39\xc7\x4a\x3d\x09\x99\x36\xb0\x3b\x1e\x03\xcb\x62\x06\x92\x83\x06\x75\x0f\x52\xb5\xbf\x89\x56\xe5\x88\xbd\x38\x7b\x5f\x7a\x61\xf7\xb5\xba\x3b\x5a\x3e\xc4\x1a\x7e\x32\x8f\xb1\x0b\xa4\x65\x01\x07\x6e\x49\xe1\x11\x17\x4c\x4f\x73\x20\xbb\x73\x36\x76\x1b\xe4\xb7\x98\xcf\xa1\x6c\x39\xbd\x20\x0c\x7b\x5e\xcf\xfb\xe4\x47\x8d\x05\x70\x14\xc8\x15\x25\x90\xb4\xcd\xf0\x5a\xf4\x98\xd9\xd7\x17\x8c\x44\x0a\x09\x4d\xa5\xda\xc0\x35\x24\xc0\xf1\x8c\x6d\x23\xde\x08\x4e\xb5\x90\x94\xdb\x3d\xde\x1e\x0a\xa7\x4b\x3e\x14\xf3\x79\xa9\x6d\x0d\x6a\xb2\x76\x4a\x58\x39\xd0\x07\xa5\x29\xaf\x1e\x88\xdb\x94\xa7\xc0\xb0\xd2\x94\x28\xc0\x92\x2c\x76\x00\x9a\x03\x9b\xe8\xb7\x90\x33\x4a\xb0\xa9\xce\x6f\x48\x53\xae\xca\x56\x36\xc8\x1b\x6b\xe4\x7b\xce\xae\xa6\x2f\x32\x4c\x6d\xf6\x65\xcf\x2c\x17\xdb\x59\xc1\x34\xa3\xca\x78\x23\x11\x5c\x4b\x61\x7d\x39\xc2\x19\xa8\x1c\x13\x18\xd2\x47\x20\x6b\xc2\xc0\x1d\xd2\x8c\x6a\xbb\x1b\xd2\x9d\x96\x9b\x14\x13\x22\x0a\xae\xdd\x89\xf1\x96\xd2\xc0\xf5\xbd\x60\x45\x06\x43\x63\x6f\xb7\xbf\x71\x83\x16\x12\xcf\x21\x61\x58\x29\xf7\x16\x94\x28\x24\x81\x3f\x0a\xa1\x71\x13\x22\xc3\x6d\x06\x89\xec\x86\x07\x61\x53\xc9\x41\x3f\x09\xb9\x9c\x34\x6e\x4b\xe3\xff\xd3\x47\x86\x39\x07\xd6\xb9\x93\x31\x03\xa9\xbb\xf6\x5c\x98\x85\x75\x52\x98\x51\xcc\xdb\x92\x09\x46\xc9\xba\x99\x92\x0b\xde\x62\x9a\xef\x30\x5b\x08\xb1\xb4\xa7\x3a\x2e\xf4\x82\x77\xf9\xe6\x76\x86\x89\x11\x3c\x1f\x0a\xa6\xe3\xab\xbb\xe1\x38\xf9\xf6\xe7\xe4\x61\x12\x8f\x06\x49\x57\x88\x78\x32\x50\x76\xeb\x2f\xb1\xa2\x24\x2e\x52\xaa\x3b\xa5\x9b\x8a\x63\xad\x29\x39\x14\xe5\x82\xb1\x29\x03\xc8\x07\x5c\x83\x5c\xd9\x5e\x70\xd6\x6a\xf5\x49\x31\x63\x94\x0c\x26\x7b\x17\x40\x7d\x1d\x9d\xbc\xfe\x0a\x00\x00\xff\xff\x7b\x13\xf9\x4e\xd4\x0d\x00\x00")

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

	info := bindataFileInfo{name: "cloud.json", size: 3540, mode: os.FileMode(420), modTime: time.Unix(1453795200, 0)}
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
