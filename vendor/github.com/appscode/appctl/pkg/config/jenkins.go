package config

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/cookiejar"

	"github.com/appscode/api/dtypes"
	"github.com/appscode/appctl/pkg/util"
	"github.com/appscode/client/cli"
	"github.com/appscode/errors"
	term "github.com/appscode/go-term"
	"github.com/bndr/gojenkins"
	"golang.org/x/net/context"
)

func Jenkins() (*gojenkins.Jenkins, error) {
	rc, err := cli.LoadApprc()
	if err != nil {
		return nil, err
	}
	a := rc.GetAuth()
	if a == nil {
		return nil, errors.New("Command requires authentication, please run `appctl login`")
	}
	c, err := Client()
	if err != nil {
		return nil, err
	}
	resp, err := c.CI().Metadata().ServerInfo(context.Background(), &dtypes.VoidRequest{})
	util.PrintStatus(err)
	if resp.Provider != "jenkins" {
		term.Fatalln("Looks like you are using ci services provided by " + resp.Provider + ". appctl ci commands only work with Jenkins. Sorry!")
	}

	tr := *(http.DefaultTransport.(*http.Transport))
	if resp.CaCert != "" {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: false,
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM([]byte(resp.CaCert))
		tlsCfg.RootCAs = pool
		tr.TLSClientConfig = tlsCfg
	}
	cookies, _ := cookiejar.New(nil)
	hc := &http.Client{
		Transport: &tr,
		Jar:       cookies,
	}
	jenkins := gojenkins.CreateJenkins(hc, resp.ServerUrl, a.UserName, "Bearer."+a.Token)
	jenkins.Requester.SslVerify = true
	_, err = jenkins.Init()
	if err != nil {
		return nil, err
	}
	return jenkins, nil
}

func JenkinsOrDie() *gojenkins.Jenkins {
	c, err := Jenkins()
	term.ExitOnError(err)
	return c
}
