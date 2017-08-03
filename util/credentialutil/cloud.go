package credentialutil

import (
	"encoding/csv"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/appscode/errors"
	"github.com/appscode/go/io"
	"github.com/appscode/log"
	ini "github.com/vaughan0/go-ini"
)

func LoadCloudCredential(path, provider string) (map[string]string, error) {
	data, err := io.ReadFile(path)
	if err != nil {
		return nil, errors.FromErr(err).Err()

	}
	m, err := ParseCloudCredential(data, provider)
	if err != nil {
		return m, errors.FromErr(err).Err()
	}
	return m, nil
}

func VerifyCloudCredential(path, provider string) (string, error) {
	cred, err := LoadCloudCredential(path, provider)
	if err != nil {
		return "", errors.FromErr(err).Err()
	}
	data, err := json.Marshal(cred)
	if err != nil {
		return string(data), errors.FromErr(err).Err()
	}
	return string(data), nil
}

func ParseCloudCredential(data, provider string) (map[string]string, error) {
	var cred map[string]string
	var err error
	provider = strings.ToLower(provider)
	if provider == "digitalocean" {
		cred, err = parseDOCredential(data)
	} else if provider == "aws" {
		cred, err = parseAwsCredential(data)
	} else if provider == "gce" {
		cred, err = parseGCloudCredential(data)
	}
	if IsCloudCredentialValid(cred, provider) {
		if err != nil {
			return cred, errors.FromErr(err).Err()
		} else {
			return cred, nil
		}
	}
	return nil, errors.New("Credential not valied").Err()
}

// Can Parse both ini or csv file. any one will be validated.
func parseAwsCredential(data string) (map[string]string, error) {
	iniRegex, err := regexp.Compile(".+,.+")
	if err != nil {
		log.Errorln("regex compiled failed")
		return nil, errors.FromErr(err).Err()
	}
	if iniRegex.MatchString(data) {
		m, err := parseAwsCSV(data)
		if err != nil {
			return m, errors.FromErr(err).Err()
		} else {
			return m, nil
		}
	}
	m, err := parseAwsINI(data)
	if err != nil {
		return m, errors.FromErr(err).Err()
	} else {
		return m, nil
	}
}

func parseAwsINI(data string) (map[string]string, error) {
	if !strings.HasPrefix(data, "[default]") {
		data = "[default]\n" + data
	}
	dataReader := strings.NewReader(data)
	configs, err := ini.Load(dataReader)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	awsCredential := make(map[string]string)

	for key, value := range configs["default"] {
		awsCredential[strings.ToLower(key)] = value
	}

	return awsCredential, nil
}

func parseAwsCSV(data string) (map[string]string, error) {
	csvReader := csv.NewReader(strings.NewReader(data))
	rawCsv, err := csvReader.ReadAll()
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	awsCredMap := map[string]string{
		parseAwsCsvKey(rawCsv[0][0]): rawCsv[1][0],
		parseAwsCsvKey(rawCsv[0][1]): rawCsv[1][1],
	}
	return awsCredMap, nil
}

func parseAwsCsvKey(key string) string {
	return "aws_" + strings.ToLower(strings.Replace(key, " ", "_", len(key)))
}

func parseGCloudCredential(data string) (map[string]string, error) {
	gCloudCredential := make(map[string]string)
	err := json.Unmarshal([]byte(data), &gCloudCredential)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	return gCloudCredential, nil
}

func parseDOCredential(data string) (map[string]string, error) {
	tokenMap := map[string]string{
		"token": data,
	}
	return tokenMap, nil
}

// This is not the propoer way to check validation. this just
// checks if some requires fields are missing or not.
func IsCloudCredentialValid(data map[string]string, provider string) bool {
	if len(data) <= 0 {
		return false
	}
	if provider == "aws" {
		if _, ok := data[AWSCredentialAccessKeyID]; !ok {
			return false
		}

		if _, ok := data[AWSCredentialSecretAccessKey]; !ok {
			return false
		}
	}

	if provider == "digitalocean" {
		if _, ok := data["token"]; !ok {
			return false
		}
	}

	if provider == "gce" {
		if _, ok := data[GCECredentialClientID]; !ok {
			return false
		}
	}
	return true
}
