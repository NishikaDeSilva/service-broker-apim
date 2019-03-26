package config

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/wso2/service-broker-apim/internal/constants"
	"os"
	"strings"
)

// AuthConf holds the username and the password for basic auth
type AuthConf struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// TLSConf holds configuration needed for HTTPS
type TLSConf struct {
	Enabled bool   `yaml:"enabled"`
	Key     string `yaml:"key"`
	Cert    string `yaml:"cert"`
}

// LogConf holds the configuration related to logging
type LogConf struct {
	LogFile  string `yaml:"logFile"`
	LogLevel string `yaml:"logLevel"`
}

// HTTPConf holds configuration needed for HTTP server
type HTTPConf struct {
	Auth AuthConf `yaml:"auth"`
	TLS  TLSConf  `yaml:"tls"`
	Host string   `yaml:"host"`
	Port string   `yaml:"port"`
}

// BrokerConfig main struct which holds references to sub configurations
type BrokerConfig struct {
	Log  LogConf  `yaml:log"`
	HTTP HTTPConf `yaml:"http"`
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
		return nil, errors.New(fmt.Sprintf(constants.ErrMsgUnableToParseConf, err))
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
	}
}
