// This is the main pkg
package main

import (
	"errors"
	"fmt"
	"github.com/pivotal-cf/brokerapi"
	"github.com/wso2/service-broker-apim/internal/config"
	"github.com/wso2/service-broker-apim/internal/utils"
)

func main() {

	// Initialize the configuration
	brokerConfig, err := config.LoadConfig()
	if err != nil {
		utils.HandleErrorAndExit(err)
	}
	//Initialize logs
	logger, err := utils.InitLogger(brokerConfig.LogConf.LogFile, brokerConfig.LogConf.LogLevel )
	if err != nil {
		utils.HandleErrorAndExit(err)
	}
	logger.Error("ssd", errors.New("Sdasdadad"))
	var s = brokerapi.AsyncBindResponse{}
	fmt.Println(s)
	fmt.Println(brokerConfig.LogConf.LogFile)

}
