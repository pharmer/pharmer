// Code generated by go-bindata.
// sources:
// cloud.json
// DO NOT EDIT!

package digitalocean

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

var _cloudJson = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xe4\x5a\x5f\x93\x9b\x36\x10\x7f\xbf\x4f\xa1\xe1\xf9\x70\x8d\xc0\x0c\xb9\xb7\x6b\x9b\xa4\x99\xa6\x77\x99\xa9\x5f\x3a\x9d\x4c\x46\x27\x64\x42\x0d\x12\x91\xb0\x33\xee\xcd\x7d\xf7\x0e\xf2\x1f\x40\x92\x85\xcc\x5c\x1f\x62\x5e\x92\x43\xbb\xe8\xb7\xfc\xb4\xda\xd5\xca\xfb\x7c\x03\x80\x47\x51\x49\xbc\x3b\xe0\xa5\x79\x96\xd7\xa8\x60\x98\x20\xea\xdd\x36\x12\x42\xb7\xc2\xbb\x03\x7f\xdf\x00\x00\x80\x97\x92\xad\x1c\x06\xc0\xfb\x86\x8e\x7f\x55\x9c\xa5\xde\x0d\x00\x9f\xe5\x0b\x9c\x64\x39\xa3\xed\x3b\xcf\xf2\x5f\x00\xbc\x82\x61\x54\xe7\x8c\x36\x38\xf7\xa5\xa8\x09\x4f\x51\x09\xc2\xc3\x2c\xa7\x37\x1b\x31\x2a\x45\x67\xfc\x5f\x46\x49\x3b\x9f\x1c\x92\x0a\x87\xc7\xcf\xf2\xff\x97\xdb\xf3\x68\x3f\x23\x9a\xa1\x82\x71\x02\x02\x13\xda\x53\xc1\x03\x2b\x9a\x54\x70\x46\x7b\xc7\x11\x5d\xaf\x36\xbc\x36\xa3\xad\x38\xb2\xa3\x49\x05\x67\xb4\x8f\x8c\xa6\x8c\x9a\xa1\x0a\x46\xed\x50\x52\xc1\x19\xea\x81\x7c\x07\x7f\x31\xbe\x36\x83\xd1\x1d\xb6\x83\x49\x85\xcb\xc1\x8c\x0e\x42\x77\xd8\xee\x20\x52\xc1\x19\xec\x4f\x44\x41\xb3\x6c\x38\x17\x98\x01\x68\x42\x14\x2b\x06\xad\x88\x52\xc1\x1d\x31\xa7\x19\xaa\xce\xba\xa4\xc8\x2a\x3b\x99\x52\xc1\x19\x6d\xc9\x38\xa3\x35\x33\x63\xd5\x6c\xc0\xfd\xa5\x42\x1f\xeb\xb4\xdb\x73\x2a\x6a\x44\x31\x59\xee\x2a\x62\xd8\xf3\x62\xbd\x69\x20\x82\x38\x7b\x6a\x21\x52\x22\x30\xcf\xab\xa3\x71\x7d\x21\x46\x35\xc9\x18\xdf\x35\x92\xf7\x84\x12\x8e\x0a\xf0\x69\xc3\x2b\x26\x48\x47\xa9\x6a\xa6\x4d\xda\x8f\x41\xa5\x77\x07\x82\xb8\x85\xc8\xc5\x5a\x8e\xcc\x07\xa2\x48\xbb\xa6\x40\x09\x3b\x40\x09\x0c\x40\xd9\xba\x40\xd9\x5f\x40\xd9\x02\xfb\x67\xa8\x3c\xf7\xe6\x17\x2b\x16\x28\xcf\x3d\xfd\x9e\x17\x98\x17\xe2\xd6\xcc\xb7\x8d\xee\x91\x6c\x07\x2a\xdb\x2a\xd9\xe1\x34\xb9\x86\x16\xae\xe1\x48\xae\xa1\xc2\x35\x54\xb9\x8e\xa6\xc9\x75\x68\x23\x3b\x1c\xcb\x76\xa0\xd2\x1d\x6a\x7c\x87\x70\x9a\x84\x47\x89\x85\xf0\xbe\xf0\x12\xc2\x63\x85\xf0\x28\xd1\x1c\x3c\x99\x77\x60\x2b\x4e\x9a\xe9\x53\xef\x0e\xd4\x7c\x43\xac\x16\xdb\x0c\x7e\xa5\xed\x18\xa9\xd6\x4e\x34\xcd\x2c\x02\x58\x9e\x67\x5b\x91\x9a\xb3\xc8\x7c\xb6\x50\xc9\x9c\xe8\x56\x8b\x6d\x9e\x1b\x8f\x76\xdd\xb9\x42\x78\xac\x3b\x6f\x34\x72\xab\xd9\x62\xc3\xd8\xd0\x10\x29\xe6\x6a\x81\x21\x99\xa6\x77\x60\x3f\x88\xcf\x92\xdd\x17\x9e\x89\xb1\x7a\x52\x83\xf3\x21\x2e\x5f\x9b\x3b\x95\xab\xd7\xe2\x06\x5a\xa8\x81\x17\x47\x73\xb8\xb8\x12\x5a\x22\x0b\x2d\xd1\xc5\x3b\x6f\x71\x2d\xde\x92\x58\x68\x49\x54\x5a\x1c\x8a\xcc\x2b\xd9\x46\xa5\x1f\x40\x5b\x48\xd7\xe4\xdd\xb0\xfe\x5b\x9e\x7d\x05\x7f\x90\xb2\x79\x1c\x8a\x44\x01\xd4\x7c\x2b\xb4\x17\x34\xff\x37\x67\x3d\x8e\xc6\x25\xc3\xd2\xb7\x5e\x72\xa8\x62\x47\xee\xd4\x58\xa5\x7b\x5f\x38\x32\x79\x97\x3e\x84\xb6\x13\x87\x26\x77\xb4\x38\xd4\x6a\x57\xa8\x05\xd8\x85\x7d\xc7\xfc\x10\xab\x6d\x2d\x45\x55\xb1\x23\x77\x6a\x08\xd6\x53\xf6\x9b\xd1\xab\x6d\x3d\x5e\xaa\x62\x47\x7b\xd5\xd8\xa8\x9f\x2d\xbb\x47\x8c\x8b\x0c\x16\x7e\xb0\xc5\xd5\xc6\xb7\xdd\x64\x19\x75\x1c\x2f\xab\x06\x73\xfc\x75\x1e\x23\x8f\x94\xd9\x9c\xd7\xa8\x63\xa6\x55\xf3\xcf\xc1\x33\xc2\x75\xd3\x1a\x3a\xd0\x1a\x0e\xd3\x1a\xaa\xb4\x4e\xf4\x7e\x41\xf8\xd0\xc1\x5b\x0d\x3a\x8e\xb7\xa8\x13\xa7\xd5\x96\x11\x8c\x3a\x8e\xf5\xd3\x44\x4b\x74\xe1\x87\x0e\x29\xcb\xa0\x73\x38\x38\x0d\xa5\xac\xc9\x7a\x6b\x24\x29\xb3\x95\x26\x46\x1d\xc7\xb2\x76\xaa\xbf\x11\x0a\x3f\xde\xbb\xa2\xad\x6c\x31\x2b\xed\x99\xd5\xca\x3a\xbd\x36\x99\xe8\x55\xae\xf0\x93\x7d\xaa\xb7\x27\x2e\x93\x92\xf9\x60\xad\x17\x02\xf1\xd5\xff\x02\x78\x6a\x31\xc0\x9c\xa4\x84\xd6\x39\x2a\x0c\x0d\x06\x15\x67\xdb\x3c\x25\xbc\x61\xf4\xd7\x7d\xf3\xd2\xe3\xa9\x79\xe9\x48\x58\x55\xa0\xdd\x3b\xc6\x4b\x54\xcb\x06\x9c\x9c\x14\x69\x2b\x47\x94\xb2\x5a\x76\x4a\x34\xd3\x3f\xb7\x36\x55\x5f\x11\x2f\x09\x9f\xa1\xaa\x12\x98\xa5\x64\x86\x59\xf9\x13\x2e\x36\xa2\x26\xdc\x6f\x8d\x6a\xa6\xec\x7e\x8a\xf1\xb5\x94\x0a\xf5\x95\xc3\x1b\x2f\x27\x43\xa4\x5d\xfd\x75\x6c\xad\xd9\x77\x62\x61\x46\x57\x79\x26\xbf\xf5\xc3\xfb\x0f\xcb\xfb\x8f\x8f\xbf\xbc\xbd\x7f\xf8\xb2\x7c\xfc\xfd\xed\x43\xc7\x86\x66\x32\xc6\x4b\xb5\xa1\xeb\x4b\xcd\xd6\x84\xf6\xf5\xfe\x11\xc7\x26\x10\x4d\x54\xa0\x27\x22\x4d\xfd\x44\xb8\x60\x14\x15\xe0\x1e\x63\x22\x04\x58\xea\xba\x39\xad\x36\x92\xdc\x0a\x09\xf1\x9d\xf1\xd4\x3b\x49\x5f\xce\x2c\xeb\x7a\xf3\x44\x38\x25\xb5\xa9\x6d\x64\x4b\xb8\x38\xb6\x2b\xcc\x92\xd9\xbc\x5d\xad\x43\x3b\x5a\x67\x99\x52\xb2\x3d\xd4\x93\xdd\x45\xe0\x2c\xd5\x47\xbf\xa1\x6e\xe5\x79\x34\x4d\xdd\xc4\x3d\xf0\x37\x2e\xe0\x2b\x54\x08\x03\xba\x3a\x2c\xe1\xe5\x60\x1f\xbf\xa1\xe4\xe6\xe5\xbf\x00\x00\x00\xff\xff\xed\x2d\x5f\x55\x88\x27\x00\x00")

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

	info := bindataFileInfo{name: "cloud.json", size: 10120, mode: os.FileMode(420), modTime: time.Unix(1453795200, 0)}
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
	"cloud.json": &bintree{cloudJson, map[string]*bintree{}},
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

