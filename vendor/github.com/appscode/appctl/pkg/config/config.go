package config

import (
	"time"

	"github.com/appscode/api/version"
	"github.com/appscode/appctl/pkg/util"
	term "github.com/appscode/go-term"
	"github.com/appscode/go/io"
)

const (
	RemoteUrl = "https://storage.googleapis.com/appscode-cdn/status.json"
)

var Version version.Version

type AnnouncementConfig struct {
	Version      string              `json:"version,omitempty"`
	Announcement *AnnouncementReader `json:"announcement,omitempty"`
}

type AnnouncementReader struct {
	ReadTime time.Time `json:"last_read,omitempty"`
	Hash     string    `json:"hash,omitempty"`
}

func (a *AnnouncementReader) Update(hash string) {
	a.ReadTime = time.Now()
	a.Hash = hash
}

func AnnouncementConfigs() *AnnouncementConfig {
	config := &AnnouncementConfig{}
	ensureConfigFile()
	err := io.ReadFileAs(util.AppctlStatusConfigPath(), config)
	if err != nil {
		term.Fatalln("failed to read config file")
	}
	if config.Announcement == nil {
		config.Announcement = new(AnnouncementReader)
	}
	return config
}

func (c *AnnouncementConfig) Update() {
	err := io.WriteJson(util.AppctlStatusConfigPath(), c)
	if err != nil {
		term.Fatalln("failed to update config file")
	}
}

func (c *AnnouncementConfig) AnnouncementChanged() (bool, *RemoteAnnouncement) {
	as, err := NewRemoteAnnouncement()
	if err != nil {
		return false, nil
	}
	if c.Announcement.Hash != as.Hash() {
		return true, as
	}
	return false, nil
}

func (c *AnnouncementConfig) IsRemoteReadTime() bool {
	if time.Since(c.Announcement.ReadTime) > (time.Hour * 24) {
		return true
	}
	return false
}
