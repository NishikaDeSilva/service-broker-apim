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

// APIMConf represents the information required to interact with the APIM
type APIMConf struct {
	Username              string `mapstructure:"username"`
	Password              string `mapstructure:"password"`
	InsecureCon           bool   `mapstructure:"insecureCon"`
	TokenEndpoint         string `mapstructure:"tokenEndpoint"`
	DynamicClientEndpoint string `mapstructure:"dynamicClientEndpoint"`
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
	LogFile  string `mapstructure:"logFile"`
	LogLevel string `mapstructure:"logLevel"`
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
}

// LoadConfig load configuration into BrokerConfig object
// Returns a pointer to the created BrokerConfig object
func LoadConfig() (*BrokerConfig, error) {
	viper.SetConfigType(constants.ConfigFileType)
	viper.SetEnvPrefix(constants.EnvPrefix)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := loadConfigFile(); err != nil {
		return nil, err
	}

	var brokerConfig = defaultConf()
	err := viper.Unmarshal(brokerConfig)
	if err != nil {
		return nil, errors.Wrapf(err, constants.ErrMsgUnableToParseConf)
	}
	return brokerConfig, nil
}

// loadConfigFile load the configuration file
// Must set the "BROKER_APIM_CONF_FILE" env to the configuration file
func loadConfigFile() error {
	confFile, exists := os.LookupEnv(constants.ConfFileEnv)
	if exists {
		fmt.Println(fmt.Sprintf(constants.InfoMsgSettingUp, confFile))
		viper.SetConfigFile(confFile)
		if err := viper.ReadInConfig(); err != nil {
			return errors.Wrapf(err, constants.ErrMsgUnableToReadConf, err)
		}
		return nil
	}
	return errors.New(fmt.Sprintf(constants.ErrMsgNoConfFile, constants.ConfFileEnv))
}

// defaultConf returns a BrokerConfig object with default values
func defaultConf() *BrokerConfig {
	return &BrokerConfig{
		Log: LogConf{
			LogFile:  "server.log",
			LogLevel: "info",
		},
		HTTP: HTTPConf{
			Auth: AuthConf{
				Username: "admin",
				Password: "admin",
			},
			TLS: TLSConf{
				Enabled: false,
				Key:     "",
				Cert:    "",
			},
			Host: "0.0.0.0",
			Port: "8443",
		},
		APIM: APIMConf{
			Username:              "admin",
			Password:              "admin",
			TokenEndpoint:         "https://localhost:8243/token",
			DynamicClientEndpoint: "https://localhost:9443/client-registration/v0.14/register",
		},
	}
}
