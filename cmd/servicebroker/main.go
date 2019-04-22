// This is the main pkg
package main

import (
	"crypto/tls"
	"fmt"
	"github.com/pivotal-cf/brokerapi"
	"github.com/wso2/service-broker-apim/pkg/broker"
	"github.com/wso2/service-broker-apim/pkg/client"
	"github.com/wso2/service-broker-apim/pkg/config"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"github.com/wso2/service-broker-apim/pkg/dbutil"
	"github.com/wso2/service-broker-apim/pkg/utils"
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
	// Initialize the logs
	logger, err := utils.InitLogger(brokerConfig.Log.LogFile, brokerConfig.Log.LogLevel)
	if err != nil {
		utils.HandleErrorAndExit(err)
	}
	// Initialize HTTP client
	client.SetupClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: brokerConfig.APIM.InsecureCon},
		},
	})

	// Initialize ORM
	dbutil.InitDB(&brokerConfig.DB)
	// Create tables
	dbutil.CreateTables()

	// Initialize Token Manager
	tManager := &broker.TokenManager{
		TokenEndpoint:         brokerConfig.APIM.TokenEndpoint,
		DynamicClientEndpoint: brokerConfig.APIM.DynamicClientEndpoint,
		UserName:              brokerConfig.APIM.Username,
		Password:              brokerConfig.APIM.Password,
		InSecureCon:           brokerConfig.APIM.InsecureCon,
	}
	tManager.InitTokenManager(broker.ScopeAPICreate, broker.ScopeSubscribe, broker.ScopeAPIPublish)

	// Initialize APIM Manager
	apimManager := &broker.APIMManager{
		PublisherEndpoint: brokerConfig.APIM.PublisherEndpoint,
		StoreEndpoint:     brokerConfig.APIM.StoreEndpoint,
		InsecureCon:       brokerConfig.APIM.InsecureCon,
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
		Username: brokerConfig.HTTP.Auth.Username,
		Password: brokerConfig.HTTP.Auth.Password,
	}
	apimServiceBroker := &broker.APIMServiceBroker{
		BrokerConfig: brokerConfig,
		TokenManager: tManager,
		APIMManager:  apimManager,
	}
	brokerAPI := brokerapi.New(apimServiceBroker, logger, brokerCreds)
	// Register router with handlers
	http.Handle("/", brokerAPI)

	host := brokerConfig.HTTP.Host
	port := brokerConfig.HTTP.Port
	logger.Info(fmt.Sprintf(constants.InfoMSGServerStart, host, port))
	if !brokerConfig.HTTP.TLS.Enabled {
		if err := http.ListenAndServe(host+":"+port, nil); err != nil {
			utils.HandleErrorWithLoggerAndExit(
				fmt.Sprintf(constants.ErrMSGUnableToStartServer, host, port), err)
		}
	} else {
		logger.Debug(constants.DebugMSGHttpsEnabled)
		if err := http.ListenAndServeTLS(host+":"+port,
			brokerConfig.HTTP.TLS.Cert, brokerConfig.HTTP.TLS.Key, nil); err != nil {
			utils.HandleErrorWithLoggerAndExit(fmt.Sprintf(constants.ErrMSGUnableToStartServerTLS,
				host,
				port,
				brokerConfig.HTTP.TLS.Key,
				brokerConfig.HTTP.TLS.Cert),
				err)
		}
	}
}
