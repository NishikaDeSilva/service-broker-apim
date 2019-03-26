package constants

// Configurations related constants
const ConfFileEnv = "BROKER_APIM_CONF_FILE"
const ConfigFileType = "yaml"
const EnvPrefix = "BROKER_APIM"

// Error messages
const InfoMsgSettingUp = "Setting up configuration file: %v "
const ErrMsgUnableToReadConf = "Unable to read configuration: %v"
const ErrMsgUnableToParseConf = "Unable to parse configuration: %v"
const ErrMsgInvalidLogLevel = "Invalid log level: %v"
const ErrMsgUnableToOpenLogFile = "Unable to open Log file: %v"
const ErrMsgNoConfFile = "Couldn't find the Configuration file env: %s"
const ErrMSGUnableToStartServerTLS = "Unable to start the server on Host: %s port: %s TLS key: %s TLS cert: %s"
const ErrMSGUnableToStartServer = "Unable to start the server on Host: %s port: %s"

// Info messages
const InfoMSGShutdownBroker = "Starting APIM Service Broker shutdown"
const InfoMSGServerStart = "Starting APIM broker on host: %s port: %s"

// Debug messages
const DebugMSGHttpsEnabled = "HTTPS is enabled"

// Logging
const LoggerName = "wso2-apim-broker"
const FilePerm = 0644

// Test messages
const ErrMsgTestCouldNotSetEnv = "Couldn't set the ENV: %v"
const ErrMsgTestIncorrectResult = "Expected value: %v but then returned value: %v"
