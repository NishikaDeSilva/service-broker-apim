/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */

package broker

import (
	"github.com/jarcoal/httpmock"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"net/http"
	"strconv"
	"testing"
	"time"
)

const (
	publisherTestEndpoint = "http://localhost/publisher"
	storeTestEndpoint     = "http://localhost/store"
	successTestCase       = "success test case"
	failureTestCase       = "failure test case"
)

var (
	dummyToken = &tokens{
		aT:        "token",
		expiresIn: time.Now().Add(10 * time.Second),
	}
	tM = &TokenManager{
		holder: map[string]*tokens{
			ScopeAPICreate:  dummyToken,
			ScopeAPIPublish: dummyToken,
			ScopeSubscribe:  dummyToken,
		},
	}
	apiM = &APIMManager{
		PublisherEndpoint: publisherTestEndpoint,
		StoreEndpoint:     storeTestEndpoint,
	}
)

func TestCreateAPI(t *testing.T) {
	t.Run("success test case", testCreateAPISuccessFunc())
	t.Run("failure test case", testCreateAPIFailFunc())
}

func testCreateAPISuccessFunc() func(t *testing.T) {
	return func(t *testing.T) {
		var id = "id"
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusCreated, APICreateResp{
			Id: id,
		})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, publisherTestEndpoint+PublisherContext, responder)

		apiID, err := apiM.CreateAPI(&APIReqBody{
			Name: "test API",
		}, tM)
		if err != nil {
			t.Error(err)
		}
		if apiID != id {
			t.Errorf(constants.ErrMsgTestIncorrectResult, id, apiID)
		}
	}
}

func testCreateAPIFailFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusConflict, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, publisherTestEndpoint+PublisherContext, responder)

		_, err = apiM.CreateAPI(&APIReqBody{
			Name: "test API",
		}, tM)
		if err == nil {
			t.Error("Expecting an error with code: " + strconv.Itoa(http.StatusConflict))
		}
	}
}

func TestPublishAPI(t *testing.T) {
	t.Run(successTestCase, testPublishAPISuccessFunc())
	t.Run(failureTestCase, testPublishAPIFailFunc())
}

func testPublishAPIFailFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusNotFound, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, publisherTestEndpoint+PublisherChangeAPIContext, responder)

		err = apiM.PublishAPI("", tM)
		if err.Error() != constants.ErrMSGAPIIDEmpty {
			t.Error("Expecting an error : " + constants.ErrMSGAPIIDEmpty + " got: " + err.Error())
		}
		err = apiM.PublishAPI("123", tM)
		if err == nil {
			t.Error("Expecting an error with code: " + strconv.Itoa(http.StatusNotFound))
		}
	}
}

func testPublishAPISuccessFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, publisherTestEndpoint+PublisherChangeAPIContext, responder)

		err = apiM.PublishAPI("123", tM)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestCreateApplication(t *testing.T) {
	t.Run(successTestCase, testCreateApplicationSuccessFunc())
	t.Run(failureTestCase, testCreateApplicationFailFunc())
}

func testCreateApplicationFailFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusInternalServerError, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, storeTestEndpoint+StoreApplicationContext, responder)

		_, err = apiM.CreateApplication(&ApplicationCreateReq{}, tM)
		if err == nil {
			t.Error("Expecting an error with code: " + strconv.Itoa(http.StatusInternalServerError))
		}
	}
}

func testCreateApplicationSuccessFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusCreated, &AppCreateRes{ApplicationId: "1"})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, storeTestEndpoint+StoreApplicationContext, responder)

		id, err := apiM.CreateApplication(&ApplicationCreateReq{
			Name: "test",
		}, tM)
		if id != "1" {
			t.Errorf(constants.ErrMsgTestIncorrectResult, "1", id)
		}
	}
}

func TestGenerateKeys(t *testing.T) {
	t.Run(successTestCase, testGenerateKeysSuccessFunc())
	t.Run(failureTestCase, testGenerateKeysFailFunc())
}

func testGenerateKeysFailFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusInternalServerError, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, storeTestEndpoint+GenerateApplicationKeyContext, responder)

		_, err = apiM.GenerateKeys("", tM)
		if err.Error() != constants.ErrMSGAPPIDEmpty {
			t.Error("Expecting an error : " + constants.ErrMSGAPPIDEmpty + " got: " + err.Error())
		}
		_, err = apiM.GenerateKeys("", tM)
		if err == nil {
			t.Error("Expecting an error with code: " + strconv.Itoa(http.StatusInternalServerError))
		}
	}
}

func testGenerateKeysSuccessFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, &ApplicationKey{
			Token: &Token{AccessToken: "abc"},
		})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, storeTestEndpoint+GenerateApplicationKeyContext, responder)

		got, err := apiM.GenerateKeys("123", tM)
		if err != nil {
			t.Error(err)
		}
		if got.Token.AccessToken != "abc" {
			t.Errorf(constants.ErrMsgTestIncorrectResult, "abc", got.Token.AccessToken)
		}
	}
}

func TestSubscribe(t *testing.T) {
	t.Run(successTestCase, testSubscribeSuccessFunc())
	t.Run(failureTestCase, testSubscribeFailFunc())
}

func testSubscribeFailFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusInternalServerError, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, storeTestEndpoint+StoreSubscriptionContext, responder)

		_, err = apiM.Subscribe("123", "123", "gold", tM)
		if err == nil {
			t.Error("Expecting an error with code: " + strconv.Itoa(http.StatusInternalServerError))
		}
	}
}

func testSubscribeSuccessFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusCreated, &SubscriptionResp{
			SubscriptionId: "abc",
		})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, storeTestEndpoint+StoreSubscriptionContext, responder)

		got, err := apiM.Subscribe("123", "123", "gold", tM)
		if err != nil {
			t.Error(err)
		}
		if got != "abc" {
			t.Errorf(constants.ErrMsgTestIncorrectResult, "abc", got)
		}
	}
}

func TestUnSubscribe(t *testing.T) {
	t.Run(successTestCase, testUnSubscribeSuccessFunc())
	t.Run(failureTestCase, testUnSubscribeFailFunc())
}

func testUnSubscribeFailFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusInternalServerError, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodDelete, storeTestEndpoint+StoreSubscriptionContext+"/abc", responder)

		err = apiM.UnSubscribe("abc", tM)
		if err == nil {
			t.Error("Expecting an error with code: " + strconv.Itoa(http.StatusInternalServerError))
		}
	}
}

func testUnSubscribeSuccessFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodDelete, storeTestEndpoint+StoreSubscriptionContext+"/abc", responder)

		err = apiM.UnSubscribe("abc", tM)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestDeleteApplication(t *testing.T) {
	t.Run(successTestCase, testDeleteApplicationSuccessFunc())
	t.Run(failureTestCase, testDeleteApplicationFailFunc())
}

func testDeleteApplicationFailFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusInternalServerError, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodDelete, storeTestEndpoint+StoreApplicationContext+"/abc", responder)

		err = apiM.DeleteApplication("abc", tM)
		if err == nil {
			t.Error("Expecting an error with code: " + strconv.Itoa(http.StatusInternalServerError))
		}
	}
}

func testDeleteApplicationSuccessFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodDelete, storeTestEndpoint+StoreApplicationContext+"/abc", responder)

		err = apiM.DeleteApplication("abc", tM)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestDeleteAPI(t *testing.T) {
	t.Run(successTestCase, testDeleteAPISuccessFunc())
	t.Run(failureTestCase, testDeleteAPIFailFunc())
}

func testDeleteAPIFailFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusInternalServerError, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodDelete, publisherTestEndpoint+PublisherContext+"/abc", responder)

		err = apiM.DeleteAPI("abc", tM)
		if err == nil {
			t.Error("Expecting an error with code: " + strconv.Itoa(http.StatusInternalServerError))
		}
	}
}

func testDeleteAPISuccessFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodDelete, publisherTestEndpoint+PublisherContext+"/abc", responder)

		err = apiM.DeleteAPI("abc", tM)
		if err != nil {
			t.Error(err)
		}
	}
}
