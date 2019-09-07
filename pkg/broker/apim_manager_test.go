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
		aT: "token",
		// Make sure the expire time is enough to run all test cases since token
		// might be expired in the middle of the testing due to retrying.
		expiresIn: time.Now().Add(100 * time.Second),
	}
	tM = &TokenManager{
		holder: map[string]*tokens{
			ScopeAPICreate:  dummyToken,
			ScopeAPIPublish: dummyToken,
			ScopeSubscribe:  dummyToken,
			ScopeAPIView:    dummyToken,
		},
	}
	apiM = &APIMClient{
		PublisherEndpoint: publisherTestEndpoint,
		StoreEndpoint:     storeTestEndpoint,
		TokenManager:      tM,
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
		})
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
		})
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

		err = apiM.PublishAPI("")
		if err.Error() != constants.ErrMSGAPIIDEmpty {
			t.Error("Expecting an error : " + constants.ErrMSGAPIIDEmpty + " got: " + err.Error())
		}
		err = apiM.PublishAPI("123")
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

		err = apiM.PublishAPI("123")
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

		_, err = apiM.CreateApplication(&ApplicationCreateReq{})
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
		})
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

		_, err = apiM.GenerateKeys("")
		if err.Error() != constants.ErrMSGAPPIDEmpty {
			t.Error("Expecting an error : " + constants.ErrMSGAPPIDEmpty + " got: " + err.Error())
		}
		_, err = apiM.GenerateKeys("")
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

		got, err := apiM.GenerateKeys("123")
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

		_, err = apiM.Subscribe("123", "123", "gold")
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

		got, err := apiM.Subscribe("123", "123", "gold")
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

		err = apiM.UnSubscribe("abc")
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

		err = apiM.UnSubscribe("abc")
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

		err = apiM.DeleteApplication("abc")
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

		err = apiM.DeleteApplication("abc")
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

		err = apiM.DeleteAPI("abc")
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

		err = apiM.DeleteAPI("abc")
		if err != nil {
			t.Error(err)
		}
	}
}

func TestSearchAPI(t *testing.T) {
	t.Run(successTestCase, testSearchAPInSuccessFunc())
	t.Run("failure test case 1", testSearchAPIFail1Func())
	t.Run("failure test case 2", testSearchAPIFail2Func())
}

func testSearchAPInSuccessFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, &APISearchResp{
			Count: 1,
			List: []APISearchInfo{{
				Id: "111-111",
			}},
		})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodGet, publisherTestEndpoint+PublisherContext+"?query=Test", responder)
		apiId, err := apiM.SearchAPI("Test")
		if err != nil {
			t.Error(err)
		}
		if apiId != "111-111" {
			t.Errorf(constants.ErrMsgTestIncorrectResult, "Test", apiId)
		}
	}
}

func testSearchAPIFail1Func() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, &APISearchResp{
			Count: 0,
		})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodGet, publisherTestEndpoint+PublisherContext+"?query=Test", responder)
		_, err = apiM.SearchAPI("Test")
		if err == nil {
			t.Error("Expecting an error")
		}
		if err.Error() != "couldn't find the API Test" {
			t.Error("Expecting the error 'couldn't find the API Test' but got " + err.Error())
		}
	}
}
func testSearchAPIFail2Func() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, &APISearchResp{
			Count: 2,
		})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodGet, publisherTestEndpoint+PublisherContext+"?query=Test", responder)
		_, err = apiM.SearchAPI("Test")
		if err == nil {
			t.Error("Expecting an error")
		}
		if err.Error() != "returned more than one API for API Test" {
			t.Error("Expecting the error 'returned more than one API for API Test' but got " + err.Error())
		}
	}
}

func TestSearchApplication(t *testing.T) {
	t.Run(successTestCase, testSearchApplicationSuccessFunc())
	t.Run("failure test case 1", testSearchApplicationFail1Func())
	t.Run("failure test case 2", testSearchApplicationFail2Func())
}

func testSearchApplicationSuccessFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, &ApplicationSearchResp{
			Count: 1,
			List: []ApplicationSearchInfo{{
				ApplicationId: "111-111",
			}},
		})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodGet, storeTestEndpoint+StoreApplicationContext+"?query=Test", responder)
		apiId, err := apiM.SearchApplication("Test")
		if err != nil {
			t.Error(err)
		}
		if apiId != "111-111" {
			t.Errorf(constants.ErrMsgTestIncorrectResult, "Test", apiId)
		}
	}
}

func testSearchApplicationFail1Func() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, &APISearchResp{
			Count: 0,
		})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodGet, storeTestEndpoint+StoreApplicationContext+"?query=Test", responder)
		_, err = apiM.SearchApplication("Test")
		if err == nil {
			t.Error("Expecting an error")
		}
		if err.Error() != "couldn't find the Application Test" {
			t.Error("Expecting the error 'couldn't find the Application Test' but got " + err.Error())
		}
	}
}

func testSearchApplicationFail2Func() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, &ApplicationSearchResp{
			Count: 2,
		})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodGet, storeTestEndpoint+StoreApplicationContext+"?query=Test", responder)
		_, err = apiM.SearchApplication("Test")
		if err == nil {
			t.Error("Expecting an error")
		}
		if err.Error() != "returned more than one Application for Test" {
			t.Error("Expecting the error 'returned more than one Application for Test' but got " + err.Error())
		}
	}
}
