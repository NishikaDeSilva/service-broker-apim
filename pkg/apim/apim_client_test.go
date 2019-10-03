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
	"fmt"
	"github.com/jarcoal/httpmock"
	"github.com/wso2/service-broker-apim/pkg/config"
	"net/http"
	"strconv"
	"testing"
)

const (
	publisherTestEndpoint              = "http://localhost/publisher"
	storeTestEndpoint                  = "http://localhost/store"
	StoreApplicationContext            = "/api/am/store/v0.14/applications"
	StoreSubscriptionContext           = "/api/am/store/v0.14/subscriptions"
	GenerateApplicationKeyContext      = StoreApplicationContext + "/generate-keys"
	PublisherContext                   = "/api/am/publisher/v0.14/apis"
	PublisherChangeAPILifeCycleContext = PublisherContext + "/change-lifecycle"
	successTestCase                    = "success test case"
	failureTestCase                    = "failure test case"
	ErrMsgTestIncorrectResult          = "expected value: %v but then returned value: %v"
)

type MockTokenManager struct {
}

func (m *MockTokenManager) Token() (string, error) {
	return "token", nil
}

func (m *MockTokenManager) Init(scopes []string) {

}

func init() {
	Init(&MockTokenManager{}, config.APIM{
		PublisherEndpoint:                  publisherTestEndpoint,
		PublisherAPIContext:                PublisherContext,
		StoreEndpoint:                      storeTestEndpoint,
		StoreSubscriptionContext:           StoreSubscriptionContext,
		StoreApplicationContext:            StoreApplicationContext,
		GenerateApplicationKeyContext:      GenerateApplicationKeyContext,
		PublisherChangeAPILifeCycleContext: PublisherChangeAPILifeCycleContext,
	})

}

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

		apiID, err := CreateAPI(&APIReqBody{
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

		_, err = CreateAPI(&APIReqBody{
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

		err = UpdateAPI("id", &APIReqBody{
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

		err = UpdateAPI("id", &APIReqBody{
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
		httpmock.RegisterResponder(http.MethodPost, publisherTestEndpoint+PublisherChangeAPILifeCycleContext, responder)

		err = PublishAPI("")
		if err.Error() != ErrMsgAPIIDEmpty {
			t.Error("Expecting an error : " + ErrMsgAPIIDEmpty + " got: " + err.Error())
		}
		err = PublishAPI("123")
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
		httpmock.RegisterResponder(http.MethodPost, publisherTestEndpoint+PublisherChangeAPILifeCycleContext, responder)

		err = PublishAPI("123")
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

		_, err = CreateApplication(&ApplicationCreateReq{})
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

		id, err := CreateApplication(&ApplicationCreateReq{
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

		err = UpdateApplication("id", &ApplicationCreateReq{
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

		err = UpdateApplication("id", &ApplicationCreateReq{
			Name: "test",
		})
		if err != nil {
			t.Error(err)
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

		_, err = GenerateKeys("")
		if err.Error() != ErrMsgAPPIDEmpty {
			t.Error("Expecting an error : " + ErrMsgAPPIDEmpty + " got: " + err.Error())
		}
		_, err = GenerateKeys("")
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

		got, err := GenerateKeys("123")
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

		_, err = Subscribe("123", "123", "gold")
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

		got, err := Subscribe("123", "123", "gold")
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

		err = UnSubscribe("abc")
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

		err = UnSubscribe("abc")
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

		err = DeleteApplication("abc")
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

		err = DeleteApplication("abc")
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

		err = DeleteAPI("abc")
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

		err = DeleteAPI("abc")
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
		apiID, err := SearchAPI("Test")
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
		_, err = SearchAPI("Test")
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
		_, err = SearchAPI("Test")
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
		apiID, err := SearchApplication("Test")
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
		_, err = SearchApplication("Test")
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
		_, err = SearchApplication("Test")
		if err == nil {
			t.Error("Expecting an error")
		}
		if err.Error() != "returned more than one Application for Test" {
			t.Error("Expecting the error 'returned more than one Application for Test' but got " + err.Error())
		}
	}
}

func TestGetAPIParamHash(t *testing.T) {
	apiParam1 := APIParam{
		APISpec: APIReqBody{
			Name:  "test app",
			Tiers: []string{"a", "b"},
			CorsConfiguration: &APICorsConfiguration{
				AccessControlAllowOrigins: []string{"a", "b"},
			},
		},
	}
	apiParam2 := APIParam{
		APISpec: APIReqBody{
			Name:  "test app",
			Tiers: []string{"b", "a"},
			CorsConfiguration: &APICorsConfiguration{
				AccessControlAllowOrigins: []string{"b", "a"},
			},
		},
	}

	hash1, err := GetAPIParamHash(apiParam1)
	if err != nil {
		t.Error(err)
	}
	expected := "2683cc0a521392c56992db1f78d5835f8af7b367ae6261b529826512147200ba"
	if expected != hash1 {
		t.Error(fmt.Sprintf(ErrMsgTestIncorrectResult, expected, hash1))
	}

	hash2, err := GetAPIParamHash(apiParam2)
	if err != nil {
		t.Error(err)
	}
	if expected != hash2 {
		t.Error(fmt.Sprintf(ErrMsgTestIncorrectResult, expected, hash2))
	}
}

func TestGetAppParamHash(t *testing.T) {
	appParam := ApplicationParam{
		AppSpec: ApplicationCreateReq{
			Name:           "test app",
			Description:    "description",
			ThrottlingTier: "unlimited",
			CallbackURL:    "dummy",
		},
	}
	hash, err := GetAppParamHash(appParam)
	if err != nil {
		t.Error(err)
	}
	expected := "480afa8f3755c39327360c8f1989c5afda8e84b95b1be941d465777d477c24d7"
	if expected != hash {
		t.Error(fmt.Sprintf(ErrMsgTestIncorrectResult, expected, hash))
	}
}

func TestGetSubsParamHash(t *testing.T) {
	subsParam := SubscriptionParam{
		SubsSpec: SubscriptionSpec{
			SubscriptionTier: "tier",
			AppName:          "test app",
			APIName:          "test API",
		},
	}
	hash, err := GetSubsParamHash(subsParam)
	if err != nil {
		t.Error(err)
	}
	expected := "8fd67e25d83849761f44a91d4c5757b51688e04650c44d8b2130952ce2359d67"
	if expected != hash {
		t.Error(fmt.Sprintf(ErrMsgTestIncorrectResult, expected, hash))
	}
}
