/*
 * Copyright (c) 2019 WSO2 Inc. (http:www.wso2.org) All Rights Reserved.
 *
 * WSO2 Inc. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http:www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

// config package responsible for loading, parsing the configuration
package config

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"os"
	"strings"
)

const (
	// FilePathEnv is used as a key to get the configuration file location
	FilePathEnv = "BROKER_APIM_CONF_FILE"

	// FileType constant is used to specify the Configuration file type(YAML)
	FileType = "yaml"

	// EnvPrefix is the prefix for configuration parameters ex: BROKER_APIM_LOGCONF_LOGFILE
	EnvPrefix = "BROKER_APIM"

	InfoMsgSettingUp        = "loading the configuration file: %s "
	ErrMsgUnableToReadConf  = "unable to read configuration: %s"
	ErrMsgUnableToParseConf = "unable to parse configuration"
)

// DBConfig represent the ORM configuration
type DBConfig struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	Database   string `mapstructure:"database"`
	LogMode    bool   `mapstructure:"logMode"`
	MaxRetries int    `mapstructure:"maxRetries"`
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
// Returns a pointer to the created BrokerConfig object or any error encountered
func LoadConfig() (*BrokerConfig, error) {
	viper.SetConfigType(FileType)
	viper.SetEnvPrefix(EnvPrefix)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	setDefaultConf()
	if err := loadConfigFile(); err != nil {
		return nil, err
	}

	var brokerConfig BrokerConfig
	err := viper.Unmarshal(&brokerConfig)
	if err != nil {
		return nil, errors.Wrapf(err, ErrMsgUnableToParseConf)
	}
	return &brokerConfig, nil
}

// loadConfigFile loads the configuration into the Viper file only if the configuration file is pointed with "BROKER_APIM_CONF_FILE" environment variable
// Returns an error if it is unable to read the config into Viper
func loadConfigFile() error {
	confFile, exists := os.LookupEnv(FilePathEnv)
	if exists {
		fmt.Println(fmt.Sprintf(InfoMsgSettingUp, confFile))
		viper.SetConfigFile(confFile)
		if err := viper.ReadInConfig(); err != nil {
			return errors.Wrapf(err, ErrMsgUnableToReadConf, confFile)
		}
	}
	return nil
}

// setDefaultConf sets the default configurations for Viper
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
	viper.SetDefault("db.maxRetries", 3)
}
