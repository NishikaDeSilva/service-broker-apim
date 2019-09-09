/*
 * Copyright (c) 2019 WSO2 Inc. (http:www.wso2.org) All Rights Reserved.
 *
 * WSO2 Inc. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http:www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

// Package utils holds a common set of Util functions
package utils

import (
	"code.cloudfoundry.org/lager"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"io"
	"net/url"
	"os"
	"path"
)

const (
	ErrMSGCannotParseURL = "unable to parse URL"
)

var logger = lager.NewLogger(constants.LoggerName)
var ioWriter io.Writer

type LogData struct {
	Data lager.Data
}

// GetEnv returns the value (which may be empty) If the Key is present in the environment
// Otherwise the default value is returned
func GetEnv(key, defaultVal string) string {
	val, exists := os.LookupEnv(key)
	if exists {
		return val
	}
	return defaultVal
}

// InitLogger initializes lager logging object
// 1. Setup log level
// 2. Setup log file
// Must initialize logger object to handle logging
func InitLogger(logFile, logLevelS string) (lager.Logger, error) {
	logL, err := lager.LogLevelFromString(logLevelS)
	if err != nil {
		return nil, errors.Wrapf(err, constants.ErrMsgInvalidLogLevel, logL)
	}
	f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, constants.FilePerm)
	if err != nil {
		return nil, errors.Wrapf(err, constants.ErrMsgUnableToOpenLogFile, logFile)
	}
	ioWriter = io.MultiWriter(os.Stdout, f)
	logger.RegisterSink(lager.NewWriterSink(ioWriter, logL))
	return logger, nil
}

// IoWriterLog returns the IO writer object for logging
func IoWriterLog() io.Writer {
	return ioWriter
}

// LogInfo logs Info level messages using configured lager.Logger
func LogInfo(msg string, data *LogData) {
	logger.Info(msg, data.Data)
}

// LogError logs Info level messages using configured lager.Logger.
func LogError(msg string, err error, data *LogData) {
	logger.Error(msg, err, data.Data)
}

// LogDebug logs Info level messages using configured lager.Logger
func LogDebug(msg string, data *LogData) {
	logger.Debug(msg, data.Data)
}

// HandleErrorAndExit prints an error and exit with exit code 1
// Only applicable upto server startup since process will be killed once invoked
func HandleErrorAndExit(err error) {
	fmt.Println(err)
	os.Exit(constants.ExitCode1)
}

// HandleErrorWithLoggerAndExit prints an error through the provided logger and exit with exit code 1
// Only applicable upto server startup since process will be killed once invoked
func HandleErrorWithLoggerAndExit(errMsg string, err error) {
	LogError(errMsg, err, &LogData{})
	os.Exit(constants.ExitCode1)
}

// IsValidParams returns false if one of the arguments are empty or argument array is nil
func IsValidParams(vals ...string) bool {
	if vals == nil {
		return false
	}
	for _, val := range vals {
		if val == "" {
			return false
		}
	}
	return true
}

// RawMSGToString converts json.RawMessage into String
// Returns the string representation of json.RawMessage and any error occurred
func RawMSGToString(msg *json.RawMessage) (string, error) {
	j, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(j), nil
}

// AddData adds data to current Log data obj
// Returns a reference to the current Log data obj
func (l *LogData) AddData(key string, val interface{}) *LogData {
	if l.Data == nil || len(l.Data) == 0 {
		l.Data = lager.Data{}
	}
	if key != "" && val != nil {
		l.Data[key] = val
	}
	return l
}

// ConstructURL construct URL by joining the paths provided
// first param will be treated as the base and the rest of params configured as paths
// An error will be thrown if the number of paths is equal to zero
// Returns constructed path and any error occurred
func ConstructURL(paths ...string) (string, error) {
	if len(paths) == 0 {
		return "", errors.New("no paths found")
	}
	u, err := url.Parse(paths[0])
	if err != nil {
		return "", errors.New(ErrMSGCannotParseURL)
	}
	for i := 1; i < len(paths); i++ {
		u.Path = path.Join(u.Path, paths[i])
	}
	return u.String(), nil
}
