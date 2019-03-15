package dokube

import (
	"fmt"
	"net/url"
	"testing"
)

func TestUrl(t *testing.T) {
	link := "https://936d7ecb-65df-4ae4-90b2-38f375f6ca83.k8s.ondigitalocean.com"
	u, e := url.Parse(link)
	fmt.Println(e, u.Host)
}
