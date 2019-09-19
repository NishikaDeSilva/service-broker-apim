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

package apim

import (
	"github.com/jarcoal/httpmock"
	"net/http"
	"strconv"
	"testing"
)

const (
	publisherTestEndpoint     = "http://localhost/publisher"
	storeTestEndpoint         = "http://localhost/store"
	successTestCase           = "success test case"
	failureTestCase           = "failure test case"
	ErrMsgTestIncorrectResult = "expected value: %v but then returned value: %v"
)

type MockTokenManager struct {
}

func (m *MockTokenManager) Token(scope string) (string, error) {
	return "token", nil
}

func (m *MockTokenManager) InitTokenManager(scopes ...string){

}

var (
	tM = &MockTokenManager{}

	apiM = &Client{
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
			ID: id,
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
			t.Errorf(ErrMsgTestIncorrectResult, id, apiID)
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

func TestUpdateAPI(t *testing.T) {
	t.Run("success test case", testUpdateAPISuccessFunc())
	t.Run("failure test case", testUpdateAPIFailFunc())
}

func testUpdateAPISuccessFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPut, publisherTestEndpoint+PublisherContext+"/id", responder)

		err = apiM.UpdateAPI("id", &APIReqBody{
			Name: "test API",
		})
		if err != nil {
			t.Error(err)
		}
	}
}

func testUpdateAPIFailFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusNotFound, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPut, publisherTestEndpoint+PublisherContext+"/id", responder)

		err = apiM.UpdateAPI("id", &APIReqBody{
			Name: "test API",
		})
		if err == nil {
			t.Error("Expecting an error with code: " + strconv.Itoa(http.StatusNotFound))
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
		if err.Error() != ErrMSGAPIIDEmpty {
			t.Error("Expecting an error : " + ErrMSGAPIIDEmpty + " got: " + err.Error())
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
		responder, err := httpmock.NewJsonResponder(http.StatusCreated, &AppCreateRes{ApplicationID: "1"})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, storeTestEndpoint+StoreApplicationContext, responder)

		id, err := apiM.CreateApplication(&ApplicationCreateReq{
			Name: "test",
		})
		if id != "1" {
			t.Errorf(ErrMsgTestIncorrectResult, "1", id)
		}
	}
}

func TestUpdateApplication(t *testing.T) {
	t.Run(successTestCase, testUpdateApplicationSuccessFunc())
	t.Run(failureTestCase, testUpdateApplicationFailFunc())
}

func testUpdateApplicationFailFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusNotFound, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPut, storeTestEndpoint+StoreApplicationContext+"/id", responder)

		err = apiM.UpdateApplication("id", &ApplicationCreateReq{
			Name: "test",
		})
		if err == nil {
			t.Error("Expecting an error with code: " + strconv.Itoa(http.StatusNotFound))
		}
	}
}

func testUpdateApplicationSuccessFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPut, storeTestEndpoint+StoreApplicationContext+"/id", responder)

		err = apiM.UpdateApplication("id", &ApplicationCreateReq{
			Name: "test",
		})
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
		if err.Error() != ErrMSGAPPIDEmpty {
			t.Error("Expecting an error : " + ErrMSGAPPIDEmpty + " got: " + err.Error())
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
		responder, err := httpmock.NewJsonResponder(http.StatusOK, &ApplicationKeyResp{
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
			t.Errorf(ErrMsgTestIncorrectResult, "abc", got.Token.AccessToken)
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
			SubscriptionID: "abc",
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
			t.Errorf(ErrMsgTestIncorrectResult, "abc", got)
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
				ID: "111-111",
			}},
		})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodGet, publisherTestEndpoint+PublisherContext+"?query=Test", responder)
		apiID, err := apiM.SearchAPI("Test")
		if err != nil {
			t.Error(err)
		}
		if apiID != "111-111" {
			t.Errorf(ErrMsgTestIncorrectResult, "Test", apiID)
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
				ApplicationID: "111-111",
			}},
		})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodGet, storeTestEndpoint+StoreApplicationContext+"?query=Test", responder)
		apiID, err := apiM.SearchApplication("Test")
		if err != nil {
			t.Error(err)
		}
		if apiID != "111-111" {
			t.Errorf(ErrMsgTestIncorrectResult, "Test", apiID)
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
