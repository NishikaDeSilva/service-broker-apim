package config

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/wso2/service-broker-apim/internal/constants"
	"os"
	"strings"
)

// Struct to hold Auth configuration
type AuthConf struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type TLSConf struct {
	Enabled bool `yaml:"enabled"`
	Key string `yaml:"key"`
	Cert string `yaml:"cert"`
}
// Logging configuration
type LogConf struct {
	LogFile  string `yaml:"logFile"`
	LogLevel string `yaml:"logLevel"`
}

// Struct to hold configuration for HTTP
type HttpConf struct {
	Auth AuthConf `yaml:"auth"`
	TLS TLSConf `yaml:"tls"`
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}
// Main configuration structure
type BrokerConfig struct {
	Log LogConf `yaml:log"`
	Http HttpConf `yaml:"http"`
}

// Load configuration into BrokerConfig object
// Returns a pointer to the created BrokerConfig object
func LoadConfig() (*BrokerConfig, error) {
	viper.SetConfigType(constants.ConfigFileType)
	viper.SetEnvPrefix(constants.EnvPrefix)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := loadConfFile(); err != nil {
		return nil, err
	}

	var brokerConfig = defaultConf()
	err := viper.Unmarshal(brokerConfig)
	if err != nil {
		return nil, errors.New(fmt.Sprintf(constants.ErrMsgUnableToParseConf, err))
	}
	return brokerConfig, nil
}

// Load the configuration file
// Must set the "BROKER_APIM_CONF_FILE" env to the configuration file
func loadConfFile() error {
	confFile, exists := os.LookupEnv(constants.ConfFileEnv)
	if exists {
		fmt.Println(fmt.Sprintf(constants.InfoMsgSettingUp, confFile))

		viper.SetConfigFile(confFile)
		if err := viper.ReadInConfig(); err != nil {
			return errors.New(fmt.Sprintf(constants.ErrMsgUnableToReadConf, err))
		}
		return nil
	}
	return errors.New(fmt.Sprintf(constants.ErrMsgNoConfFile, constants.ConfFileEnv))
}

// Returns a BrokerConfig object with default values
func defaultConf() *BrokerConfig {
	return &BrokerConfig{
		Log: LogConf{
			LogFile: "server.log",
			LogLevel: "info",
		},
	}
}
