package config

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type WhoAmIResponse struct {
	ErrorCode interface{}      `json:"error_code"`
	ErrorInfo interface{}      `json:"error_info"`
	Result    *ConduitUserData `json:"result"`
}

type ConduitUserData struct {
	Image        string   `json:"image"`
	Phid         string   `json:"phid"`
	PrimaryEmail string   `json:"primaryEmail"`
	RealName     string   `json:"realName"`
	Roles        []string `json:"roles"`
	URI          string   `json:"uri"`
	UserName     string   `json:"userName"`
}

type ConduitClient struct {
	Url  string
	err  error
	body []byte

	Token string
}

func (p *ConduitClient) Call() *ConduitClient {
	client := http.Client{}
	form := url.Values{}
	form.Add("api.token", p.Token)

	phReq, err := http.NewRequest("POST", p.Url, strings.NewReader(form.Encode()))
	if err != nil {
		p.err = err
		return p
	}
	phReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	phResp, err := client.Do(phReq)
	if err != nil {
		p.err = err
		return p
	}
	message, err := ioutil.ReadAll(phResp.Body)
	if err != nil {
		p.err = err
		return p
	}
	p.body = message
	return p
}

func (p *ConduitClient) Into(i interface{}) error {
	if p.err != nil {
		return p.err
	}

	err := json.Unmarshal(p.body, i)
	return err
}
