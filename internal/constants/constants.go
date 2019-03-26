package constants

// ConfFileEnv is used as a key to get the configuration file location
const ConfFileEnv = "BROKER_APIM_CONF_FILE"

// ConfigFileType constant is used to specify the Configuration file type(YAML)
const ConfigFileType = "yaml"

// EnvPrefix is the prefix for configuration parameters ex: BROKER_APIM_LOGCONF_LOGFILE
const EnvPrefix = "BROKER_APIM"

// Error messages
const InfoMsgSettingUp = "Loading the configuration file: %s "
const ErrMsgUnableToReadConf = "Unable to read configuration: %s"
const ErrMsgUnableToParseConf = "Unable to parse configuration: %s"
const ErrMsgInvalidLogLevel = "Invalid log level: %s"
const ErrMsgUnableToOpenLogFile = "Unable to open Log file: %s"
const ErrMsgNoConfFile = "Couldn't find the Configuration file env: %s"
const ErrMSGUnableToStartServerTLS = "Unable to start the server on Host: %s port: %s TLS key: %s TLS cert: %s"
const ErrMSGUnableToStartServer = "Unable to start the server on Host: %s port: %s"

// Info messages
const InfoMSGShutdownBroker = "Starting APIM Service Broker shutdown"
const InfoMSGServerStart = "Starting APIM broker on host: %s port: %s"

// DebugMSGHttpsEnabled when HTTPS is enabled
const DebugMSGHttpsEnabled = "HTTPS is enabled"

// LoggerName is used to specify the source of the logger
const LoggerName = "wso2-apim-broker"

// FilePerm is the permission for the server log file
const FilePerm = 0644

// Test messages
const ErrMsgTestCouldNotSetEnv = "Couldn't set the ENV: %v"
const ErrMsgTestIncorrectResult = "Expected value: %v but then returned value: %v"
