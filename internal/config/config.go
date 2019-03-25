package config

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/wso2/service-broker-apim/internal/constants"
	"os"
	"strings"
)

type Log struct {
	LogFile  string `yaml:"logFile"`
	LogLevel string `yaml:"logLevel"`
}
type BrokerConfig struct {
	LogConf Log `yaml:logConf"`
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
		fmt.Printf(constants.ErrMsgSettingUp, confFile)
		viper.SetConfigFile(confFile)
		if err := viper.ReadInConfig(); err != nil {
			return errors.New(fmt.Sprintf(constants.ErrMsgUnableToReadConf, err))
		}
	}
	return nil
}

// Returns a BrokerConfig object with default values
func defaultConf() *BrokerConfig {
	return &BrokerConfig{
		LogConf: Log{
			LogFile: "server.log",
		},
	}
}
