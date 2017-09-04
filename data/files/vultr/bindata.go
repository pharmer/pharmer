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

var _cloudJson = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xcc\x59\x5d\x73\xda\x3c\x16\xbe\xcf\xaf\xf0\xf8\xda\xe5\x95\x3f\x30\x90\x3b\x07\xda\x9d\xec\x9b\xa4\x6c\xa0\xe9\xc5\x4e\xa7\xa3\xd8\x27\x44\x8b\x90\xbc\x92\xa0\x43\x3b\xf9\xef\x3b\x36\x60\xfc\x25\x63\x53\x9a\xd9\x9b\x7c\x58\xb2\x9e\x73\x9e\x73\xf4\x9c\x23\xf9\xd7\x95\x61\x98\x0c\xaf\xc0\xbc\x36\xcc\xcd\x9a\x2a\x61\x5a\xc9\x23\x60\x1b\xf3\xda\xf8\xf7\x95\x61\x18\x86\x19\xc1\x26\x7d\x6a\x18\xe6\x7f\xb1\x79\x65\x18\xdf\xd2\x39\x02\x16\x84\x33\x99\xcd\xfb\x95\xfe\x34\x0c\x93\xf2\x10\x2b\xc2\x59\xb2\xe6\x03\xfc\x30\xfe\x09\x42\xc2\x76\xbf\x44\xf6\x62\x32\x6a\x1f\x1f\xfe\xe4\x0c\x8e\x6b\xa5\x8f\x6c\x73\xff\xf7\xb7\xf4\xf7\x9b\xa5\x87\x19\xbf\x92\x10\x2f\x78\x1d\x86\xd3\x88\xe1\xb4\xc7\x98\x60\x4a\xb1\xac\x83\x70\x1b\x21\xdc\xf6\x10\x33\xc0\x4a\x51\xa8\xc3\xf0\x1a\x31\xbc\xf6\x18\x77\x5c\x1a\x01\x5b\x00\x85\x5a\x5f\xfa\x8d\x38\xfd\xf6\x38\x81\xa2\x98\x29\x5c\x87\xe1\x37\x62\xf8\x1d\x30\x56\x52\x81\x88\xf0\xaa\x0e\x65\xd0\x88\x32\xe8\xc2\x18\x8b\x38\xab\x83\x18\x36\x42\x0c\xdb\x43\x7c\x12\x98\x2d\x5f\xd6\x42\xd5\xa1\x8c\x1a\x51\x46\x1d\xd2\x8b\x50\x12\x72\x66\x3c\x61\x4a\x35\x1b\xb2\x79\xb7\xd8\x1d\xb6\xcb\x6c\x1b\x31\x0d\x48\xb3\x43\x76\x07\x8f\xa6\x58\x90\xda\x34\x76\x9a\xf7\x8b\xd3\x61\xc3\xcc\xf9\x72\x5b\xaf\x2c\xcd\x7b\xc5\xe9\xb0\x59\xee\x09\x5e\x91\x5a\x69\x69\xe6\xca\xed\x14\x7d\xb6\xc0\x31\x17\xf5\xf2\x82\x9a\xf5\x05\x95\x70\xb2\x2a\x40\x98\x54\x98\x85\x30\xdf\xc6\x50\x53\x0b\xe4\x72\x9d\xa6\x70\x4e\x22\x23\x90\xa1\x20\xf1\xc1\x2c\x1b\x39\x9e\x71\x7f\x63\x3c\x06\xf7\x96\x83\x8c\x7f\xdc\x18\xb3\xd9\xc4\x72\x7a\x08\x19\xf3\x1b\xe3\xe6\xeb\xf1\xc5\x10\x2b\x58\x70\xb1\x4d\x9d\x99\x4d\x72\x03\x71\x02\x62\x1f\xdd\xc2\xab\xc2\xff\x11\x91\x4b\xf3\xda\x70\x50\xce\x86\x58\x40\xb2\x5e\x64\x5e\x1b\x4a\xac\xa1\x96\xbd\x83\xf1\x9e\xd6\x78\x07\x79\xc3\x83\xf1\x5e\xff\x60\xbc\xdb\xdd\x78\xa7\x64\xbc\x53\x36\xde\xeb\x9f\x69\x7c\x5f\x6b\xbc\x87\x46\xfe\xc1\xf8\x51\xc6\xbc\xd7\xdd\x78\xaf\x64\xbc\x57\x36\x7e\x84\x4a\x29\x57\xa9\xf4\x56\xbe\x5e\x5a\xf9\xc2\x66\xe5\xab\x8f\x95\x2f\x13\x56\x5e\xcd\xad\xbc\xee\x5a\x79\x79\xb4\x0a\xd2\x62\x15\x44\xc0\x2a\x6c\x25\xab\x36\xe1\xcf\xe4\xdd\xd7\xf2\x3e\xb4\x47\xce\x81\x77\xbb\x9f\x11\xdf\xef\x4e\xbc\x5f\x22\x7e\x58\x26\xde\xee\xbf\x2b\xf3\xef\xc9\xef\x40\xaf\x28\xbe\x3b\xcc\x24\xc5\x45\x19\xc1\x7e\x77\x82\x87\x65\x4d\xf1\xcb\x0c\xbb\xe8\x6c\x86\xff\x0f\x49\x1d\x6a\x49\x75\x9d\x81\x9f\x49\x9d\x7f\x24\xd5\x46\x67\x28\x75\x39\x6f\xdd\x8a\xdc\xf9\x7f\x82\xd6\x26\x26\x7f\x9b\x3b\x1b\x21\x2d\x79\x7e\xbf\xef\x66\x52\x3b\xc8\x91\x77\xc6\x9e\x77\xca\x6a\xeb\x57\xe4\x76\xd0\x89\xbc\x06\x8a\x7e\x9f\x14\x5b\x5f\x7e\x0a\x32\x68\xb7\x4f\xa8\xc9\xc7\xc9\xed\x38\x98\x7f\x3c\x59\x43\xab\x6a\x68\x77\xe1\xe5\x54\xff\x96\xb9\xa8\x57\xfa\x82\x12\xe5\x7c\x74\xce\xf6\xb1\x1c\xfc\xaa\x20\xb5\x70\xb2\x9d\x5b\x7a\x81\x75\xbc\xfe\xc0\xaf\x71\xcb\x3d\xdb\xad\xb2\x20\x38\x95\x9c\xbe\x98\x5b\x2d\x25\x2e\xe7\x96\x77\xb6\x5b\xe5\xf2\x51\xd5\xb9\x0b\xb9\x35\xd4\x7b\x55\xe8\xaf\xf7\xdd\x46\x30\x0f\x4e\x77\xd8\xc1\x3c\xe8\xde\x62\x77\xea\x37\xfa\xda\x9a\xd7\x72\xef\x0d\x47\xad\x3a\xf3\x3e\x3a\xba\x7d\xb2\x37\x6f\xe1\x76\x25\x8a\xfd\x4e\x82\xfb\xdb\x6e\x8f\xf4\x95\xc6\x45\x83\x4c\x54\x07\xb9\x68\x9f\xec\xea\x6b\xdc\x2e\xeb\xa9\x5b\xa9\x33\xef\x1b\xed\x91\xdd\xea\x28\x63\xa3\x5c\xb8\x4f\x16\xd8\x16\x7e\x57\xb5\x08\xbd\x6f\xbc\x1d\xa4\xf7\xbc\xb8\xbd\xb3\x22\xfa\x27\x8e\xcf\xfd\x66\x1b\xf5\x47\xfc\x7c\x74\x7c\x74\xc1\x53\x72\x25\x32\x3e\x6a\xb6\xd1\x69\x77\x92\xbf\xe4\x35\x44\xf5\x24\x7f\xc2\x46\x7d\xc7\x54\x68\x27\x1c\x74\xc9\x93\x63\xb5\x8f\x70\xd0\x09\x3b\xf5\xb7\x22\x85\xce\x0e\x5d\xf2\x66\xa1\xda\xd2\x9d\xb2\x52\xdf\xc5\x14\x9a\x72\xef\xc2\x27\x9a\x6a\x53\xee\x9d\xb2\x54\xdf\x46\x16\x1a\x93\xcb\x1e\x68\xab\x1d\x89\x7b\xca\x4e\x7d\xab\x31\x1a\xba\x28\xcb\xcf\xe1\x85\x8f\x39\xa3\x4a\x86\x0e\x0f\x96\x66\x97\x91\xa1\x80\x08\x98\x22\x98\xd6\x5c\x45\x4e\x05\xdf\x90\x08\x44\x02\xf8\x94\x7d\xea\x3a\x2c\x18\x53\xbc\xfd\xc4\xc5\x0a\xab\x64\xfc\x85\x00\x8d\x8e\xe3\x98\x31\xae\xd2\x8b\xd4\x64\xdd\x5f\x47\x01\x8f\x5f\xb1\x58\x81\xe8\xe1\x38\x96\x21\x8f\xa0\x17\xf2\xd5\x5f\x21\x5d\x4b\x05\xe2\xc3\xd1\x9a\x64\xc9\xbc\xee\xd7\xbe\x16\x31\x59\x7e\x65\xff\xc6\x5b\x66\x48\x6a\x57\xb1\xd6\x1c\xad\xd9\x7d\xb7\x0b\x39\x7b\x21\x8b\xd4\xc9\x2f\x77\xf3\xc7\xef\xf3\xcf\x7f\x7f\x7c\xc8\x81\x27\xab\x70\xb1\xca\x3e\xf8\x7d\x57\x7c\x09\xac\x38\xe1\x3f\x72\x17\xd0\x9a\x21\x8a\x9f\x21\x35\x6e\x0a\x42\x72\x86\xa9\x11\x84\x21\x48\x69\xcc\xab\x73\x09\x8b\xd7\x29\x9d\x31\x96\xf2\x07\x17\x91\x99\x8d\xbe\x15\xeb\x5e\x16\xc1\xe5\xfa\x19\x04\x03\x05\xf2\x09\x84\xac\xff\xbe\xb8\xd9\x8d\xa4\x8a\xd8\x1b\xf4\xf4\x3a\x54\x1a\xdd\x7d\xd4\xcc\x45\x2f\x82\x4d\xfe\x0c\x9b\x27\x3a\x82\x17\xbc\xa6\x6a\x16\x43\x58\x7c\x67\x1f\xdc\xdb\xf8\x11\xb3\x05\xec\x2a\x70\xcf\xf1\xbc\x1e\xea\xa1\xbf\xec\xc2\x59\x5a\x82\xd8\x90\x10\xc6\x75\x6f\xa0\x9a\xf9\x98\xa6\xd7\xf5\xf0\xc0\x23\x18\x93\x48\xc8\xbd\x71\xb9\x29\xc0\xf0\x33\x3d\xac\x78\xcf\x19\x51\x5c\x10\x96\x46\xfb\x90\x4b\xa6\x6e\xfa\x1d\x5f\x2c\x76\x73\x6b\x17\x4d\x50\xb5\x53\xe8\x6e\x60\x02\x52\x11\x96\x7d\x51\x38\x40\x7e\x00\x8a\xa5\x22\xa1\x04\x2c\xc2\xd7\x82\x01\xf9\x81\xfd\xea\x8f\x10\x53\x12\x62\x99\xaf\x95\x09\xe3\x4c\xce\x40\x6c\x12\xa2\x72\x1c\xd9\xc8\x2c\xce\x99\xf0\x15\x26\x29\xfa\xb2\x97\xd0\x45\x0b\x0c\x46\x2b\x22\x93\xdc\x18\x73\xa6\x04\x4f\xf3\xf4\x01\xaf\x40\xc6\x38\x84\x3b\xf2\x02\xe1\x36\xa4\x60\xdd\x91\x15\x51\x69\x34\x84\x35\xdb\x05\x29\x08\x43\xbe\x66\xca\x4a\xb2\x9a\x48\x05\x4c\x3d\x71\xba\x5e\xc1\x5d\x92\xee\xd6\x64\x9f\x0d\x8a\x0b\xbc\x80\x31\xc5\x52\x5a\x8f\x20\xf9\x5a\x84\xf0\xaf\x35\xcf\x7d\xe5\x34\x0c\x73\x85\xeb\x12\xc4\x4f\x03\x5e\xbc\x73\x62\xa0\x7e\x70\xb1\xcc\x0b\x53\x92\xff\x1f\x5e\x28\x66\x0c\xa8\x36\x92\x01\x05\xa1\x74\x31\xe7\x09\xb1\x66\x04\xcf\x04\xb3\x3a\x30\x4e\x49\xb8\xcd\x43\x32\xce\x6a\x92\xe6\x2b\x3c\xbf\x72\xbe\x4c\x77\x75\xb0\x56\xaf\x4c\x97\x37\x8f\xcf\x38\x4c\x26\xfc\xac\x4e\x98\x7d\xfe\x34\xbf\xfb\x3c\xfe\xfb\xcb\xf4\xfb\x34\x78\xb8\x1d\xeb\x96\x08\xa6\xb7\x32\x0d\xfd\x0d\x96\x24\x0c\xd6\x11\x51\xda\xa9\x7b\x8f\x03\xa5\x48\x58\x9d\x14\x73\x4a\x67\x14\x20\xbe\x65\x0a\xc4\x26\x95\x51\xb7\x36\xd5\xa7\xeb\x67\x4a\xc2\xdb\x69\x49\x00\x8e\x72\x74\xf5\xf6\xbf\x00\x00\x00\xff\xff\x11\x19\xb1\x68\x1d\x21\x00\x00")

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

	info := bindataFileInfo{name: "cloud.json", size: 8477, mode: os.FileMode(420), modTime: time.Unix(1453795200, 0)}
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
