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

var _cloudJson = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x58\x51\x6f\x1a\x39\x10\x7e\xcf\xaf\xb0\xf6\x99\x50\x58\x28\xc7\xe5\x8d\x26\x6d\x2f\x3a\x9a\x70\x21\xea\xe9\x74\xaa\x2a\xe3\x1d\x36\x3e\xbc\x9e\x3d\xdb\x4b\x44\xa3\xfc\xf7\x93\x0d\x81\x5d\xef\x62\x10\xca\xc3\xa9\xca\x4b\x2b\x76\xbe\xf5\xf7\x79\xe6\xf3\xec\x38\x4f\x67\x84\x44\x92\x66\x10\x5d\x90\x28\xe1\x29\x37\x54\x20\x03\x2a\xa3\x96\x8d\x80\x5c\x46\x17\xe4\xef\x33\x42\x08\x89\x12\x58\xba\xa7\x84\x44\xff\xd2\xe8\x8c\x90\x6f\x0e\xa3\x20\xe5\x28\xf5\x16\xf7\xe4\xfe\x25\x24\x12\xc8\xa8\xe1\x28\xed\xd2\xa3\x4c\x1b\x50\x09\xcd\x36\x2b\x6c\xdf\xb3\x41\x9a\xe9\x78\xf7\xfc\x07\x4a\xd8\xad\xe6\x1e\x39\xc0\xe6\xe7\x37\xf7\xff\x73\xeb\x74\xae\xde\x21\xae\xde\xf1\x5c\x1f\xa8\x4c\xa9\x40\x05\x4d\x5c\x33\xa1\xba\x41\x2e\x07\x38\x9a\xeb\x93\xa2\x72\x31\x2f\x94\x69\xe2\x9a\x2b\x1a\xe6\x72\x80\xa3\xb9\xc6\x28\x13\x94\x4d\x44\x02\x65\x98\xc8\x01\x8e\x26\xba\x81\x47\xf2\x17\xaa\x45\x13\x95\x5c\xb1\x30\x95\x03\xbc\x16\x55\xd8\x82\x0e\xf0\x5a\x54\x61\x07\x3a\xc0\xd1\x54\x53\x2a\x89\x75\x06\xe3\x9a\x61\x13\x9f\x9e\x63\x38\x8b\x0e\xf0\xaa\x7c\xe1\x54\x3a\xc0\xf1\x7c\x5c\xa6\x34\xdf\x73\xc2\x74\x9a\x1f\xd8\x9b\x05\x1c\xcd\x75\x8f\x0a\xa5\x69\xdc\x95\xc1\x03\x67\xd9\x01\xaa\x4c\xdb\x16\xc9\xa5\x36\x54\x32\xb8\x5f\xe5\xd0\xd0\x28\xf5\xa2\xb0\x14\xdd\x74\xb6\x63\x48\x40\x33\xc5\xf3\x17\x65\x95\x18\xa3\x06\x52\x54\x2b\x1b\xf8\x0c\x12\x14\x15\x64\x52\xa8\x1c\x75\x29\x49\x2c\xb7\x8b\x76\x77\x5b\xa1\x59\xe5\x77\xc2\xf5\x22\xba\x20\xbd\x4e\x63\x56\x36\x92\xe2\x80\xa4\xf8\x44\x49\xb1\x27\x29\xf6\x25\xf5\x83\x92\xfa\x01\x49\xfd\x57\x92\xd4\xf7\x25\x0d\x82\x92\x86\x01\x49\xc3\x13\x25\xf5\x3d\x49\x43\x5f\xd2\x30\x28\xa9\x3b\x08\x99\x69\x70\xa2\xa8\xa1\xef\xa6\x81\xaf\xaa\x1b\xce\x54\x76\x1e\x14\xe6\x87\xcb\xd2\x7e\xe3\xe9\x03\xf9\x02\x99\xfd\x79\xa0\x7c\x75\x59\x61\x97\xf7\x42\x36\xef\x9d\xea\xf3\xae\x2f\xab\x57\x73\x7a\x2f\x3e\x90\xad\xa0\x32\x3f\x7c\x64\xb6\x7c\x67\xd5\x65\xfd\x1a\x3e\x80\x21\xbb\xf7\x4f\xf5\x7b\xa9\x66\x9b\x33\x58\x73\x7c\x3f\x6c\xf9\x41\xa8\x33\x0c\x4e\x6e\x0d\x1d\x4f\xd7\xa0\xde\x1c\xc2\x0d\x2b\x3b\x0f\x2a\xf3\xc3\x47\x56\xd1\x3f\x8a\x75\x59\x71\xe7\xd0\x51\x8c\x43\x95\xac\xc5\x8f\x14\x56\x2b\x64\x37\xae\x55\xb2\xd7\xef\x78\x9f\xd8\xa6\xc1\xb8\xb5\xfb\x5d\x99\x6e\x89\x37\x84\x12\x6f\x52\x24\xde\x8c\x45\xbc\x99\xa4\xf9\x63\xbd\x2f\x4b\x71\x1c\xae\x9e\x17\x3f\x32\x4b\xbd\xda\x57\x30\xae\x15\xf0\x7d\xe7\xff\x97\xa5\xed\x48\xc3\x14\x24\x20\x0d\xa7\xa2\x61\xa0\xc9\x15\x2e\x79\x02\xca\x26\xe1\x6a\x7d\xa9\xbc\xdd\x5e\x2a\x5f\xb6\x98\x0b\xba\xfa\x84\x2a\xa3\xc6\xdd\x5f\x38\x88\x64\x17\xa7\x52\xa2\x71\x73\x99\x5d\xfe\x69\xa7\x29\x7f\xa0\x2a\x03\xd5\xa6\x79\xae\x19\x26\xd0\x66\x98\xbd\x63\xa2\xb0\x57\xbe\xf3\x9d\x28\xbb\x64\x79\x2b\x8d\xaf\x25\x52\xfb\xaf\x6c\xde\x78\xde\x0a\x71\xba\xaa\x79\xdf\xa9\x59\xdf\x90\x19\xca\x39\x4f\xdd\x5e\xaf\x3f\x5f\xdf\x8f\xc6\xb7\x97\x1f\x47\x37\xdf\xef\x6f\x7f\xff\x78\x53\xd2\x60\x17\x43\x95\xf9\x17\xed\xef\x06\x17\x20\xab\xb8\x7f\xf4\xcb\xd0\x59\x0b\x09\x3a\x03\x27\x75\x02\x4a\xa3\xa4\x82\x8c\x18\x03\xad\xc9\x7d\x1d\xcb\x65\x5e\xb8\xe4\xe6\x54\xeb\x47\x54\x49\xb4\x8d\x3e\xef\x29\xeb\xa2\x98\x81\x92\x60\x40\x7f\x05\xa5\x9b\xef\xf5\xcb\x75\xc4\x2e\xbc\xec\xb6\x7f\x69\xbf\xdf\x7b\x3a\xfc\xf0\xfa\xcf\x09\xa5\x6a\x26\x60\x1f\x18\x55\x40\x2d\xf1\x09\xcc\x69\x21\xcc\x34\x07\xe6\x39\x00\x93\x69\x31\x93\xe0\x76\xd6\xed\xb4\xe3\x7e\xbf\xdd\x69\x77\xde\x75\x07\x15\x2f\x83\x5a\x72\x06\x15\x64\xa7\x01\x47\x85\xbb\x00\xc0\x0d\x26\x70\xc9\x13\xa5\x37\x7a\x4a\x10\x90\x74\x26\xe0\x72\xed\xb1\x2f\x28\xb9\x41\xc5\xa5\x2b\xf8\x8b\x9d\xa2\x7d\xf0\x31\xa6\xe9\x1a\xdb\xb8\xa8\x65\xdd\x0b\x11\xeb\xc0\x15\x68\xc3\xe5\xf6\x8e\xf2\x42\x79\x0e\x82\x6a\xc3\x99\x06\xaa\xd8\x43\x45\x40\x39\xb0\x59\xfd\x0e\x72\xc1\x19\xd5\xe5\x2b\x80\x4d\xb2\xd4\x53\x50\x4b\x50\xd7\x93\x52\x8e\xba\x9d\xa8\x8a\xb9\xc2\x8c\x72\xc7\xbe\x39\x69\x6d\x9b\x34\x51\xc9\x63\x92\x71\x6d\x6d\x71\x89\xd2\x28\x74\x16\xbd\xa1\x19\xe8\x9c\x32\x18\xf3\x39\xb0\x15\x13\xd0\x1a\xf3\x8c\x9b\x3b\x2a\x53\x50\xad\xe9\xba\x44\x23\xc6\xb0\x90\xa6\x65\x0d\xcd\xb5\x01\x69\xbe\xa2\x28\x32\x18\x5b\xa7\xb7\xae\x36\x36\x30\xa8\x68\x0a\x97\x82\x6a\xdd\xba\x03\x8d\x85\x62\xf0\x47\x81\x86\x96\x45\x64\xd4\x8a\xf3\xdc\x31\x70\x55\x8f\xfb\x95\xce\x07\xe6\x11\xd5\x62\x52\xea\x53\x73\x41\xa5\x04\xb1\xb7\x92\x23\x01\xca\xec\xab\x39\x6a\x77\xac\x61\xc6\xa9\x6c\xe2\x41\xc1\xd9\xaa\xcc\x26\x51\x36\x98\xe6\x4f\x98\x3d\x20\x2e\xdc\x51\x1e\x15\xe6\x41\xee\xf3\xcd\xdd\x8c\x32\x0b\xf8\xb1\x0f\x30\x9a\x5c\x6b\x57\xd8\x0f\x54\x73\x36\x2a\x12\x6e\xf6\x42\x37\xfb\x19\x19\xc3\x59\x1d\x94\xa3\x10\x53\x01\x90\x5f\x4b\x03\x6a\xe9\xfa\x64\xaf\xd1\xc8\x93\x62\x26\x38\x73\x4e\x2a\x9f\xe8\xc6\xcf\x6b\xb5\x87\x0c\xdb\x9d\x50\x0f\xa9\x84\xdf\x7a\x48\x43\xea\xdf\x7a\xc8\x5b\x0f\xf9\xf9\x7b\x88\x9d\x52\xce\x9e\xff\x0b\x00\x00\xff\xff\x50\x87\x3e\x96\xb3\x18\x00\x00")

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

	info := bindataFileInfo{name: "cloud.json", size: 6323, mode: os.FileMode(420), modTime: time.Unix(1453795200, 0)}
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
