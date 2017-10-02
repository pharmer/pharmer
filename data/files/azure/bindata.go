// Code generated by go-bindata.
// sources:
// cloud.json
// DO NOT EDIT!

package azure

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

var _cloudJson = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xc4\x9b\x6f\x73\xe2\x38\xf2\xc7\x9f\xcf\xab\x50\xf1\x38\x61\xb1\xf9\x13\x76\x9e\x31\x10\xf8\xf1\x9b\x5c\x26\x1b\x27\x33\x75\x77\xb5\x95\x12\x76\x87\x68\x31\x92\x4f\x92\xc9\x66\xb6\xe6\xbd\x5f\xc9\x26\xe0\xd8\x96\xdd\x76\x98\xdb\x27\xbb\x53\xb8\x51\x7f\xa4\xee\x6f\x77\xe3\xd8\x7f\x7d\x20\xa4\xc3\xe9\x16\x3a\x1f\x49\x87\x7e\x8f\x25\x74\xce\xcc\x47\xc0\x77\x9d\x8f\xe4\xdf\x1f\x08\x21\xa4\x13\xc0\x2e\xf9\x94\x90\xce\x7f\x68\xe7\x03\x21\xbf\x27\x36\x12\xd6\x4c\x70\x75\xb0\xfb\x2b\xf9\x2f\x21\x9d\x50\xf8\x54\x33\xc1\xcd\x9a\x5f\x99\x5c\x33\xce\xe8\x7e\x81\xc3\xd7\xcc\xb5\x4b\xaa\x34\xb9\xf7\x8e\x97\xbe\x0b\x0e\xc7\xf5\x92\x8f\x80\x2a\x1d\xab\xce\xfe\x83\xdf\x93\xff\xff\x38\x7b\x97\x3f\xe2\x22\x3c\xba\x78\x97\x4b\xf1\x5c\xea\x6e\x0a\x5c\x4b\x1a\xd6\xed\xd0\x4f\xcd\x9a\x6c\x72\x19\x86\x8c\x0b\xa6\xca\xbc\x5e\x0b\xa9\x9f\x08\xd2\x37\x37\xc6\x2d\x00\xee\xe0\x4f\x5a\xea\xdd\x13\x31\xde\xbb\x32\xc6\x2d\xbc\x7f\x03\xa5\x4b\x7d\x1c\x39\xac\x26\x25\x18\xcf\xa0\x74\x0b\x8a\x29\x0d\xd9\xa3\x90\x96\x5c\x4b\x00\x10\x8e\x1b\xef\xfb\x6d\xfe\x16\x1c\xd6\x24\x77\xea\xb2\x41\x72\xff\x16\xc3\x0a\x7c\x32\x65\xfa\xa5\x34\xc7\x29\xa7\x01\x25\x46\x59\xd5\x49\x9e\xd8\x19\x69\x35\x48\x32\x21\x05\xd7\xa2\xc2\xed\x3e\xc0\x08\xcf\xfb\xf8\xe2\x9d\x7b\x54\x90\x1b\x1a\x87\x82\x78\x9a\x6a\x28\x83\xf8\x24\xe9\x77\x16\x92\x24\xe5\x2b\x11\x56\x89\x61\x92\xed\x0d\x34\x2e\x21\xa4\x3c\xb0\x4b\xfc\x32\x96\x22\x82\x7a\x79\x43\x6a\x87\x76\x7c\x0d\xfa\x09\xa4\xf1\x5d\xaa\xf0\x24\xcf\x10\xbe\x4d\xa6\x35\x75\x3d\xa5\x32\x60\x8f\x8f\x65\x6e\xef\x3f\x13\xe3\xb9\xd2\x65\xbc\x31\x4e\xf1\xee\xae\x04\x0f\x04\xb7\x78\xab\x0f\x6b\xbc\x69\x18\x52\x8f\xf1\x35\x8d\x84\x2c\xcd\xa6\xc4\x9f\x11\x08\x99\xa8\x6c\x49\xb1\xd5\x4d\x63\x4a\x8d\x25\xda\xfd\xff\x09\xbe\x26\x9f\x05\x5f\x5b\x7b\x63\xad\xe7\xe6\x4e\xaf\xe1\x39\x3d\x4a\xf2\x8d\x86\x50\x9a\x51\x93\x58\x19\x6d\x32\x44\x19\xa1\xaf\xa6\xcd\x2a\xc9\x57\xe6\x6b\x21\xcb\x0b\xf5\xd1\xfb\x21\x02\x38\x84\x43\x14\x9a\x54\xb4\xcd\x8b\x38\x23\x1e\x65\x9a\x6e\x4b\x69\xfe\x9f\x46\x94\xd7\x9f\xc3\x1f\xc6\xac\x99\xef\x2f\x8a\x6e\x2a\x5c\xd6\x8a\x2b\x71\x59\xa2\xaf\xc3\x38\xc8\xb8\xd2\x94\xfb\x70\xf7\x12\x41\xc9\x50\xa8\x36\x71\x92\xe6\x9a\xf2\x80\xca\xe0\x61\xd2\x3b\xba\x0b\x40\xf9\x92\x45\x07\x9d\x94\xd9\xf8\x54\xc3\x5a\xc8\x97\x24\x66\xe7\x0a\x24\xcb\x66\x93\x1f\x99\xd5\x9d\xe3\xf6\xe8\xb6\xf3\x91\xf4\xba\x17\xa3\xf1\xd1\x0b\x53\x9b\xce\x47\xe2\xf6\x4a\xcf\xaa\xc0\xe7\x20\xf8\x9c\xf7\xf1\x39\xdd\x8b\x61\x1e\xef\x02\x89\xe7\x22\xf0\xdc\x66\x78\x6e\x0e\xaf\xdf\x2d\xd0\x39\xfd\x21\x0e\xaf\x8f\xc0\xeb\x37\xc3\x1b\xe4\xf0\x2e\x0a\x91\x1d\x23\xe1\x06\x08\xb8\x41\x33\xb8\x71\x3e\xb4\x83\x3c\xdd\xa8\x87\xa4\x1b\x22\xe8\x86\xef\x8b\x6c\x91\x0e\x1d\xd8\x11\x82\x6e\xf4\xbe\xc0\xba\x45\xcd\x62\x23\x7b\x81\xa0\xbb\x78\x5f\x64\x87\xa3\xd6\x91\x1d\x23\xe8\xc6\xd5\x74\xe4\x9c\xf8\x62\x1b\xc5\x1a\xce\x19\xd7\xc0\x15\xdb\x01\x79\x2d\xbd\x2d\xd8\xfb\x63\x17\xc7\xfe\x2b\x82\xfd\xd7\x53\xb2\x3b\xa3\x7c\xd2\x3a\x6e\x6b\x7a\x07\xd3\x6d\x9c\x9a\x76\xf3\xb7\x9d\xbd\x83\xea\x45\x35\xcd\xe8\xef\x3b\xfd\x19\x02\x7f\x66\xa1\x9f\x21\x5b\x69\x49\xaf\x1a\xe2\x3a\xe9\x0c\xd1\x49\x67\x96\x4e\x6a\xa3\xcb\xd7\xdb\x42\xab\x72\x7a\x48\x38\x44\x1f\x9d\x59\xfa\xa8\x0d\x2e\x5f\x6e\x8b\xcd\xc0\xc5\xd2\x21\x1a\xe9\xcc\xd2\x48\x6d\x74\x79\xd9\x14\x9b\xc1\x00\x4b\x87\x91\xcd\xcc\x26\x1b\x6c\x68\x4b\x5a\x29\x9a\x0f\x93\x79\x4e\xc3\xd4\x43\x34\x53\x34\x1f\x26\xf9\x9c\x86\xd9\x57\x5f\x16\xf1\xf1\xc5\xa4\x9f\xd3\x30\xff\x30\x85\x6f\x8c\x26\x7c\xd8\xa1\x62\xfc\xc6\xec\x0d\xe5\xce\xfd\x1f\x14\x40\x1c\xa5\xdb\x82\xf2\x84\x85\x10\x07\xd9\x6f\x01\x79\xca\x82\x88\xa3\x1c\xb4\xa0\x3c\x61\x61\x1c\xe2\x28\x87\x6d\xd2\x32\x2f\x9f\xa2\xbe\xf1\xea\xc1\xca\xa7\x8d\x7e\x4e\x5a\xc7\x91\x9c\x6d\x14\x74\xd2\x7a\x8e\xe4\x6c\x23\xa2\x93\xd6\x75\x24\x67\x1b\x19\x9d\xb6\xbe\x23\x95\xe4\xb4\x91\x92\xdb\x2b\xa4\x68\xaf\x24\x47\x91\xa8\x1e\x66\x18\xf2\x6c\xc3\x90\xd7\xbe\x0b\x5d\x20\xf9\x30\xe7\xe8\xd9\x0e\xd1\xc6\x57\xdf\x7f\x06\x48\x3c\xcc\x2c\xe4\xd9\x66\x21\x1b\x1e\xa2\xf3\x8c\x91\x7c\x98\x59\xc8\xb3\xcd\x42\x36\xbe\xfa\x9e\x33\x1c\x61\xd3\x0f\x97\x7f\x4d\x13\xb0\xbe\x8c\xa3\x4f\x10\x35\x8e\x7b\xd6\x79\x1c\x1b\xe3\xf7\x9c\x21\x2a\x09\xad\x13\x39\x36\xca\xc5\xd2\xed\x38\xc8\x5f\xfa\x1e\x6a\x26\xf7\xac\x43\xb9\xb5\xce\x20\xaa\xb6\xeb\x62\xb5\x8c\x9c\x2b\xbc\x8a\xb9\xc2\x7b\xcf\x60\x8e\x2e\x89\x48\xce\x8a\xb9\xc2\xce\x79\xc2\xd2\x88\xc4\xac\x18\x2b\xec\x98\xa7\x2c\x91\x48\xce\x8a\xb1\xc2\xce\x79\xc2\x52\x89\x1c\x2a\xbc\x8a\xa1\xa2\x22\x3d\xeb\x07\xf4\x06\x6a\x47\x2b\xa9\x95\x94\x4e\x5a\xdc\xb1\xa4\xad\xc4\x74\xd2\x22\x8f\x25\x6d\xa5\xa7\xd3\x16\x7b\x2c\x6a\x2b\x49\x9d\xb8\xe8\x63\x65\x55\x35\xac\x57\xe4\x2a\x62\x5a\x77\xc7\xb8\x61\x7d\x8e\x98\x95\xe6\x96\x49\x69\x8e\x6c\x4b\x85\x93\x74\x70\x09\x3a\x47\x9c\xe1\xdc\x72\x7c\x36\xb6\xbc\xce\x0b\x32\xef\xe3\x32\x72\x8e\x18\x3e\xe6\x96\xd1\xc3\xc6\x96\x57\x76\x41\xd8\x23\x5c\x02\xce\x11\x7f\xfc\x9b\x5b\xfe\xf8\x67\x63\x2b\xfc\xd1\xb9\xa8\x65\x64\x81\x9c\x3b\x88\xbf\xeb\xbe\x31\x42\xe5\x5c\x5e\xc0\xfd\xa2\x7e\x91\x75\x71\xee\x28\x0c\xa0\xb2\x00\xaa\xb6\xaa\x40\x06\xd7\xc5\xd0\xb9\x4d\xe9\x6a\x75\x81\x0c\xee\x00\x43\x37\x68\x4a\x57\xab\x0c\x6c\x45\x19\x63\xf0\xc6\x4d\xf1\xea\xc5\x81\xad\x2a\xce\x08\x95\x7b\xa3\xc6\xc9\x57\x2f\x0f\x64\x71\x59\x20\x1a\xc6\xc2\xd2\x30\x16\xc8\xe4\x2b\x8e\x34\xfd\x31\x92\x0e\xd1\x32\x16\x96\x96\x61\xa3\xcb\x27\x5f\x71\x8c\xb9\x18\xe1\xd4\xb1\x40\xfc\xa8\x5e\x58\x7e\x52\xdb\xe8\x0a\xb9\x57\x1c\x5c\x9c\x61\x1f\x27\x8f\x05\xa2\xa9\x2d\x2c\x4d\xcd\xc6\x57\xc8\x3c\xd7\x2d\xf6\xdc\xde\x05\x4e\x1f\x0b\xc4\x03\x4b\x0b\xcb\x03\x4b\x36\xc0\x7e\xa1\xf4\x0d\x8a\x8d\xd7\x19\x20\xf3\x0f\x73\xef\x73\x61\xbb\xf7\xb9\xc0\xde\x7a\x6a\x3d\xf3\x2f\x30\xf7\x3e\x17\xb6\x7b\x9f\x56\xbe\x7a\x89\x0c\x91\xd3\xc1\x02\x73\xf7\x73\x61\xbb\xfb\x69\x05\xc4\xa8\xa4\x87\x3e\x43\x8c\x4c\x6c\x37\x40\xad\x88\x18\xa1\xb8\xd8\x1f\x4c\x0b\x0f\xa3\x14\xcf\x26\x15\x1b\x23\x46\x2b\x83\xc3\xef\xa4\xc3\xf3\xba\xbe\x84\x00\xb8\x66\x34\x2c\x79\x5a\x37\x92\x62\xc7\x02\x90\xc6\xf1\xe4\xf0\x5a\xd8\xeb\x8a\x51\x48\x5f\xe6\x42\x6e\xa9\x36\xd7\x1f\x19\x84\x99\xf7\x14\x28\xe7\x42\x27\x8f\x1d\x9b\x75\x5f\x57\x34\x6b\x3e\x51\xb9\x05\xd9\xa5\x51\xa4\x7c\x11\x40\xd7\x17\xdb\x5f\xfc\x30\x56\x1a\xe4\xf9\x91\xc6\x2c\x79\x58\xcd\xf6\xb5\x80\xab\xfc\x57\xf6\xdf\xf8\x71\x00\x49\xb8\xde\x3e\xc5\x7c\xa4\x49\xdf\x71\xf3\x05\x7f\x64\xeb\x64\x93\xff\xba\xbf\xbd\x7c\xb8\xbb\xbc\x9e\x5c\xdf\x3d\x2c\x67\x19\x00\xb3\x92\x90\xe6\x5c\xd3\x17\xe4\x1e\x34\x70\xca\xf5\x03\x0b\xde\x1a\xfd\xa1\xd2\x48\xa6\x97\xf3\x4b\x84\x74\x05\x09\xe7\x5d\x72\x99\x2c\x73\xdf\x66\x3c\x8a\x75\xfa\xf5\x3f\x0f\x8f\x58\x67\x76\x53\xcf\xee\xdd\x7f\xf2\xa6\xb7\xcb\x9b\xbb\xe5\x97\xeb\x9a\x1d\xa8\x78\x75\x48\x3e\xeb\x3e\xb2\x46\xd6\xdd\x78\x19\xa3\x9f\xb0\xa7\xe9\xd5\xf2\xb2\x36\x1e\x7e\xc8\xa0\x22\x1e\xe9\x65\xeb\x0e\xa6\xc9\xe5\x9f\xc7\xee\x5d\x4e\x6f\x2f\xef\x10\xfc\x0a\x7c\x09\xba\x6a\x0f\x5e\x89\x45\x7e\x1f\x65\x36\x87\xbd\x44\x54\xa9\x67\x21\x83\xcc\x7e\xf6\xff\x2a\x7f\x7f\xa0\x50\x05\x3c\x2d\x24\x5d\xff\xb4\x62\xa0\xd2\xe5\x7f\x86\xb2\xbd\xbb\x2f\xb7\x93\xc5\xe5\xc3\x64\x3a\xfd\x72\x7f\x5d\x19\x8f\x3d\xc5\x03\xf5\x7d\x11\x73\x4b\x44\x4a\x2f\x1e\x82\x91\x1c\x16\xd9\x9f\x16\x99\x94\xd9\x9e\x40\xf0\xfb\x2d\x7d\xbe\xfc\x27\x66\x3b\x1b\x78\x29\xdf\x4a\xe1\x42\xf5\x36\xc8\xe7\xbc\x3d\x3e\xbf\x0e\xcd\x67\x13\xaf\x40\x72\xd0\xa0\xbe\x82\x54\xe5\xaf\x11\xef\xd2\x2b\x66\x61\xa7\x3b\xee\xda\x9f\xe0\xcd\x5d\x4d\xdf\x5d\xce\xe4\x5a\x00\xe6\x03\x2d\x63\x28\x64\x52\x00\x8f\x34\x0e\xb5\x17\x81\x9f\xcb\x4f\x11\x78\xf1\x8a\x43\xb2\x2f\xa7\xd7\x75\x07\x83\x6e\xaf\xdb\xfb\x25\x73\x0b\xc4\x74\x78\x90\x3b\xe6\xc3\x1b\xcb\x5e\x89\x1d\x0d\x93\x57\x71\xe0\x5a\x04\x30\x65\x81\x54\x7b\x9e\x8c\x09\x70\xba\x0a\x61\x9a\xb6\xc3\x7f\x08\xce\xb4\x90\x8c\x27\xe1\x7e\xd5\x48\xc7\x66\x7e\x25\xd6\xeb\xd4\xb6\x74\x51\xe3\xd5\x6a\x12\xa6\x17\x66\xa0\x34\xe3\x87\xb7\x85\x5e\x5d\x9e\x43\x48\x95\x66\xbe\x02\x2a\xfd\xa7\x37\x00\xd9\x0b\xfb\xd5\x6f\x21\x0a\x99\x4f\x55\xf6\xc6\x8a\x39\x64\xae\x3c\x90\x3b\x90\xcb\x9b\xcc\x19\x65\x9e\xb6\x4e\x6d\x66\x62\x4b\xd9\xbe\xdc\x25\xdb\xea\x9a\x43\x0b\xdf\x9c\x63\xb0\x65\xca\x24\xc5\x54\x70\x2d\x45\x92\xa4\xd7\x74\x0b\x2a\xa2\x3e\x5c\xb1\x47\xf0\x5f\xfc\x10\xce\xae\xd8\x96\xe9\x5b\xca\xd7\x20\xcf\xbc\x34\x44\xfb\xe4\x3d\xbb\x31\x49\xa5\x34\x70\xfd\x55\x84\xf1\x16\xae\x4c\xae\x9f\xcd\xf6\x69\x90\x66\xfa\x34\xa4\x4a\x9d\xdd\x82\x12\xb1\xf4\xe1\xb7\x58\x68\x9a\x85\xd8\x52\x03\x97\xcb\x8e\x51\x12\x75\x77\x90\x35\xe4\xa0\x9f\x85\xdc\xdc\x64\xaa\xe8\x63\x48\x39\x87\xd0\x1a\xc9\x49\x08\x52\xdb\x62\x2e\xcc\xc1\x76\x02\x58\x31\xca\xcb\xfc\x88\x90\xf9\x2f\x59\x6f\x5c\xf0\x92\xa4\xf9\x06\xab\x27\x21\x36\x77\x62\x03\x7c\x12\xeb\x27\x6e\xcb\x9b\xdb\x15\xf5\x8d\xc1\x77\x9b\xc1\xe4\x66\xa9\x92\xc0\x7e\xa2\x8a\xf9\x93\x38\x60\xda\x6a\xba\xdf\xcf\x44\x6b\xe6\x17\x8d\x22\x11\x86\x5e\x08\x10\x2d\xb9\x06\xb9\x4b\x0a\x7f\xbf\x34\x91\x6f\xe2\x55\xc8\xfc\x24\x93\xb2\x8a\x3e\xd6\x97\x0f\x3f\xfe\x1b\x00\x00\xff\xff\xb9\x41\x63\x97\xd5\x40\x00\x00")

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

	info := bindataFileInfo{name: "cloud.json", size: 16597, mode: os.FileMode(420), modTime: time.Unix(1453795200, 0)}
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
