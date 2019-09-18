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

// log package handles logging
package log

import (
	"code.cloudfoundry.org/lager"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
)

const (
	// LoggerName is used to specify the source of the logger
	LoggerName = "wso2-apim-broker"

	// FilePerm is the permission for the server log file
	FilePerm = 0644

	// ExitCode represents the OS exist code 1
	ExitCode1 = 1

	ErrMsgUnableToOpenLogFile = "unable to open the Log file: %s"
)

var logger = lager.NewLogger(LoggerName)
var ioWriter io.Writer

type Data struct {
	lData lager.Data
}

// InitLogger initializes lager logging object
// 1. Setup log level
// 2. Setup log file
// Must initialize logger object to handle logging
func InitLogger(logFile, logLevelS string) (lager.Logger, error) {
	logL, err := lager.LogLevelFromString(logLevelS)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, FilePerm)
	if err != nil {
		return nil, errors.Wrapf(err, ErrMsgUnableToOpenLogFile, logFile)
	}
	ioWriter = io.MultiWriter(os.Stdout, f)
	logger.RegisterSink(lager.NewWriterSink(ioWriter, logL))
	return logger, nil
}

// IoWriterLog returns the IO writer object for logging
func IoWriterLog() io.Writer {
	return ioWriter
}

// Info logs Info level messages using configured lager.Logger
func Info(msg string, data *Data) {
	if data == nil {
		data = &Data{}
	}
	logger.Info(msg, data.lData)
}

// Error logs Info level messages using configured lager.Logger.
func Error(msg string, err error, data *Data) {
	if data == nil {
		data = &Data{}
	}
	logger.Error(msg, err, data.lData)
}

// Debug logs Info level messages using configured lager.Logger
func Debug(msg string, data *Data) {
	if data == nil {
		data = &Data{}
	}
	logger.Debug(msg, data.lData)
}

// HandleErrorAndExit prints an error and exit with exit code 1
// Only applicable upto server startup since process will be killed once invoked
func HandleErrorAndExit(err error) {
	fmt.Println(err)
	os.Exit(ExitCode1)
}

// HandleErrorWithLoggerAndExit prints an error through the provided logger and exit with exit code 1
// Only applicable upto server startup since process will be killed once invoked
func HandleErrorWithLoggerAndExit(errMsg string, err error) {
	Error(errMsg, err, &Data{})
	os.Exit(ExitCode1)
}

// NewData returns a pointer a lData struct
func NewData() *Data {
	return &Data{}
}

// Add adds data to current data obj
// Returns a reference to the current Log data obj
func (l *Data) Add(key string, val interface{}) *Data {
	if l.lData == nil {
		l.lData = lager.Data{}
	}
	if key != "" && val != nil {
		l.lData[key] = val
	}
	return l
}
