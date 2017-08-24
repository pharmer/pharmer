package main

import (
	"bytes"
	"fmt"
	"github.com/appscode/pharmer/api"
	"github.com/ghodss/yaml"
	"github.com/graymeta/stow"
	lol "github.com/graymeta/stow/local"
	"os"
)

func main() {
	loc, err := stow.Dial(lol.Kind, stow.ConfigMap{
		lol.ConfigKeyPath: "/tmp",
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	_, err = loc.Container("pharmer/abcd")
	if err != nil {
		c, err := loc.CreateContainer("pharmer/abcd")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		cluster := api.Cluster{
			Spec: api.ClusterSpec{
				KubernetesVersion: "1.2.3",
			},
		}
		bi, err := yaml.Marshal(cluster)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(string(bi))

		item, err := c.Put("pharmer/abcd/data.yaml", bytes.NewBuffer(bi), int64(len(bi)), nil)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(item.ID())
	} else {
		fmt.Println("Container already exists!")
	}
}
