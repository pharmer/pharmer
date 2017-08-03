package util

import (
	homeDir "github.com/mitchellh/go-homedir"
)

const (
	appscodePath     = "/.appscode"
	apprcPath        = "/apprc.json"
	repoDbPath       = "/repo.db"
	appctlLogPath    = "/.log"
	appctlConfigPath = "/.status.cfg"
	kubeConfigPath   = "/.kube/config"
	dockerPath       = "/.docker"
	dockerConfigPath = "/config.json"
	settingsPath     = "/.m2"
	mavenConfigPath  = "/settings.xml"
	npmConfigPath    = "/.npmrc"
)

var (
	home, _ = homeDir.Dir()
)

func Home() string {
	return home
}

func AppscodePath() string {
	return Home() + appscodePath
}

func AppctlLogPath(namespace string) string {
	if namespace == "" {
		return Home() + appscodePath + appctlLogPath
	}
	return Home() + appscodePath + "/" + namespace + appctlLogPath
}

func ApprcPath() string {
	return Home() + appscodePath + apprcPath
}

func AppctlStatusConfigPath() string {
	return Home() + appscodePath + appctlConfigPath
}

func KubeConfigPath() string {
	return Home() + kubeConfigPath
}

func DockerConfigPath() string {
	return Home() + dockerPath + dockerConfigPath
}

func MavenConfigPath() string {
	return Home() + settingsPath + mavenConfigPath
}

func NpmConfigPath() string {
	return Home() + npmConfigPath
}
