// Code generated by go-bindata.
// sources:
// cloud.json
// credential.json
// DO NOT EDIT!

package softlayer

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

var _cloudJson = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x5c\xdd\x72\x1b\x37\xd2\xbd\xd7\x53\xa0\xe6\x4a\xa9\x1a\xb1\x38\x7f\xd0\xc8\x77\x8e\x9c\xcf\x71\xfc\x29\x9b\x5d\x65\xb3\x5b\xb5\xe5\x52\xc1\x1c\x88\x9c\x68\x08\x30\x00\x28\x9b\x49\xf9\xdd\xb7\x86\x92\xf9\x03\x0c\x9b\x3d\x54\x3b\x65\x65\x79\x93\x48\x28\x70\xfa\xf4\x41\xf7\x39\x6c\x88\xf4\x1f\x27\x8c\x45\x4a\x4c\x65\xf4\x82\x45\x56\xdf\xba\x46\x2c\xa4\x89\xe2\x76\x59\xaa\x7b\x1b\xbd\x60\xff\x39\x61\x8c\xb1\xa8\x92\xf7\xcb\x65\xc6\xa2\xdf\x44\x74\xc2\xd8\xbb\xe5\x26\x23\xc7\xb5\x56\xeb\x7d\x7f\x2c\xff\xcb\x58\xd4\xe8\x91\x70\xb5\x56\xed\x83\x5f\x4e\xad\x93\xa6\x12\xd3\x98\xfd\x28\xdd\x44\x9a\x46\xa8\xca\x3e\x3e\x6e\xf5\x90\x76\xa7\x98\x6e\x2c\xff\xae\x95\x5c\x3f\x79\xb9\x24\xa6\x76\x98\xac\x76\x3c\x2e\x64\xd1\xe3\xef\xef\x96\xff\xff\x14\xef\x46\x72\x39\x91\x4a\x89\x3a\x66\x6f\x54\x55\x8b\x2e\x04\xa3\x89\x04\x11\x8c\x26\x72\x98\xe0\x03\xbe\x12\x4d\x23\x6c\xcc\xfe\x79\xfd\xb2\x2b\x5a\x25\x1a\x30\x5a\x25\x9a\xed\x7c\xdb\x85\xc2\x5f\xe0\xfe\xc2\x85\xb7\x90\x0c\xf1\x88\xff\xcf\x08\x75\x77\x3b\x37\x2e\x66\xaf\xa5\x99\x0a\xb5\xe8\x02\x7e\x6b\x04\x08\xfc\xd6\x88\x61\x8a\x0f\xfa\xbd\x56\x63\xf6\x56\xab\x71\xcc\x2e\x27\xb5\xea\x3c\x99\xc9\xdd\x18\x0c\x39\xb9\x1b\xf7\x0b\x39\xb7\x4e\xab\x9d\x47\x33\xd1\x73\x38\x9c\x9e\xf7\x09\xf7\xff\x5a\x55\xcb\x68\x6f\xbb\x82\x35\x5a\x81\xc1\x1a\xad\xfa\x04\xbb\x92\xcd\x7b\x3d\x37\x4a\xc6\xec\xe5\xdc\x3a\x23\x9a\xee\x62\x9f\x4a\xb8\xfc\xa6\xb2\xe9\x53\xec\x7f\x9f\x4b\x23\x9d\x30\x3a\x66\x57\xf2\x63\x3d\xd2\xdd\x31\x3f\xee\x89\xf9\xb1\x4f\xcc\xab\xba\x11\x2a\x66\x6f\x9c\x68\x3a\x0b\x75\x5a\xef\x49\xb1\xee\x95\xe2\x95\x56\xce\x48\xd1\xc4\xec\x52\x28\x51\x75\xb3\xba\xe7\x30\xa7\x5a\xf5\x09\xf9\x37\xdb\xe8\x98\xfd\xa8\xcd\x07\xd1\x99\xa1\xb6\x70\x86\xda\xf6\xca\xf0\x27\x61\x6a\x1b\xb3\x56\x06\x46\xb2\x2b\xde\x4c\x18\x30\xde\x4c\x98\x3e\xf1\xae\x85\x62\x3f\x68\x2b\x77\x36\xa2\xfd\x75\x04\xc6\xb3\xbf\x8e\xb6\x35\xb2\x5d\xe8\xe1\x09\xd7\x42\xb3\x9f\xc4\xbc\x25\x79\x17\x02\xa1\x61\x04\x42\xf7\xca\x58\xea\x79\x13\xb3\xb7\xda\xc8\xce\xfa\xb1\x72\x4f\x38\xd9\x33\x9c\x70\xae\x01\xf8\x95\xb0\x94\x5b\x29\x7a\x85\xab\xd5\x58\xcc\xb4\x91\x31\x5b\xfd\xd8\x19\x56\xc1\x72\x6e\xd5\xb8\x57\xd8\x45\xa5\xe4\x62\x8f\xde\xd9\x45\x05\xc7\x5c\x54\x7d\x62\xfe\xac\xef\x16\x3a\x66\x3f\x88\x99\x50\x5d\xe1\x9c\xbe\x03\xc3\x39\x7d\xd7\x47\xd5\x7f\xd6\x46\x2b\xa7\x21\xe9\x71\x1a\xee\x4d\xa7\x7b\xf5\xe6\xbf\x84\x9d\xd4\x6a\xbc\xb4\xc9\x57\x97\x3b\x2b\xe8\x43\x05\x77\xe8\x87\xca\xeb\xd0\x76\x21\xf7\x60\xac\xde\x52\xd6\xca\xba\x56\x7b\x6e\xdc\x62\x26\x3b\xde\x59\xca\x8f\x4e\x1a\x25\x9a\x1b\x7b\x37\x6f\xa3\x27\xa3\x64\xba\x0e\x5f\x49\x3b\x32\xf5\xec\x73\x06\x09\x1b\x69\x23\x99\x50\x15\x4b\x5e\x7f\xcb\xfe\xf1\xf2\x6a\xbd\x75\x24\x9c\x1c\x6b\xb3\x78\x7c\x37\x60\x9a\x05\xfb\xa5\x36\x6e\x2e\x1a\x76\x2d\xcd\xbd\xdc\xa0\x72\x34\x6b\x43\x25\xeb\xdc\xc5\xb4\xfd\xbd\x93\xc0\x10\x5f\x8a\xc2\x97\x92\xe3\x4b\x91\xf8\x72\x14\xbe\x9c\x1c\x5f\x8e\xc4\xc7\x51\xf8\x38\x39\x3e\x8e\xc4\x57\xa2\xf0\x95\xe4\xf8\x4a\x24\xbe\x04\x57\x80\x09\x7d\x05\x26\xd8\x12\x4c\x70\x67\x9c\xd0\x1f\x72\x82\x3b\xe5\x14\x52\x99\x74\x89\xd0\x12\xc9\x4c\x7a\x90\xcc\xa4\x90\xcc\x6c\x02\x7c\xfa\x29\xfb\x00\x71\x87\x9c\x42\x3a\xb3\x09\xf0\xe9\x42\xe3\x03\xc4\x09\x4d\x0a\x09\xcd\x26\xc0\xa7\x17\xa1\x0f\x10\x5b\x83\x80\xd2\x6c\x02\x7c\xba\xd4\xf8\x00\x71\x52\x93\x82\x52\xb3\xd5\x25\xf4\x55\x88\xd4\x9a\x14\xd4\x9a\x2d\x8c\xf4\xe7\x8c\x14\x9b\x1c\x12\x9b\x9c\x54\x6c\xf2\x83\xc4\x26\x87\xc4\x26\x27\x15\x1b\x1f\x20\xee\x94\x73\x48\x6c\x72\x52\xb1\xf1\x01\xe2\xc4\x26\x87\xc4\x26\x27\x15\x1b\x1f\x20\xb6\x06\x01\xb1\xc9\x49\xc5\xc6\x07\x88\x13\x9b\x1c\x14\x9b\x9c\x56\x6c\x82\x36\xc1\x96\x21\x24\x36\x39\xad\xd8\x04\x18\xb1\x07\x9d\x21\x79\xcc\xe8\x79\xcc\xd0\xed\x8c\xac\xc6\x9c\xbe\x1c\x73\x6c\x3d\x72\xa4\xe6\x70\x7a\xd1\xe1\x38\xd5\x29\x21\x63\x29\x49\x8d\xa5\x3c\xc8\x58\x4a\xc8\x58\x4a\x52\x63\xf1\x01\xe2\x2a\xb1\x84\x8c\xa5\x24\x35\x16\x1f\x20\xf6\x88\x01\xc5\x29\x49\x8d\xc5\x07\x88\xd3\x9b\x12\x32\x96\x92\xd4\x58\x7c\x80\xb8\x46\x2e\x41\x63\x29\x69\x8d\x25\x68\x13\x6c\x19\x42\xc6\x52\xd2\x1a\x4b\x80\x11\x7b\xd0\x90\xb1\x94\xb4\xc6\xe2\x63\x44\x1a\x4b\x09\x1a\x4b\x49\x6b\x2c\x41\x43\x63\xeb\x11\x32\x96\x92\xd6\x58\x82\x9e\x46\x5e\xd2\x81\xf7\x23\x09\xed\x05\x49\x72\xd8\x0d\x49\x02\x5e\x91\x24\xb4\x77\x24\x01\x46\xe4\x4d\x18\x78\x4b\x92\xd0\x5e\x93\x04\x18\xd1\x67\x0d\xdd\xd6\xd1\xde\x94\x04\x18\x91\x97\xb2\xe0\x5d\x49\x42\x7b\x59\x12\x60\x44\x5e\xcc\xc2\xd7\x25\x09\xf1\x7d\x49\xd8\x35\xe8\x92\x04\x6f\x67\x89\xaf\x4c\x42\x98\xe8\x13\x87\xec\x66\x0b\x26\x81\xdf\x04\x30\x91\x86\xd3\x36\x38\xb6\x32\x09\x2c\x27\x6c\x71\x74\x6d\x42\xa6\xb3\xdd\xe4\x5f\x40\x89\xb0\xb6\xc3\x41\xdb\xe1\xb4\xb6\xc3\x0f\xb3\x1d\x0e\xda\x0e\xa7\xb5\x1d\x1f\x23\xb2\x2a\x39\x68\x3b\x9c\xd6\x76\x7c\x8c\xe8\xb3\x86\x64\x88\xd3\xda\x8e\x8f\x11\x29\x42\x1c\xb4\x1d\x4e\x6b\x3b\x3e\x46\x64\x6b\x73\xd8\x76\x38\xb1\xed\x04\x5d\x83\x2e\x49\xd0\x76\x38\xb1\xed\x04\x30\xd1\x27\x0e\xda\x0e\x27\xb6\x1d\x1f\x26\xd6\x76\x38\x6c\x3b\x9c\xd8\x76\x82\x16\x47\xd7\x26\x68\x3b\x9c\xd8\x76\x82\x2e\xc7\xdb\x4e\x8a\xa5\x33\x49\xbf\x00\x9f\x49\x8a\x23\x34\x03\xe7\xb2\x8c\x76\x2e\xcb\x0e\x9b\xcb\x32\x70\x2e\xcb\x68\xe7\xb2\x00\x23\xae\x7f\x32\x70\x2e\xcb\x68\xe7\xb2\x00\x23\xae\x2a\x33\x70\x2e\xcb\x68\xe7\xb2\x00\x23\x4e\x2e\x33\x70\x2e\xcb\x68\xe7\xb2\x00\x23\xbe\x67\xb0\x05\x49\x60\x90\x61\xd7\xa0\x4b\x12\x32\xc8\x8c\x78\x2e\x0b\x61\xa2\x4f\x1c\x32\xc8\x8c\x78\x2e\x0b\x60\x22\x0d\x32\x83\xe7\xb2\x8c\x78\x2e\x0b\x5b\x1c\x5d\x9b\x90\x41\x66\xc4\x73\x59\xd8\xe5\x68\x29\x02\x0d\xd2\xeb\xa1\x2f\xc0\x67\x0f\x83\x4c\x73\xb4\xfd\xe4\x5f\xc2\x80\x72\x5c\x85\x16\xe0\xac\x5b\xd0\xce\xba\xc5\x61\xb3\x6e\x01\xce\xba\x05\xed\xac\x1b\x60\x44\xf3\x08\x74\x50\x41\x3b\xeb\x06\x18\x71\xfd\x53\x80\xb3\x6e\x41\x3b\xeb\x06\x18\x71\xc2\x5e\x80\xb3\x6e\x41\x3b\xeb\x06\x18\x71\xdd\x5d\xc0\xb3\x6e\x41\x3c\xeb\x86\x5d\x83\x6f\x6d\xec\x79\x13\x58\x79\x08\x13\x7d\xe2\x90\x95\x17\xc4\xb3\x6e\x00\x13\x69\xe5\x05\x3c\xeb\x16\xc4\xb3\x6e\xd8\xe2\xe8\xda\x84\xac\xbc\x20\x9e\x75\xc3\x2e\x47\x4b\x11\x68\xe5\x05\xf5\xac\xdb\xd1\x44\x68\x42\x41\x2b\xdf\xb6\x1f\x02\x2b\x0f\x0d\x08\x69\xe5\x6f\x94\x93\x0d\xfb\xb7\xd4\x8a\x7d\x97\x9d\x25\xe9\xf9\x90\xdd\x67\x3b\x71\x5f\xd7\x6a\xdc\x48\xd6\xf9\x22\x76\x9a\xb3\xcb\x36\xa9\x98\x65\x83\x62\xc8\x5e\x7f\xff\xfb\x37\x60\x46\xdf\x0a\x23\xd9\x95\x74\xd4\x9f\xd0\xda\x40\xc7\xbe\x2b\xce\x52\x9e\x82\x39\xbd\x6a\x59\xed\x7e\x0d\x3b\x4d\xd2\xcf\x39\xa5\x83\x9c\x22\xa7\x43\xff\xbc\xb0\x49\xf9\x67\x7c\xf9\x01\x39\xe5\xec\x34\xe1\xeb\x9c\x12\x92\x9c\x0e\xec\xe7\xbf\x76\xed\x1d\x70\x4c\x5f\xfd\x29\xb5\xf8\x8a\x5e\xdd\xb4\x7e\x09\x3b\x4d\x87\xeb\x94\x32\x8a\x94\xd2\xe1\x81\xfa\xec\x03\xbc\xe8\x9f\xd3\xc5\x63\x4e\xf9\x3a\x27\x4e\x92\x53\xf8\xf9\xde\xc3\x73\xea\x59\x7a\x17\x0f\xa5\x97\x96\xd4\x39\x85\x1f\x2d\xfb\xeb\x9d\xd3\x81\xed\xf4\x35\xa7\x94\x16\xb8\x77\xe3\xff\x0b\xa5\xf7\x35\x2b\xf9\x2a\xa7\xd5\x37\x97\xef\xe6\xef\xa5\x51\xd2\x2d\xbf\xb5\xfc\x90\x63\x54\xc9\x5b\x31\x6f\xdc\x8d\x95\x6e\x3e\x5b\x2d\x33\x16\x29\x5d\x49\x2b\xdd\xc6\x12\x63\xd1\x2f\x97\xc9\xf5\xea\x22\xe3\xe1\xd1\x2b\xd2\xa2\x7b\x69\x6c\xad\x95\xbd\x79\xbf\xb8\x91\xea\x7e\xf3\x61\x95\xbc\xdf\xfa\xca\xf5\xfa\x91\xeb\xd7\x2d\xff\x30\x35\xc8\x07\x9b\xff\x7c\x4c\xc7\x9f\xae\xda\x1d\xec\xf4\xbd\x74\xe2\x9b\xed\x8d\x62\x36\xb3\x5b\x68\x97\xab\xeb\x9c\xcf\xec\x03\x6f\x9d\x61\xbc\x9d\xa2\x71\xab\x7d\x67\x62\xe4\x6f\xb5\x4e\x18\x77\xb6\x45\x67\x34\x1c\xf0\x41\x7a\xe1\xef\x9c\x68\xeb\x6e\xc5\xc8\xd9\x87\xc7\x15\x03\x1e\x6d\x6c\xf8\xe4\x65\x3a\x33\xb2\x3d\xfa\x2a\x7a\xc1\x9c\x99\xcb\x93\x8e\x7d\x10\x73\x7c\x2f\x73\x9c\x82\x39\x8e\x64\x8e\x3f\x17\xe6\x8a\x41\xb2\x87\xb9\x62\x90\x3c\x9d\x39\x3f\xcc\x2e\xe6\x8a\x41\xf2\x7c\x98\x4b\xf7\x32\x97\x52\x30\x97\x22\x99\x4b\x9f\x0f\x73\xd9\x5e\xe6\x32\x0a\xe6\x32\x24\x73\xd9\xf3\x61\x2e\xdf\xcb\x5c\x4e\xc1\x5c\x8e\x64\x2e\x7f\x3e\xcc\xed\xf3\xd6\x82\xc2\x5b\xfd\x30\xbb\x99\x7b\x36\xde\x5a\xec\xf5\xd6\x82\xc2\x5b\xfd\x30\xbb\x99\x7b\x46\xde\x7a\xbe\x97\xb9\xf3\x27\x51\x76\x8e\xa4\xec\x9c\x96\xb2\xf3\x2d\xca\xfa\x91\x72\x3e\x18\xee\x21\x25\xd8\xd1\x8b\x14\xff\xd5\xbb\x48\x39\x1f\x0c\xff\x1c\x52\x1e\x7f\x7a\xb7\x1a\x51\x7e\x13\xc7\x79\xe0\x38\x0f\x7c\xa5\xcc\x1d\xe7\x81\xe3\x3c\xf0\xe7\x33\x77\x9c\x07\x8e\xf3\xc0\x71\x1e\x78\x2e\xcc\x1d\xe7\x81\xe3\x3c\x70\x9c\x07\x9e\x3a\x0f\x9c\x7c\xfe\xed\xd3\xc9\xa7\xff\x06\x00\x00\xff\xff\x5c\x3f\xe0\x40\xec\x5f\x00\x00")

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

	info := bindataFileInfo{name: "cloud.json", size: 24556, mode: os.FileMode(420), modTime: time.Unix(1453795200, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _credentialJson = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\x8f\xcd\x6a\x84\x30\x14\x46\xf7\x79\x8a\xcb\x5d\xb5\xe0\x13\xb8\xab\x3b\x29\x85\x42\xe9\xaa\x88\x44\xbc\x96\x8c\x31\x09\xf9\x19\x14\xf1\xdd\x87\xdc\x61\xfe\xd0\x4d\x48\x72\x0e\x1c\xbe\x55\x00\xa0\xf3\xf6\xac\x7a\xf2\x58\x02\x06\x3b\x44\x2d\x17\xf2\x58\x64\xd4\xab\xe0\xb4\x5c\x32\xa9\xab\x2f\xa8\x74\xa2\x49\xcd\xf0\xf6\x73\xd3\xde\x5f\xbc\x76\xb0\x7e\x92\x31\xeb\x83\x22\xdd\x5f\x21\x5f\x03\x96\xf0\x27\x00\x00\x56\x3e\x01\xf0\x14\xac\xc9\x6a\x0a\xe4\x8d\x9c\x88\x6d\x26\xfc\x3a\x24\x5a\x76\xa4\x33\xfa\xdd\x21\x65\x5c\xe2\x74\xa4\x39\x22\xff\x6e\xc5\x71\x51\x3a\xd5\x8e\xb4\xec\x83\x3b\x70\xef\x7d\x7c\xd7\xf0\xf9\x4c\x8e\x72\x02\xa0\xe1\xc9\x51\xfe\x3f\x06\xe3\x98\x3a\xf2\x86\x22\x85\xec\x35\x62\xbb\x04\x00\x00\xff\xff\x6e\x2d\xd4\x3a\x77\x01\x00\x00")

func credentialJsonBytes() ([]byte, error) {
	return bindataRead(
		_credentialJson,
		"credential.json",
	)
}

func credentialJson() (*asset, error) {
	bytes, err := credentialJsonBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "credential.json", size: 375, mode: os.FileMode(420), modTime: time.Unix(1453795200, 0)}
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
	"cloud.json":      cloudJson,
	"credential.json": credentialJson,
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
	"cloud.json":      &bintree{cloudJson, map[string]*bintree{}},
	"credential.json": &bintree{credentialJson, map[string]*bintree{}},
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
