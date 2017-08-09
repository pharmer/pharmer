package config

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/appscode/appctl/pkg/util"
	"github.com/appscode/go/io"
)

type RemoteAnnouncement struct {
	Type    int    `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
	hash    string
}

func NewRemoteAnnouncement() (*RemoteAnnouncement, error) {
	r := &RemoteAnnouncement{}
	if err := r.Read(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *RemoteAnnouncement) Read() error {
	resp, err := http.Get(RemoteUrl)
	if err != nil {
		return err
	}
	fileContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(fileContent, r)
	if err != nil {
		return err
	}
	r.hash = digest(fileContent)
	return nil
}

func (r *RemoteAnnouncement) UpdateConfig() {
	// update configs.
	config := AnnouncementConfigs()
	if config.Announcement == nil {
		config.Announcement = new(AnnouncementReader)
	}
	config.Announcement.Update(r.hash)
	config.Update()
}

func (r *RemoteAnnouncement) Hash() string {
	return r.hash
}

func ensureConfigFile() {
	if !io.IsFileExists(util.AppctlStatusConfigPath()) {
		// update empty config
		emptyConfig := &AnnouncementConfig{}
		emptyConfig.Update()
	}
}

func digest(d []byte) string {
	hasher := sha256.New()
	hasher.Write(d)
	return base64.RawStdEncoding.EncodeToString(hasher.Sum(nil))
}
