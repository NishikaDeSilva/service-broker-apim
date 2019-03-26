package utils

import (
	"github.com/wso2/service-broker-apim/internal/constants"
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	envKey := "TEST_KEY"
	envVal := "testValue"
	envDefault := "default"
	if err := os.Setenv(envKey, envVal); err != nil {
		t.Errorf(constants.ErrMsgTestCouldNotSetEnv, envKey)
	}
	re1 := GetEnv(envKey, envDefault)
	if re1 != envVal {
		t.Errorf(constants.ErrMsgTestIncorrectResult, envVal, re1)
	}
	re2 := GetEnv("", envDefault)
	if re2 != envDefault {
		t.Errorf(constants.ErrMsgTestIncorrectResult, envDefault, re2)
	}
}
