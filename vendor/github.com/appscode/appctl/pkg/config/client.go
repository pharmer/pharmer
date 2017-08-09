package config

import (
	"github.com/appscode/client"
	"github.com/appscode/client/cli"
)

func Client() (*client.Client, error) {
	return cli.Client("appctl/" + Version.Version)
}

func ClientOrDie() *client.Client {
	return cli.ClientOrDie("appctl/" + Version.Version)
}
