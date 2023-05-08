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

type EtcdPasswordAuth struct {
	Username string
	Password string
}

type ConfigEtcdAuth struct {
	CaCert 	   string   `yaml:"ca_cert"`
	ClientCert string   `yaml:"client_cert"`
	ClientKey  string   `yaml:"client_key"`
	PasswordAuth string `yaml:"password_auth"`
	Username     string `yaml:"-"`
	Password     string `yaml:"-"`
}

type ConfigEtcdClient struct {
	Endpoints         []string
	ConnectionTimeout time.Duration	`yaml:"connection_timeout"`
	RequestTimeout    time.Duration `yaml:"request_timeout"`
	RetryInterval     time.Duration `yaml:"retry_interval"`
	Retries           uint64
	Auth              ConfigEtcdAuth
}

type ConfigLock struct {
	Timeout       time.Duration
	RetryInterval time.Duration `yaml:"retry_interval"`
}

type ConfigServerTls struct {
	Certificate string
	Key         string
}

type ConfigServer struct {
	Port      int64
	Address   string
	BasicAuth string          `yaml:"basic_auth"`
	Tls       ConfigServerTls
	DebugMode bool            `yaml:"debug_mode"`
}

type ConfigLegacySupport struct {
	Read     bool
	Clear    bool
	AddSlash bool `yaml:"add_slash"`
}

type Config struct {
	EtcdClient         ConfigEtcdClient    `yaml:"etcd_client"`
	Lock    	       ConfigLock
	Server             ConfigServer
	LegacySupport      ConfigLegacySupport `yaml:"legacy_support"`
	RemoteTerminiation bool                `yaml:"remote_termination"`
}

func getConfigFilePath() string {
	path := os.Getenv("ETCD_BACKEND_CONFIG_FILE")
	if path == "" {
	  return "config.yml"
	}
	return path
}

func getPasswordAuth(path string) (EtcdPasswordAuth, error) {
	var a EtcdPasswordAuth

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return a, errors.New(fmt.Sprintf("Error reading the password auth file: %s", err.Error()))
	}

	err = yaml.Unmarshal(b, &a)
	if err != nil {
		return a, errors.New(fmt.Sprintf("Error parsing the password auth file: %s", err.Error()))
	}

	return a, nil
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

	if c.EtcdClient.Auth.PasswordAuth != "" {
		pAuth, pAuthErr := getPasswordAuth(c.EtcdClient.Auth.PasswordAuth)
		if pAuthErr != nil {
			return c, pAuthErr
		}
		c.EtcdClient.Auth.Username = pAuth.Username
		c.EtcdClient.Auth.Password = pAuth.Password
	}

	if len(c.EtcdClient.Endpoints) == 0 {
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