// Code generated by go-bindata.
// sources:
// cloud.json
// DO NOT EDIT!

package vultr

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

var _cloudJson = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xcc\x59\x5d\x6f\xdb\x36\x14\x7d\xcf\xaf\x20\xf4\xac\x65\x14\xf5\x9d\x37\xa7\x49\x87\x6d\x4d\x17\x34\x5e\xf7\x30\x14\x05\x2b\x31\xae\x66\x89\xd4\x28\xd9\x85\x17\xf8\xbf\x0f\x52\x6c\x89\xa2\x48\x7d\x38\x6e\xd0\x97\xba\x11\x45\x9e\x73\x0f\xc9\x73\x2f\xa9\xa7\x0b\x00\x0c\x8a\x33\x62\x5c\x01\x63\xbb\x49\x4b\x6e\x98\xd5\x23\x42\xb7\xc6\x15\xf8\xfb\x02\x00\x00\x8c\x98\x6c\xeb\xa7\x00\x18\xff\x62\xe3\x02\x80\x4f\xf5\x3b\x9c\xac\x12\x46\x8b\xe6\xbd\xa7\xfa\x5f\x00\x8c\x94\x45\xb8\x4c\x18\xad\xc6\x7c\x4f\xbe\x81\xdf\x08\x2f\xc8\xee\x30\x44\xd3\xb1\x6a\xb5\xda\x87\xff\x31\x4a\xda\xb1\xea\x47\x96\x71\xf8\xff\xa7\xfa\x77\x6f\xea\x61\xde\x7c\x4d\x22\xbc\x62\x2a\x0c\x34\x88\x81\xa6\x63\xdc\xe0\x34\xc5\x85\x0a\xc2\x1e\x84\xb0\xa7\x43\x3c\x10\x5c\x96\x29\x51\x61\x38\x83\x18\xce\x74\x8c\x77\xac\x00\x0b\xba\x22\x29\x51\xc6\xe2\x0e\xe2\xb8\xd3\x71\x16\x65\x8a\x69\x89\x55\x18\xde\x20\x86\x37\x03\x23\x2b\x4a\xc2\x63\x9c\xa9\x50\xfc\x41\x14\x7f\x8e\x62\x34\x66\x54\x05\x11\x0c\x42\x04\xd3\x21\xde\x72\x4c\xd7\x8f\x1b\x5e\xaa\x50\xc2\x41\x94\x70\xc6\xf2\x4a\xd2\x24\x62\x14\x7c\xc4\x69\xaa\xd9\x90\xc3\xbb\xc5\x9a\xb1\x5d\x1e\x76\x31\xd5\x80\x0c\x07\x64\xcd\x88\xe8\x1e\xf3\x44\xb9\x8c\xd1\xf0\x7e\x41\x33\x36\xcc\x92\xad\x77\x6a\x67\x19\xde\x2b\x68\xc6\x66\xb9\x4b\x70\x96\x28\xad\x65\x58\x2b\x7b\xd6\xec\xd3\x15\xce\x19\x57\xdb\x0b\x1c\xf6\x17\x28\xe1\x34\x59\x20\xa1\x45\x89\x69\x44\x96\xbb\x9c\x28\x72\x41\xb1\xde\xd4\x4b\x58\xb0\xc8\x98\x14\x11\x4f\xf2\x23\x2d\x0b\x22\x07\xdc\x5d\x83\x0f\x8b\x3b\x13\x41\xf0\xcb\x35\x78\x78\xb8\x31\xd1\x25\x84\x60\x79\x0d\xae\xff\x6a\x3b\x46\xb8\x24\x2b\xc6\x77\x75\x30\x0f\x37\x42\x43\x5e\x81\x58\x6d\x58\x38\xeb\xfc\x1d\x27\xc5\xda\xb8\x02\x08\x0a\x1c\x72\x4e\xaa\xf1\x62\xe3\x0a\x94\x7c\x43\x94\xea\x1d\xc9\x3b\x5a\xf2\x08\x3a\xc1\x91\xbc\xe3\x1e\xc9\xdb\xf3\xc9\x23\x89\x3c\x92\xc9\x3b\xee\x89\xe4\x5d\x2d\x79\x07\x86\xde\x91\x7c\xd8\x28\xef\xcc\x27\xef\x48\xe4\x1d\x99\x7c\x08\xa5\x25\xd7\xcb\xf4\xa6\x98\x2f\x4d\x31\xb1\x99\x62\xf6\x31\xc5\x34\x61\x8a\x6e\x6e\x8a\xbe\x6b\x8a\xf6\x68\x76\xac\xc5\xec\x98\x80\xd9\xd9\x4a\xa6\x72\xc1\x9f\xa8\xbb\xa7\xd5\x3d\xb0\x42\x74\xd4\xdd\x72\x1b\xe1\xdd\xf9\xc2\x7b\x92\xf0\x81\x2c\xbc\xe5\xbe\xaa\xf2\xaf\xa9\xaf\xaf\x77\x14\xcf\x0e\x1a\x4b\xb1\x61\x23\xb0\x37\x5f\xe0\x40\xf6\x14\x4f\x56\xd8\x86\x27\x2b\xfc\x03\x8a\x1a\x68\x45\xb5\x91\xef\x35\x56\xe7\xb5\xa2\x5a\xf0\x04\xa7\x96\xd7\xad\xdd\xb3\x3b\xef\x7b\xc8\x3a\xa4\xe4\x8b\xb5\xb3\x20\xd4\x8a\xe7\xb9\xae\xdd\x58\xad\x2f\x88\x77\xc2\x9e\x47\xb2\xdb\x7a\x3d\xbb\xf5\x67\x89\x37\x20\xd1\xcb\x45\xb1\xf4\xe9\xa7\x63\x83\xd6\xf4\x05\x75\x73\x7b\xf3\xeb\x9b\xc5\xf2\x76\x34\x87\xf6\xdd\xd0\x9a\xa3\xcb\x58\xfd\xd6\x84\xa8\x77\xfa\x8e\x13\x09\x31\xa2\x93\x63\x94\x27\xbf\x6f\x48\x13\x82\x9c\x16\x96\xde\x60\x91\xe3\xfa\x9e\x22\x2c\xfb\xe4\xb0\x64\x43\x40\xbd\x35\x7d\xb6\xb0\x26\x5a\x9c\x10\x96\x73\x72\x58\x72\xfa\xe8\xfb\xdc\x99\xc2\x0a\xf4\x51\x75\xea\xeb\x43\xb5\xb1\x58\x2e\xc6\x2b\xec\xc5\x72\x31\xbf\xc4\x9e\x55\x6f\xb8\xda\x9c\x37\x71\xef\x05\xe1\xa4\xca\xdc\x85\x6d\xd8\xa3\xb5\xf9\x84\xb0\x7b\xb3\xe8\xce\x32\xdc\x17\x87\x1d\xea\x33\x8d\x0d\xfd\xc6\x54\x7d\x61\xb6\x47\xab\x7a\x45\xd8\xb2\x9f\xda\xbd\x3c\xf3\xba\xb3\x1d\x5a\x93\x8e\x32\x16\x14\xa6\x7b\x34\xc1\x4e\x88\xbb\xef\x45\xf0\x75\xe7\x1b\x41\x7d\xe4\xdd\xed\xdd\x24\xd1\xef\x71\x7c\x76\x87\x39\xea\x8f\xf8\xe2\xec\x78\xf0\x8c\xa7\xe4\xde\xcc\x78\x70\x98\x23\x9a\x76\x92\x3f\xe7\x35\x44\xff\x24\x3f\xc2\x51\x5f\x31\x75\xca\x09\x04\xcf\x79\x72\xec\xd7\x11\x08\x8e\xf0\xd4\xdf\x8a\x74\x2a\x3b\x78\xce\x9b\x85\x7e\x49\x37\xc6\x52\x5f\xc5\x74\x8a\x72\xe7\xcc\x27\x9a\x7e\x51\xee\x8c\x31\xd5\x97\x91\x9d\xc2\xe4\xbc\x07\xda\x7e\x45\x62\x8f\xf1\xd4\x97\x1a\x61\x60\xc3\x66\x7d\x06\x67\x3e\xe6\x84\xbd\x15\x1a\x1c\x99\x36\x97\x91\x11\x27\x31\xa1\x65\x82\x53\xc5\x55\xe4\x3d\x67\xdb\x24\x26\xbc\x02\xfc\xd8\x7c\xea\x3a\x0e\x98\xa7\x78\xf7\x96\xf1\x0c\x97\x55\xfb\x63\x42\xd2\xb8\x6d\xc7\x94\xb2\xb2\xbe\x48\xad\xc6\x7d\x6a\x0d\x3c\xff\x8a\x79\x46\xf8\x25\xce\xf3\x22\x62\x31\xb9\x8c\x58\xf6\x73\x94\x6e\x8a\x92\xf0\x9f\x5a\x36\xd5\x90\xa2\xef\x2b\xbb\xc5\xb4\x90\xbb\x1c\x7a\xec\x1b\x22\x35\xaf\x6e\xae\x69\xd9\x3c\x7f\xb7\x8b\x18\x7d\x4c\x56\x75\x90\x7f\xbe\x5b\x7e\xf8\xbc\xfc\xe3\xf7\xdb\xf7\x02\x78\x35\x0a\xe3\x59\xf3\xc1\xef\x73\xc9\xd6\x84\x76\x5f\xf8\xa7\x78\x9e\x50\x45\x53\x8a\xbf\x90\x9a\xdc\x3d\xe1\x05\xa3\x38\x05\x8b\x28\x22\x45\x01\x96\xfd\x77\x13\x9a\x6f\x6a\x39\x73\x5c\x14\xdf\x18\x8f\x8d\xa6\x75\xdf\xcd\x7b\xcd\x0c\xae\x37\x5f\x08\xa7\xa4\x24\xad\xd0\xc6\x96\xf0\x42\x4a\xb1\xc2\x14\x1c\x5a\x6b\x7f\xbc\xf4\x2f\x3b\x87\x7a\xd9\x3f\xe5\xf6\x92\xb1\xb4\x3b\xa3\x07\x0a\x38\xce\xda\x0e\x2d\x69\xa1\xeb\xf3\x17\xd2\x4e\xc7\x98\x6c\xc5\x23\xb1\x18\xe6\xf3\x6f\x15\xec\xfe\x62\xff\x7f\x00\x00\x00\xff\xff\x1b\x86\x14\xf5\x75\x1d\x00\x00")

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

	info := bindataFileInfo{name: "cloud.json", size: 7541, mode: os.FileMode(420), modTime: time.Unix(1453795200, 0)}
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
