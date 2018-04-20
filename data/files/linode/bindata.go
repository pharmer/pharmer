// Code generated by go-bindata.
// sources:
// cloud.json
// DO NOT EDIT!

package linode

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

var _cloudJson = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xac\x97\xdf\x6f\xa3\x38\x10\xc7\xdf\xf3\x57\x8c\x78\xe6\x72\x60\x08\x81\xbc\x55\xd7\xb4\x77\xed\x6d\x5a\xa9\xa9\xb4\xd2\xaa\xaa\x5c\x98\xa4\x6c\xc0\xa6\x86\xa4\xca\x56\xf9\xdf\x57\xd0\x34\x10\x03\x86\x44\xfb\xd2\x86\xc1\x9e\xef\x87\xf9\x61\x8d\x3f\x06\x00\x1a\xa3\x31\x6a\x13\xd0\xa2\x90\xf1\x00\x35\x3d\xb7\x21\xdb\xa4\xda\x04\x7e\x0c\x00\x00\xb4\x00\x37\x85\x19\x40\x7b\xa3\x5f\xbf\x12\xc1\x03\x6d\x00\xf0\x54\x6c\x10\xb8\x0c\x39\x2b\xf7\x7c\x14\x7f\x01\xb4\x88\xfb\x34\x0b\x39\xcb\x15\xae\x04\x65\xab\xc5\x5a\x64\x3a\x5c\x4e\xf7\x7e\x0e\x7b\xf3\x05\xa6\x51\x5a\x7f\x71\x86\xa5\xbf\xc2\x64\x1a\xda\xfe\xe1\xa9\xf8\xbf\xd3\xdb\xb5\xe6\x7c\xb5\xe5\x40\x74\xb8\xb9\x6f\x54\x32\xd5\x4a\x66\x7f\xa5\x4b\x1a\x45\x34\xd5\x61\xfe\x5d\x87\xc7\x87\x8b\x26\x35\xa2\x14\x23\xfd\xb5\xae\x04\xc6\x9c\x65\x3a\xfc\x73\xd1\x2a\x66\x29\xc5\xac\xfe\x62\x17\x59\x44\x59\x46\x75\xb8\x6e\x17\xb3\x95\x62\x76\x7f\xb1\x19\xbe\x53\xb1\xd2\x61\x76\xd3\xaa\xe5\x28\xb5\x9c\xfe\x5a\xff\x73\x16\x70\xa6\xc3\x94\x2d\x23\xca\x02\x1d\x1e\x6f\x9b\xf4\xc6\x4a\xbd\xf1\x89\xb5\xd8\x56\x89\xae\x52\xc5\xed\xaf\xf2\x10\xb2\x25\x4d\xb8\x40\x1d\x1e\xae\x9b\x94\x3c\xa5\x92\x27\x29\x1d\x5a\x3b\x64\x69\x46\x99\x8f\xf3\x6d\x82\x0d\x0d\x9e\xae\xd6\x45\x43\x95\xce\x03\x4c\x7d\x11\x26\x87\x68\x17\xe7\x0a\x98\x06\xa9\x14\x8b\x4f\x33\x5c\x72\xb1\xcd\x17\x5c\x23\x43\x41\x23\xb8\x5f\x8b\x84\xa7\x58\x59\x94\xe4\xae\xcd\xf2\x53\x68\x7c\xf4\x1c\x84\xe9\x4a\x9b\x00\x31\x1a\x63\xf3\x05\x66\x74\x92\x39\x96\xdb\x82\xf6\x6f\xb8\x7c\x85\x6f\x18\xe7\x8f\x5d\x58\xce\x89\x5c\x9d\x11\xb3\xc8\xd8\x71\x4f\xe6\x22\x12\x97\x45\x64\x2e\x5b\xcd\x45\xba\xb8\x1c\xd3\xb6\x8d\x93\xb9\x6c\x89\xcb\x31\x64\x2e\x4f\xcd\x65\xf5\xa9\x30\xe3\x74\x30\x57\x4e\xa4\x51\x23\x23\x86\x1a\xcd\xee\x42\x23\x86\xed\x9e\x81\x56\xa9\xa9\x4f\x36\x52\x67\xb3\xd4\xe9\xec\xcc\x66\x8e\xf6\x47\xfa\xb2\x56\x67\x96\x12\xac\x33\x9d\xb6\xe1\x39\x67\x81\xc9\x1d\x60\xd7\x1a\xc0\x55\x81\x75\x26\xd3\x35\x3d\x72\x16\x98\xdc\x02\x6e\xad\x03\x1c\x15\xd8\xa8\xb3\x01\x08\x71\xcf\xcb\xa5\x5c\x67\x66\x2d\x99\xa6\x47\x54\x6c\x4e\x67\x99\xd9\xa3\xf1\x79\xe9\x94\xfb\x93\xd4\xf2\x69\xb9\xb6\x8a\x6d\xdc\x59\x69\x9e\x39\x3a\x2f\xa3\x66\xad\xd6\x6a\x39\x1d\x3b\xca\x6a\x73\x3b\x4f\xdb\xd1\xc8\x3a\x2f\x70\xb5\xd3\xc3\xa9\x45\xce\x34\x47\xca\xb4\x7a\x7d\x7a\xa1\xe5\x5c\xeb\xea\x52\x43\xee\x86\xda\xd1\x66\x8e\x2c\x47\x1a\x46\x7c\x81\x01\xb2\x2c\xa4\x51\xc3\x28\x92\x08\xbe\x09\x03\x14\x25\x9f\x56\x75\x99\x44\x74\x7b\xc5\x45\x4c\xb3\x7c\xc1\x22\xc4\x28\x28\xdf\x53\xc6\x78\x56\xcc\x52\xb9\xe3\x8f\x72\x2e\x4a\x5e\xa9\x88\x51\x0c\x69\x92\xa4\x3e\x0f\x70\xe8\xf3\xf8\x6f\x3f\x5a\xa7\x19\x8a\xbf\x4a\x9c\xdc\xe5\xc1\x5b\xdb\xb6\x80\xa5\xf2\x96\xfd\x8e\xdd\x01\xa4\xe0\x3a\x9e\xcd\x4a\x9a\xcf\xab\x99\xcf\xd9\x22\x5c\x16\x5f\xf9\xdf\xec\xee\x72\xfa\x3c\xbf\xbb\x9d\xce\x2a\xea\xb9\x1b\x2e\xe2\xf2\x56\xf7\x9c\xf1\x15\xb2\xe3\x15\x3f\xd3\xcf\x4c\x36\xbc\x8a\xe8\x0b\x46\xfb\xc9\x55\x7e\x17\xb2\x64\x5d\x04\x30\xa1\x69\xfa\xce\x45\xa0\x1d\xde\xee\xf6\xbf\xe4\x09\x72\xb5\x7e\x41\xc1\x30\x6b\x1a\x1f\x37\x28\xd2\xaf\x3b\xd9\xd0\x1d\x56\x6a\x69\x7f\x07\xad\xa4\x22\xbf\x87\x4e\x20\x13\x6b\xac\x06\x3a\xbf\x87\xd6\xac\x6f\x74\x6f\x1b\x54\xd1\xe4\x2a\x3f\x12\xf7\xfa\x88\x2f\x68\x94\x36\xa8\xcb\xe6\x42\xbe\x30\x1e\xeb\xe7\x21\x19\xec\x7e\x07\x00\x00\xff\xff\x2d\xf5\x85\xbe\x77\x0f\x00\x00")

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

	info := bindataFileInfo{name: "cloud.json", size: 3959, mode: os.FileMode(420), modTime: time.Unix(1453795200, 0)}
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

