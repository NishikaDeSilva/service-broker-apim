/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */

// config package responsible for loading, parsing the configuration
package config

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"os"
	"strings"
)

// DBConfig represent the ORM configuration
type DBConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	LogMode  bool
}

// APIMConf represents the information required to interact with the APIM
type APIMConf struct {
	Username              string `mapstructure:"username"`
	Password              string `mapstructure:"password"`
	InsecureCon           bool   `mapstructure:"insecureCon"`
	TokenEndpoint         string `mapstructure:"tokenEndpoint"`
	DynamicClientEndpoint string `mapstructure:"dynamicClientEndpoint"`
	PublisherEndpoint     string `mapstructure:"publisherEndpoint"`
	StoreEndpoint         string `mapstructure:"storeEndpoint"`
}

// AuthConf represents the username and the password for basic auth
type AuthConf struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// TLSConf represents configuration needed for HTTPS
type TLSConf struct {
	Enabled bool   `mapstructure:"enabled"`
	Key     string `mapstructure:"key"`
	Cert    string `mapstructure:"cert"`
}

// LogConf represents the configuration related to logging
type LogConf struct {
	LogFilePath string `mapstructure:"logFilePath"`
	Level       string `mapstructure:"level"`
}

// HTTPConf represents configuration needed for HTTP server
type HTTPConf struct {
	Auth AuthConf `mapstructure:"auth"`
	TLS  TLSConf  `mapstructure:"tls"`
	Host string   `mapstructure:"host"`
	Port string   `mapstructure:"port"`
}

// BrokerConfig main struct which holds references to sub configurations
type BrokerConfig struct {
	Log  LogConf  `mapstructure:log"`
	HTTP HTTPConf `mapstructure:"http"`
	APIM APIMConf `mapstructure:"apim"`
	DB   DBConfig `mapstructure:"db"`
}

// LoadConfig load configuration into BrokerConfig object
// Returns a pointer to the created BrokerConfig object
func LoadConfig() (*BrokerConfig, error) {
	viper.SetConfigType(constants.ConfigFileType)
	viper.SetEnvPrefix(constants.EnvPrefix)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	setDefaultConf()
	if err := loadConfigFile(); err != nil {
		return nil, err
	}

	var brokerConfig BrokerConfig
	viper.SetDefault("log.level", "info")
	err := viper.Unmarshal(&brokerConfig)
	if err != nil {
		return nil, errors.Wrapf(err, constants.ErrMsgUnableToParseConf)
	}
	return &brokerConfig, nil
}

// loadConfigFile loads the configuration file
// Must set the "BROKER_APIM_CONF_FILE" env to the configuration file
func loadConfigFile() error {
	confFile, exists := os.LookupEnv(constants.ConfFileEnv)
	if exists {
		fmt.Println(fmt.Sprintf(constants.InfoMsgSettingUp, confFile))
		viper.SetConfigFile(confFile)
		if err := viper.ReadInConfig(); err != nil {
			return errors.Wrapf(err, constants.ErrMsgUnableToReadConf, err)
		}
	}
	return nil
}

// setDefaultConf sets the default configurations
func setDefaultConf() {
	viper.SetDefault("log.logFilePath", "server.log")
	viper.SetDefault("log.level", "info")

	viper.SetDefault("http.auth.username", "admin")
	viper.SetDefault("http.auth.password", "admin")
	viper.SetDefault("http.tls.enabled", false)
	viper.SetDefault("http.tls.key", "key.pem")
	viper.SetDefault("http.tls.cert", "cert.pem")
	viper.SetDefault("http.host", "0.0.0.0")
	viper.SetDefault("http.port", "8444")

	viper.SetDefault("apim.username", "admin")
	viper.SetDefault("apim.password", "admin")
	viper.SetDefault("apim.insecureCon", true)
	viper.SetDefault("apim.tokenEndpoint", "https://localhost:8243")
	viper.SetDefault("apim.dynamicClientEndpoint", "https://localhost:9443")
	viper.SetDefault("apim.publisherEndpoint", "https://localhost:9443")
	viper.SetDefault("apim.storeEndpoint", "https://localhost:9443")

	viper.SetDefault("db.host", "localhost")
	viper.SetDefault("db.port", "3306")
	viper.SetDefault("db.username", "root")
	viper.SetDefault("db.password", "root123")
	viper.SetDefault("db.database", "broker")
	viper.SetDefault("db.logMode", false)
}
