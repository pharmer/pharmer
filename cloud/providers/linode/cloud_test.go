package linode

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/linode/linodego"
	"golang.org/x/oauth2"
)

func TestDeleteSS(t *testing.T) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "API_TOKEN"})

	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	c := linodego.NewClient(oauth2Client)
	scripts, err := c.ListStackscripts(context.Background(), nil)
	fmt.Println(err)
	for _, script := range scripts {
		if script.Username == "tahsin" {
			if err := c.DeleteStackscript(context.Background(), script.ID); err != nil {
				fmt.Println(err)
			}
		}
	}
}
