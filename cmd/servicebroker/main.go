// This is the main pkg
package main

import (
	"fmt"
	"github.com/pivotal-cf/brokerapi"
	"github.com/wso2/service-broker-apim/internal/config"
	"github.com/wso2/service-broker-apim/internal/constants"
	"github.com/wso2/service-broker-apim/internal/utils"
	"github.com/wso2/service-broker-apim/pkg/broker"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	// Initialize the configuration
	brokerConfig, err := config.LoadConfig()
	if err != nil {
		utils.HandleErrorAndExit(err)
	}
	//Initialize the logs
	logger, err := utils.InitLogger(brokerConfig.Log.LogFile, brokerConfig.Log.LogLevel)
	if err != nil {
		utils.HandleErrorAndExit(err)
	}

	// Handling terminating signal
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGTERM)
	go func() {
		<-sigChannel
		logger.Info(constants.InfoMSGShutdownBroker)
		os.Exit(0)
	}()

	brokerCreds := brokerapi.BrokerCredentials{
		Username: brokerConfig.Http.Auth.Username,
		Password: brokerConfig.Http.Auth.Password,
	}
	apimServiceBroker := &broker.APIMServiceBroker{

	}
	brokerAPI := brokerapi.New(apimServiceBroker, logger, brokerCreds)
	http.Handle("/", brokerAPI)

	logger.Info(fmt.Sprintf(constants.InfoMSGServerStart, brokerConfig.Http.Host, brokerConfig.Http.Port))
	if !brokerConfig.Http.TLS.Enabled {
		if err := http.ListenAndServe(brokerConfig.Http.Host+":"+brokerConfig.Http.Port, nil); err != nil {
			utils.HandleErrorWithLoggerAndExit(logger,
				fmt.Sprintf(constants.ErrMSGUnableToStartServer, brokerConfig.Http.Host,
					brokerConfig.Http.Port), err)
		}
	} else {
		logger.Debug(constants.DebugMSGHttpsEnabled)
		if err := http.ListenAndServeTLS(brokerConfig.Http.Host+":"+brokerConfig.Http.Port,
			brokerConfig.Http.TLS.Cert, brokerConfig.Http.TLS.Key, nil); err != nil {
			utils.HandleErrorWithLoggerAndExit(logger, fmt.Sprintf(constants.ErrMSGUnableToStartServerTLS,
				brokerConfig.Http.Host,
				brokerConfig.Http.Port,
				brokerConfig.Http.TLS.Key,
				brokerConfig.Http.TLS.Cert),
				err)
		}
	}
}
