package util

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/golang/glog"
	ini "github.com/vaughan0/go-ini"
)

const (
	AWSCredentialAccessKeyID     = "aws_access_key_id"
	AWSCredentialSecretAccessKey = "aws_secret_access_key"
	DigitalOceanCredentialToken  = "token"
	LinodeCredentialToken        = "token"
	VultrCredentialToken         = "token"
	GCECredentialClientID        = "client_id"
	AzureClientID                = "client_id"
	AzureClientSecret            = "client_secret"
	AzureTenantID                = "tenant_id"
	AzureSubscriptionID          = "subscription_id"
)

func ParseCloudCredential(data, provider string) (map[string]string, error) {
	var cred map[string]string
	var err error
	provider = strings.ToLower(provider)
	if provider == "aws" {
		cred, err = parseAwsCredential(data)
	} else {
		cred, err = parseJsonCredential(data)
	}
	if IsCloudCredentialValid(cred, provider) {
		return cred, err
	}
	return nil, errors.New("Credential not valid")
}

// Can Parse both ini or csv file. any one will be validated.
func parseAwsCredential(data string) (map[string]string, error) {
	iniRegex, err := regexp.Compile(".+,.+")
	if err != nil {
		glog.V(4).Infoln("regex compiled failed")
		return nil, err
	}
	if iniRegex.MatchString(data) {
		return parseAwsCSV(data)
	}
	return parseAwsINI(data)
}

func parseAwsINI(data string) (map[string]string, error) {
	if !strings.HasPrefix(data, "[default]") {
		data = "[default]\n" + data
	}
	dataReader := strings.NewReader(data)
	configs, err := ini.Load(dataReader)
	if err != nil {
		return nil, err
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
		return nil, err
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

func parseJsonCredential(data string) (map[string]string, error) {
	cred := make(map[string]string)
	err := json.Unmarshal([]byte(data), &cred)
	if err != nil {
		return nil, err
	}
	return cred, nil
}

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
	} else if provider == "gce" {
		if _, ok := data[GCECredentialClientID]; !ok {
			return false
		}
	} else if provider == "digitalocean" {
		if _, ok := data[DigitalOceanCredentialToken]; !ok {
			return false
		}
	} else if provider == "linode" {
		if _, ok := data[LinodeCredentialToken]; !ok {
			return false
		}
	} else if provider == "vultr" {
		if _, ok := data[VultrCredentialToken]; !ok {
			return false
		}
	} else if provider == "azure" {
		if _, ok := data[AzureClientSecret]; !ok {
			return false
		}
		if _, ok := data[AzureClientID]; !ok {
			return false
		}
		if _, ok := data[AzureTenantID]; !ok {
			return false
		}
		if _, ok := data[AzureSubscriptionID]; !ok {
			return false
		}
	}
	return true
}
