package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"
	yaml "gopkg.in/yaml.v2"
	"github.com/gin-gonic/gin"
)

type ConfigEtcdCerts struct {
	CaCert 	   string `yaml:"ca_cert"`
	ClientCert string `yaml:"client_cert"`
	ClientKey  string `yaml:"client_key"`
}

type ConfigEtcdClient struct {
	Endpoints         string
	ConnectionTimeout time.Duration	`yaml:"connection_timeout"`
	RequestTimeout    time.Duration `yaml:"request_timeout"`
	Retries           uint64
	Auth              ConfigEtcdCerts
}

type ConfigLock struct {
	Timeout       time.Duration
	RetryInterval time.Duration `yaml:"retry_interval"`
}

type ConfigServer struct {
	Port      int64
	Address   string
	BasicAuth string `yaml:"basic_auth"`
}

type Config struct {
	EtcdClient     ConfigEtcdClient `yaml:"etcd_client"`
	Lock    	   ConfigLock
	Server         ConfigServer
}

func getConfigFilePath() string {
	path := os.Getenv("ETCD_BACKEND_CONFIG_FILE")
	if path == "" {
	  return "config.yml"
	}
	return path
}

func getConfig() (Config, error) {
	var c Config

	b, err := ioutil.ReadFile(getConfigFilePath())
	if err != nil {
		return c, errors.New(fmt.Sprintf("Error reading the configuration file: %s", err.Error()))
	}
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return c, errors.New(fmt.Sprintf("Error parsing the configuration file: %s", err.Error()))
	}

	if c.EtcdClient.Endpoints == "" {
		return c, errors.New("No etcd endpoint specified in the configuration")
	}

	if int64(c.EtcdClient.ConnectionTimeout) == 0 {
		c.EtcdClient.ConnectionTimeout = 2 * time.Minute
	}

	if int64(c.EtcdClient.RequestTimeout) == 0 {
		c.EtcdClient.RequestTimeout = 2 * time.Minute
	}

	if int64(c.Lock.Timeout) == 0 {
		c.Lock.Timeout = 30 * time.Second
	}

	if int64(c.Lock.RetryInterval) == 0 {
		c.Lock.RetryInterval = 500 * time.Millisecond
	}

	if c.Server.Port == 0 {
		c.Server.Port = 14443
	}

	if c.Server.Address == "" {
		c.Server.Address = "0.0.0.0"
	}

	return c, nil
}

func getAccounts(conf Config) (gin.Accounts, error) {
	var accounts gin.Accounts
	if conf.Server.BasicAuth == "" {
		return accounts, nil
	}

	b, err := ioutil.ReadFile(conf.Server.BasicAuth)
	if err != nil {
		return accounts, errors.New(fmt.Sprintf("Error reading the basic auth file: %s", err.Error()))
	}
	err = yaml.Unmarshal(b, &accounts)
	if err != nil {
		return accounts, errors.New(fmt.Sprintf("Error parsing the basic auth file: %s", err.Error()))
	}

	return accounts, nil
}