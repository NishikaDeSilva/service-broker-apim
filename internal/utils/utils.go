package utils

import (
	"code.cloudfoundry.org/lager"
	"fmt"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/internal/constants"
	"io"
	"os"
)

// Initialize lager logging object
// 1. Setup log level
// 2. Setup log file
func InitLogger(logFile, logLevelS string) (lager.Logger, error) {
	logLevel, err := lager.LogLevelFromString(logLevelS)
	if err != nil {
		return nil, errors.New(fmt.Sprintf(constants.ErrMsgInvalidLogLevel, logLevel))
	}
	f, err := os.Create(logFile)
	if err != nil {
		return nil, errors.New(fmt.Sprintf(constants.ErrMsgUnableToOpenLogFile, logFile))
	}
	bl := lager.NewLogger(constants.LoggerName)
	bl.RegisterSink(lager.NewWriterSink(io.MultiWriter(os.Stdout, os.Stderr, f), logLevel))
	return bl, nil
}


func HandleErrorAndExit(err error) {
	fmt.Println(err)
	os.Exit(1)
}