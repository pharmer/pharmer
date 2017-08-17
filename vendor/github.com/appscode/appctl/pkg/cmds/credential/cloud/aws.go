package cloud

import (
	"io/ioutil"
	"strings"

	api "github.com/appscode/api/credential/v1beta1"
	"github.com/appscode/appctl/pkg/config"
	"github.com/appscode/appctl/pkg/util"
	term "github.com/appscode/go-term"
	ini "github.com/vaughan0/go-ini"
)

func CreateAWSCredential(req *api.CredentialCreateRequest) {
	apiReq = req

	bytes, err := ioutil.ReadFile(util.Home() + "/.aws/credentials")
	term.ExitOnError(err)

	data := string(bytes)
	if !strings.HasPrefix(data, "[default]") {
		data = "[default]\n" + data
	}

	dataReader := strings.NewReader(data)
	configs, err := ini.Load(dataReader)
	term.ExitOnError(err)

	req.Data = make(map[string]string)
	for key, value := range configs["default"] {
		req.Data[strings.ToLower(key)] = value
	}
	c := config.ClientOrDie()
	_, err = c.CloudCredential().Create(c.Context(), apiReq)
	util.PrintStatus(err)
}
