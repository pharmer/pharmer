// Code generated by go-bindata.
// sources:
// cloud.json
// DO NOT EDIT!

package aliyun

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

var _cloudJson = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x5c\x4f\x8f\xdb\xb6\x13\xbd\xfb\x53\x10\x3a\xfd\x7e\x80\x6d\xd4\xb2\x91\x2e\xf6\x96\xa4\x68\x92\x02\x6d\x50\x24\xcd\xa5\xc8\x81\x96\x59\x99\xb5\x44\xb9\xfa\xb3\x5d\xef\x62\xbf\x7b\x21\xc9\xb1\x25\x92\x12\x87\x14\x6d\x39\xb2\x2f\x6b\x40\x7c\x2b\x8e\xde\x9b\x21\x87\x43\x4a\xcf\x23\x84\x1c\x86\x43\xe2\xdc\x23\x07\x07\x74\x97\x31\x67\x9c\x5f\x23\xec\xc1\xb9\x47\x7f\x8e\x10\x42\xc8\x59\x91\x07\x67\x84\xd0\xd7\xa2\x25\x26\x3e\x8d\x58\x72\x68\x7d\x2e\xfe\x22\xe4\x04\x91\x87\x53\x1a\xb1\xfc\x56\xef\x31\xf3\x9f\xd6\x51\x36\x46\x6f\xd7\x94\xe1\xe2\x9e\x05\xa8\xfc\xef\x1c\xe2\xb1\xc9\x7a\x8f\x3a\x36\x3f\x45\x8c\x1c\x6f\x5d\x5c\xaa\xe2\xf6\x57\xbf\x16\xbf\x2f\xe3\xe6\xfe\x7f\xa7\xcc\x5f\xe1\xa8\xbd\xfb\x7f\x4a\x90\xaa\xf7\x3d\x6c\xb2\x84\x77\xff\x86\xd0\xbf\x29\xf3\xdb\xbb\x5f\x96\x20\x55\xf7\x7b\xd8\xe4\x78\x9b\xfa\xf5\x65\xc3\x75\x0f\x6e\xee\xfb\x88\xf9\x9b\xa8\x6a\x4a\x5d\x26\xa1\xb9\x41\xa6\x3d\x4e\x87\xa9\x4f\x6b\xc2\x9e\xd6\x84\xb5\x53\x95\xec\x51\x2a\x0b\xbe\xe1\x04\xb2\x0e\x0d\x3a\xa6\xd1\x80\x7a\x11\x43\x5f\x70\x10\x90\xdd\x18\xfd\xf1\xe9\xb5\xcc\xbc\x2c\x99\xfc\x4b\x92\x74\x32\x6b\x35\xee\x80\xaa\x9b\x76\xbc\xac\x61\xd8\x17\x1a\xfb\x94\x51\xdc\x66\x12\xc1\x10\x93\x4a\xd4\x04\xeb\x90\xc2\x7c\xbc\x8d\x62\x22\xeb\x17\x6f\x27\x49\x94\xa5\x6b\x40\xe7\x75\x68\x9d\x14\xae\x4d\xcb\x9b\x30\xf3\xd7\x98\xaa\xbc\xa9\x44\xa9\xbd\xa9\xc4\x49\xbc\x69\xdf\xc0\xc7\xde\xa1\x41\x23\xf8\x7e\xca\x96\x55\x53\x8e\x96\x86\x04\xa2\xe2\x01\xa5\xa3\xe2\xe7\x68\xb3\x8b\xc6\xe8\x17\xbc\xc5\xac\x41\x48\x16\xc5\x50\x21\x2b\x50\x2d\x57\xda\xad\x58\x1e\x57\xaf\xb3\x24\x8d\x71\x40\xa5\x72\xd5\x7c\xc1\x85\xbb\x94\xab\x63\xc9\xcf\x31\x66\x9b\xbf\xb2\x38\x1d\xa3\x77\x24\x0e\x31\xdb\xc9\x4c\x21\xd9\xc4\x23\x2c\x37\x55\x41\x49\x15\x28\x98\x71\x98\x47\x29\x4b\x52\xcc\x3c\xf2\x79\xb7\x25\x92\xd9\x34\xd9\x64\x45\xa7\x5e\x32\xf5\x66\xd3\x00\xc7\x7e\x25\xe4\x56\x24\xf1\x62\xba\xfd\x66\xbf\x1c\xe4\xe1\x94\xf8\x51\xbc\xcb\x11\xef\x08\x23\x71\xf1\xbc\xe8\x03\xfa\xdf\x87\x8f\xe8\xe3\x36\xa5\x21\x7d\x22\xab\xff\x57\xfe\x61\x9b\x77\x79\x77\x7c\x74\x1c\x3a\xf7\x68\xf6\x8a\xe3\xa2\x35\x92\xe5\x93\x11\x77\x55\x98\x50\x90\x74\x3e\x46\xf2\xb1\x1f\xc9\x06\xb8\xea\x58\xda\x2e\x7c\x9d\xd8\x24\xc4\x41\xa0\x22\x96\x03\x35\x13\xfb\x5b\xfe\xa3\x43\xee\x5d\x3b\xb7\x8d\x0c\x2a\xa8\x02\x13\xe0\x42\x3c\x4b\x00\x19\x7a\x56\xd5\x93\x8a\xa7\x9f\xbb\xc3\x75\x2d\x77\x1a\x92\x15\xcd\x42\x15\xb5\x3c\xaa\x83\x73\x09\xfc\xaa\x42\xf7\x0c\xee\xf5\x08\xf2\x2f\x1e\x65\xcb\xc1\x5e\x2d\x0c\x09\xb8\x14\x37\x22\xf3\xe9\x1c\x40\xa1\x0c\x56\xe5\xf0\x57\x12\x46\xf1\x0e\xa5\xbb\x2d\x41\x64\xae\xf6\x1b\x57\x31\x2e\x99\x04\x26\xbf\xd6\x33\x22\x9a\x4b\xdc\x9a\xf9\x17\x93\x27\xab\xa2\x80\x24\x31\x16\x64\x71\x86\x71\x72\x50\x72\x00\x86\x5a\x09\x0a\x2e\x88\x7b\x86\x9c\x68\x50\x82\xa8\xd3\x2a\xe6\xb6\xa5\x55\x8a\x01\x4b\x2f\x8b\xba\x7a\x35\x60\x33\x88\xf9\x04\xc2\x67\xb5\xaa\x69\xf7\xca\x05\x59\x4c\x5d\x88\x22\x12\x58\xa3\x24\x0b\x0b\x92\xc8\x56\xfc\xc2\xb2\x1a\x40\x69\x6d\x9d\xac\x41\xca\x02\x46\x8a\x00\x83\x93\x62\x94\xe8\xf4\x4d\x0b\x88\x14\x63\x4a\x0c\x66\xb6\xbe\x09\x81\xb9\x89\xb9\x97\x18\x64\x5f\xbd\x52\x12\xce\x20\xf9\x8f\x04\x65\xb8\xca\xe2\xf9\x19\x70\x81\x28\x9c\x41\xbc\x4d\x82\xb2\x54\x7b\x1b\x70\x81\x24\x04\x15\x48\x24\x28\x8b\xd5\x37\x15\xbd\xa7\xae\x8f\x84\xa0\xfa\x88\x04\x65\xc9\xbd\x4e\x90\xa7\xd9\x62\x86\xc1\x92\x24\x29\xae\xca\xce\x1b\x1c\x60\xe6\x91\x55\x39\xd8\x87\x4c\x99\x28\x5d\xfc\x68\xcf\x60\x99\x92\x14\xa7\xc5\x8c\x76\x35\xed\x12\xa8\xb9\x03\x52\x23\xe0\xb4\xa8\x99\x0b\x69\xd3\xa5\x27\x92\xf9\x33\xc3\x98\xe9\xc2\x0b\x4f\xcb\x77\x40\x8a\xba\x52\x21\x41\xe9\xc5\x11\x47\xca\x77\x10\x45\xc0\x20\xea\x14\x43\x06\x69\x64\xaf\xbc\xb0\x19\xa8\x14\x2f\x83\x89\xd3\x75\x50\xd2\xc2\x66\xca\x41\xf7\x56\xfb\x55\x88\xf2\x23\x4c\x14\x01\xa6\x21\x8a\x30\xdc\xdf\x0a\x5c\x0a\x51\x40\x92\x98\x0b\xc2\x8f\x1d\xb7\x02\x70\xbb\x1c\x80\x95\x96\x04\xa5\x21\x08\x1f\x20\xb7\xf8\x68\x17\x04\xb0\x3f\x02\x38\x76\xd2\x3c\x8b\x70\x72\xdc\xe6\x90\x76\x39\x52\x5a\x3d\xf2\x25\x57\xa3\x8e\xe9\x20\xc6\xec\x26\x46\xab\x18\xb0\xf9\xbc\xc3\x74\x7e\x8e\x13\x6e\x03\x52\xc4\x85\xe5\xbd\x12\x58\xb3\x26\xae\x8d\x62\xc3\x75\x8b\x02\xca\x7b\x25\x30\x0d\x51\x8c\xca\x1c\xd7\xad\x0a\x48\x13\x73\x45\xce\xb1\xf7\x32\x28\x3d\x20\x99\x2f\x64\x8f\xa1\x51\x11\xfd\x92\xd7\x75\x0b\xd2\xf5\x64\x90\x6a\x1e\xb9\x2d\x44\xb4\xe4\x80\x4d\x22\x1d\xe6\x90\x73\x6c\x69\x0e\x48\x11\xd8\xb6\x97\x6a\xd7\xab\xae\x89\x72\xcf\xeb\xe2\x4b\xaf\xb0\x2d\x2f\xd5\x8e\x97\x82\x15\x93\xd2\x6b\xdf\xb4\x40\xb6\xbb\x54\xbb\x5d\x0a\x5a\x4c\x8a\x9f\x7d\xd3\x02\x22\xc5\x9c\x12\xfd\x6a\x57\xdf\x84\x00\x66\xdd\xd6\x4d\x2e\x55\xe8\x70\x84\x5c\x7e\xe0\xc0\xe2\xa6\x43\xd8\xe8\x97\xa8\x7b\xa5\x24\x81\xd4\xec\x45\x50\x97\xb7\x95\xf4\x08\x3a\xf5\x89\x93\x04\x54\x25\x97\xa0\x4c\xdf\x55\x3a\x79\xa2\x6a\x8f\x18\xf5\xe8\x21\x82\x2c\xd1\x62\x3f\x5b\xb4\x45\x8b\x0b\xca\xd4\x64\xb0\x0e\x41\xa3\x7b\x5c\xf7\x0c\x2c\x80\x38\xb0\x73\x7e\xed\x0c\xbb\x4c\x17\x72\x3a\x32\x81\x2c\x94\x45\x90\x45\xcf\xea\xf9\x68\x64\x02\x5a\x9a\x4a\x50\x96\x5c\xeb\x04\x65\x9c\x4b\x71\x2d\xc8\xdb\x8b\x22\xc8\xd2\x61\xf1\x41\xf3\x0a\x49\x20\xda\x5f\x43\xd4\x0c\x5a\x9e\x5d\xd3\xd7\x9d\x2d\x05\x6d\x0a\x49\x15\x44\x90\xc5\x24\x52\xb1\xb7\x7a\x6a\x02\x1e\x2d\x2c\xb5\xde\x46\xe1\x16\x7b\x69\xb9\xae\x78\x54\xaf\xb5\xd4\xdb\xc9\x67\x58\x58\x1c\xbe\x6a\xb2\xc9\x96\x24\x66\x24\x2d\x3e\x69\x52\x52\xe4\x3c\x90\x38\xe1\x0c\x7b\x3e\xde\x75\xdf\x9a\x3f\xfa\x6c\x7a\x37\xfd\xa1\xda\x23\x47\x9e\xd0\x5e\x7e\xa5\xec\x78\xb3\xfd\xb7\xca\xee\x51\x1a\x67\xe4\x70\xf5\x65\x54\xfd\xcd\x2d\x7e\x19\xbd\xfc\x17\x00\x00\xff\xff\x32\xcf\x67\xb0\xfa\x4c\x00\x00")

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

	info := bindataFileInfo{name: "cloud.json", size: 19706, mode: os.FileMode(420), modTime: time.Unix(1453795200, 0)}
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
