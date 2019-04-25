/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */
// constants package holds the all the constants
package constants

// ConfFileEnv is used as a key to get the configuration file location
const ConfFileEnv = "BROKER_APIM_CONF_FILE"

// ConfigFileType constant is used to specify the Configuration file type(YAML)
const ConfigFileType = "yaml"

// EnvPrefix is the prefix for configuration parameters ex: BROKER_APIM_LOGCONF_LOGFILE
const EnvPrefix = "BROKER_APIM"

// Error messages
const ErrMsgUnableToReadConf = "unable to read configuration: %s"
const ErrMsgUnableToParseConf = "unable to parse configuration"
const ErrMsgInvalidLogLevel = "invalid log level: %s"
const ErrMsgUnableToOpenLogFile = "unable to open the Log file: %s"
const ErrMsgNoConfFile = "couldn't find the Configuration file env: %s"
const ErrMSGUnableToStartServerTLS = "unable to start the server on Host: %s port: %s TLS key: %s TLS cert: %s"
const ErrMSGUnableToStartServer = "unable to start the server on Host: %s port: %s"

// Info messages
const InfoMsgSettingUp = "loading the configuration file: %s "
const InfoMSGShutdownBroker = "starting APIM Service Broker shutdown"
const InfoMSGServerStart = "starting APIM broker"

// DebugMSGHttpsEnabled when HTTPS is enabled
const DebugMSGHttpsEnabled = "HTTPS is enabled"

// LoggerName is used to specify the source of the logger
const LoggerName = "wso2-apim-broker"

// FilePerm is the permission for the server log file
const FilePerm = 0644

// Test messages
const ErrMsgTestCouldNotSetEnv = "couldn't set the ENV: %v"
const ErrMsgTestIncorrectResult = "expected value: %v but then returned value: %v"

// ExitCode represents the OS exist code 1
const ExitCode1 = 1
