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

// Package broker hologDatas the implementation of brokerapi.ServiceBroker interface.
package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mitchellh/hashstructure"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/apim"
	"github.com/wso2/service-broker-apim/pkg/client"
	"github.com/wso2/service-broker-apim/pkg/db"
	"github.com/wso2/service-broker-apim/pkg/log"
	"github.com/wso2/service-broker-apim/pkg/model"
	"github.com/wso2/service-broker-apim/pkg/utils"
)

const (
	LogKeyAppID                      = "application-id"
	LogKeyServiceID                  = "service-id"
	LogKeyPlanID                     = "plan-id"
	LogKeyInstanceID                 = "instance-id"
	LogKeyBindID                     = "bind-id"
	LogKeyApplicationName            = "application-name"
	LogKeyPlatformApplicationName    = "platform-application-name"
	LogKeySpaceID                    = "cf-space-id"
	LogKeyOrgID                      = "cf-org-id"
	ApplicationDashboardURL          = "application-dashboard-url"
	ErrMsgUnableToStoreInstance      = "unable to store service instance in database"
	ErrActionStoreInstance           = "store service instance in DB"
	ErrActionDelAPP                  = "delete Application"
	ErrActionDelInstanceFromDB       = "delete service instance from DB"
	ErrActionCreateAPIMResource      = "creating API-M resource"
	ErrActionUpdateAPIMResource      = "update API-M resource"
	ErrMsgUnableDelInstance          = "unable to delete service instance"
	ErrMsgUnableToGetBindFromDB      = "unable to retrieve Bind from the database"
	ErrMsgUnableToGenInputSchema     = "unable to generate %s plan input Schema"
	ErrMsgUnableToGenBindInputSchema = "unable to generate %s plan bind input Schema"
	ErrMsgInvalidPlanID              = "invalid plan id"
	ErrMsgUnableGenerateKeys         = "unable generate keys for application"
	ServiceID                        = "460F28F9-4D05-4889-970A-6BF5FB7D3CF8"
	ServiceName                      = "wso2apim-service"
	ServiceDescription               = "Manages WSO2 API Manager artifacts"
	ApplicationPlanID                = "00e851cd-ce8b-43eb-bc27-ac4d4fbb3204"
	ApplicationPlanName              = "app"
	ApplicationPlanDescription       = "Creates an Application with a set of subscription for a given set of APIs in WSO2 API Manager"
	DebugMsgDelInstanceFromDB        = "delete instance from DB"
	ApplicationPrefix                = "ServiceBroker_"
	StatusInstanceAlreadyExists      = "Instance already exists"
	StatusInstanceDoesNotExist       = "Instance does not exist"
)

var (
	ErrNotSupported                 = errors.New("not supported")
	ErrInvalidSVCPlan               = errors.New("invalid service or getServices")
	applicationPlanBindable         = true
	appPlanInputParameterSchema     map[string]interface{}
	appPlanBindInputParameterSchema map[string]interface{}
)

// APIM struct implements the interface brokerapi.ServiceBroker.
type APIM struct{}

// API struct represent an API.
type API struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServiceParams represents the SVC create and update parameter.
type ServiceParams struct {
	APIs []API `json:"apis" hash:"set"`
}

type ServiceAtrributes struct {
	serviceParams  ServiceParams
	organizationID string
	spaceID        string
}

type InternalServerFailureResponseError struct {
	errorMessage      string
	loggerAction      string
	sysError          error
	revertApplication *ConfigApplicationError
}

type BadRequestFailureResponseError struct {
	errorMessage string
	loggerAction string
}

type ConfigApplicationError struct {
	err           error
	applicationID string
	logData       *log.Data
}

type BindingNotExistsError struct{}

// Init method initialize the API-M broker. If there is an error it will cause a panic.
func (apimBroker *APIM) Init() {
	var err error
	appPlanInputParameterSchema, err = utils.GetJSONSchema(apim.AppPlanInputParameterSchemaRaw)
	if err != nil {
		log.HandleErrorAndExit(fmt.Sprintf(ErrMsgUnableToGenInputSchema, "app"), err)
	}
	appPlanBindInputParameterSchema, err = utils.GetJSONSchema(apim.AppPlanBindInputParameterSchemaRaw)
	if err != nil {
		log.HandleErrorAndExit(fmt.Sprintf(ErrMsgUnableToGenBindInputSchema, "app"), err)
	}
}

// Services returns the getServices offered(catalog) by this broker.
func (apimBroker *APIM) Services(ctx context.Context) ([]domain.Service, error) {
	return getServices()
}

func createCommonLogData(svcInstanceID, serviceID, planID string) *log.Data {
	return log.NewData().
		Add(LogKeyServiceID, serviceID).
		Add(LogKeyPlanID, planID).
		Add(LogKeyInstanceID, svcInstanceID)
}

func getValidServiceParams(rawParam json.RawMessage, logData *log.Data) (ServiceParams, error) {
	s, err := unmarshalServiceParams(rawParam)
	if err != nil {
		log.Error("unable to parse parameters", err, logData)
		return s, errors.Wrap(invalidParamFailureResponse("parse ServiceParams"), "error in getting valid service paramters")
	}

	if len(s.APIs) == 0 {
		log.Error("no APIs defined", nil, logData)
		return s, errors.Wrap(invalidParamFailureResponse("no APIs defined"), "error in gettig valid service parameters")
	}

	return s, nil
}

func hasValidSpaceIDAndOrgID(spaceID string, orgID string) bool {
	return spaceID != "" && orgID != ""
}

func validateHashSpaceIDOrgID(svcInstance *model.ServiceInstance, paramHash string, spaceID string, orgID string) bool {
	return (svcInstance.ParameterHash == paramHash) && (svcInstance.SpaceID == spaceID) && (svcInstance.OrgID == orgID)
}

// checkServiceInstanceWithAttributes function returns domain.ProvisionedServiceSpec and error type apiresponses.FailureResponse.
// If there is a instance with the same attributes the domain.ProvisionedServiceSpec.AlreadyExists will be set to "true".
// and if the instance exist with different attributes apiresponses.ErrInstanceAlreadyExists is returned.
func checkServiceInstanceWithAttributes(svcInstanceID string, svcAttributes *ServiceAtrributes, logData *log.Data) (string, error) { //TODO: dstruct attribute containing space, org, params

	svcInstance, err := getServiceInstanceFromDB(svcInstanceID, logData)
	if err != nil {
		//	return domain.ProvisionedServiceSpec{}, err return check...Error
		return "", errors.Wrap(err, "Unable to retrieve instance from database")
	}

	if svcInstance != nil {
		parameterHash, err := generateHashForServiceParam(svcInstance.ApplicationID, svcAttributes.serviceParams, logData)
		if err != nil {
			//	return domain.ProvisionedServiceSpec{}, err return check...error
			return "", err
		}
		if ok := validateHashSpaceIDOrgID(svcInstance, parameterHash, svcAttributes.spaceID, svcAttributes.organizationID); ok { //check: naming

			existingAPIs, err := getExistingAPIsForAppID(svcInstance.ApplicationID, logData)
			if err != nil {
				return "", err
			}
			if !isSameAPIs(existingAPIs, svcAttributes.serviceParams.APIs) {
				log.Debug("APIs does not match", logData)
				return "", apiresponses.ErrInstanceAlreadyExists //check: define another error > response error 509
			}

			return StatusInstanceAlreadyExists, nil
		}

	}

	return StatusInstanceDoesNotExist, nil
}

func removeSubscriptions(removedSubsIds []string, svcInstance string) error {

	for _, sub := range removedSubsIds {
		err := apim.UnSubscribe(sub)
		if err != nil {
			return err
		}
		err = removeSubscriptionFromDB(sub, svcInstance)
		if err != nil {
			return err
		}
	}
	return nil
}

func removeSubscriptionFromDB(subId, svcInstId string) error {
	sub := &model.Subscription{
		ID:            subId,
		SVCInstanceID: svcInstId,
	}
	err := db.Delete(sub)
	if err != nil {
		return err
	}

	return nil
}

func getRemovedSubscriptionsIDs(applicationID string, existingAPIs, requestedAPIs []API, logData *log.Data) ([]string, error) {
	removedAPIs := getRemovedAPIs(existingAPIs, requestedAPIs)
	var removedSubsIDs []string
	for _, rAPI := range removedAPIs {
		rSub, err := getSubscriptionForAppAndAPI(applicationID, rAPI, logData)
		if err != nil {
			return nil, err
		}
		removedSubsIDs = append(removedSubsIDs, rSub.ID)
	}
	return removedSubsIDs, nil
}

func getRemovedAPIs(existingAPIs, requestedAPIs []API) []API {
	var removedAPIs []API
	for _, api := range existingAPIs {
		if !isArrayContainAPI(requestedAPIs, api) {
			removedAPIs = append(removedAPIs, api)
		}
	}
	return removedAPIs
}

func getAddedAPIs(existingAPIs, requestedAPIs []API, logData *log.Data) []API {
	var addedAPIs []API
	for _, api := range requestedAPIs {
		if !isArrayContainAPI(existingAPIs, api) {
			addedAPIs = append(addedAPIs, api)
		}
	}
	return addedAPIs
}

func (apimBroker *APIM) GetBinding(ctx context.Context, svcInstanceID,
	bindingID string) (domain.GetBindingSpec, error) {
	return domain.GetBindingSpec{}, ErrNotSupported
}

func (apimBroker *APIM) GetInstance(ctx context.Context,
	svcInstanceID string) (domain.GetInstanceDetailsSpec, error) {
	return domain.GetInstanceDetailsSpec{}, ErrNotSupported
}

func (apimBroker *APIM) LastBindingOperation(ctx context.Context, svcInstanceID,
	bindingID string, details domain.PollDetails) (domain.LastOperation, error) {
	return domain.LastOperation{}, ErrNotSupported
}

// getServices returns an array of getServices offered by this service broker and any error encountered.
func getServices() ([]domain.Service, error) {
	return []domain.Service{
		{
			ID:                   ServiceID,
			Name:                 ServiceName,
			Description:          ServiceDescription,
			Bindable:             true,
			InstancesRetrievable: false,
			PlanUpdatable:        true,
			Plans: []domain.ServicePlan{
				{
					ID:          ApplicationPlanID,
					Name:        ApplicationPlanName,
					Description: ApplicationPlanDescription,
					Bindable:    &applicationPlanBindable,
					Schemas: &domain.ServiceSchemas{
						Instance: domain.ServiceInstanceSchema{
							Create: domain.Schema{
								Parameters: appPlanInputParameterSchema,
							},
							Update: domain.Schema{
								Parameters: appPlanInputParameterSchema,
							},
						},
						Binding: domain.ServiceBindingSchema{
							Create: domain.Schema{
								Parameters: appPlanBindInputParameterSchema,
							},
						},
					},
				},
			},
		},
	}, nil
}

// createApplication creates Subscription in API-M and returns App ID, App dashboard URL and
// an error type apiresponses.FailureResponse if encountered.
func createApplication(appName string, logData *log.Data) (string, string, error) {
	req := &apim.ApplicationCreateReq{
		Name:           appName,
		ThrottlingTier: "Unlimited",
		Description:    "Application " + appName + " created by WSO2 APIM Service Broker",
	}
	appID, err := apim.CreateApplication(req)
	if err != nil {
		log.Error("unable to create application", err, logData)
		return "", "", handleAPIMResourceCreateError(err, appName, logData)
	}
	dashboardURL := apim.GetAppDashboardURL(appName)
	return appID, dashboardURL, nil
}

// handleAPIMResourceCreateError handles the API-M resource creation error. Returns an error type apiresponses.FailureResponse.
func handleAPIMResourceCreateError(e error, resourceName string, logData *log.Data) error {
	invokeErr, ok := e.(*client.InvokeError)
	if ok && invokeErr.StatusCode == http.StatusConflict {
		log.Debug("API-M resource already exists", logData)
		return &InternalServerFailureResponseError{
			errorMessage: fmt.Sprintf("API-M resource %s already exist", resourceName),
			loggerAction: ErrActionCreateAPIMResource,
		}
	}

	log.Debug("unable to create API-M resource", logData)
	return &InternalServerFailureResponseError{
		errorMessage: "unable to create the API-M resource",
		loggerAction: ErrActionCreateAPIMResource,
	}
}

// getServiceInstanceFromDB function check whether the given instance already exists. If the given instance exists then an initialized instance and
// if the given instance doesn't exist or unable to retrieve it from database, an error type apiresponses.FailureResponse is returned.
func getServiceInstanceFromDB(svcInstanceID string, logData *log.Data) (*model.ServiceInstance, error) {
	instance := &model.ServiceInstance{
		ID: svcInstanceID,
	}
	exists, err := db.Retrieve(instance)
	if err != nil {
		log.Error("unable to retrieve the service instance from database", err, logData)
		return nil, &InternalServerFailureResponseError{
			errorMessage: "unable to get Service instance from database",
			loggerAction: "get instance from the database",
			sysError:     err,
		}
	}
	if !exists {
		log.Debug("instance doesn't exists", logData)
		return nil, nil //check: add error ... it's not internal failure though.| this is not an error; this is suppose to happen
	}
	return instance, nil
}

// deleteInstanceFromDB function deletes the given instance from database. A domain.DeprovisionServiceSpec{} and an error type apiresponses.FailureResponse is returned.
func deleteInstanceFromDB(i *model.ServiceInstance, logData *log.Data) error {
	err := db.Delete(i)
	if err != nil {
		log.Error("unable to delete the instance from the database", err, logData)
		err := &InternalServerFailureResponseError{
			errorMessage: ErrMsgUnableDelInstance,
			loggerAction: ErrActionDelInstanceFromDB,
		}
		return errors.Wrap(err, "unable to delete the instance from the database")
	}
	return nil
}

// handleAPIMResourceUpdateError handles the API-M resource update errors. Returns an error type apiresponses.FailureResponse.
func handleAPIMResourceUpdateError(err error, resourceName string) error {
	e, ok := err.(*client.InvokeError)
	if ok && e.StatusCode == http.StatusNotFound {
		return &InternalServerFailureResponseError{
			errorMessage: fmt.Sprintf("API-M resource %s not found !", resourceName),
			loggerAction: ErrActionUpdateAPIMResource,
		}
	}
	return &InternalServerFailureResponseError{
		errorMessage: "unable to update the API-M resource ",
		loggerAction: ErrActionUpdateAPIMResource,
	}
}

func invalidPlanFailureResponse(a string) error {
	return &BadRequestFailureResponseError{
		errorMessage: a,
		loggerAction: "invalid getServices or Services",
	}
}

func invalidParamFailureResponse(a string) error {
	return &BadRequestFailureResponseError{
		errorMessage: a,
		loggerAction: "invalid parameters",
	}
}

func (e *InternalServerFailureResponseError) Error() string {
	//return apiresponses.NewFailureResponse(errors.New(e.errorMessage), http.StatusInternalServerError, e.loggerAction)
	return "Internal Failure"
}

func (e *ConfigApplicationError) Error() string {
	return "Error in Application configration. Deleting Application"
}

func (e *BadRequestFailureResponseError) Error() string {
	//	return apiresponses.NewFailureResponse(errors.New(msg),
	//		http.StatusBadRequest, action)
	return "Bad Request Failure"
}

func (e *BindingNotExistsError) Error() string {
	return "Binding does not exists"
}

// func compareParams(appID string, apis []API, logData *log.Data) (bool, error) {

// 	// If existing parameter hash and the new parameter has matches then check each fielogData again to avoid hash collision.
// 	if !isSameAPIs(existingAPIs, apis) {
// 		log.Debug("APIs does not match", logData)
// 		return false, apiresponses.ErrInstanceAlreadyExists
// 	}
// 	return true, nil
// }

func getSubscriptionsListForAppID(applicationID string, logData *log.Data) ([]model.Subscription, error) {
	subscription := &model.Subscription{
		ApplicationID: applicationID,
	}
	var subscriptionsList []model.Subscription
	hasSubscriptions, err := db.RetrieveList(subscription, &subscriptionsList)
	if err != nil {
		log.Error("unable to get subscription from DB", err, logData)
		return nil, &InternalServerFailureResponseError{
			errorMessage: "unable to query database",
			loggerAction: "get the subscription from DB",
		}
	}
	if !hasSubscriptions {
		log.Error("no subscriptions are available", err, logData)
		return nil, &InternalServerFailureResponseError{
			errorMessage: "no subscriptions are available",
			loggerAction: "get the subscription from DB",
		}
	}
	return subscriptionsList, nil
}

func getSubscriptionForAppAndAPI(applicationID string, api API, logData *log.Data) (*model.Subscription, error) {
	subscription := &model.Subscription{
		ApplicationID: applicationID,
		APIName:       api.Name,
		APIVersion:    api.Version,
	}
	hasSubscription, err := db.Retrieve(subscription)
	if err != nil {
		log.Error("unable to get subscription from DB", err, logData)
		return nil, &InternalServerFailureResponseError{
			errorMessage: "unable to query database",
			loggerAction: "get the subscription from DB",
		}
	}
	if !hasSubscription {
		log.Error("no subscription is available", err, logData)
		return nil, &InternalServerFailureResponseError{
			errorMessage: "no subscription is available",
			loggerAction: "get the subscription from DB",
		}
	}
	return subscription, nil
}

func getExistingAPIsForAppID(applicationID string, logData *log.Data) ([]API, error) {
	subscriptionsList, err := getSubscriptionsListForAppID(applicationID, logData)
	if err != nil {
		return nil, errors.Wrap(err, "Could not find subscriptions for the application")
	}
	var existingAPIs []API
	for _, sub := range subscriptionsList {
		existingAPIs = append(existingAPIs, API{
			Name:    sub.APIName,
			Version: sub.APIVersion,
		})
	}
	return existingAPIs, nil
}

func isSameAPIs(existingAPIs, requestedAPIs []API) bool {
	if len(existingAPIs) != len(requestedAPIs) {
		return false
	}
	for _, existingAPI := range existingAPIs { //TODO: try to optimize this (n*n too complex)
		if !isArrayContainAPI(requestedAPIs, existingAPI) {
			return false
		}
	}
	return true
}

func isArrayContainAPI(apisArr []API, api API) bool {
	for _, a := range apisArr {
		if a.Name == api.Name && a.Version == api.Version {
			return true
		}
	}
	return false
}

func createAndAddSubscriptionsToDB(instance *model.ServiceInstance, apis []API, logData *log.Data) error {
	subscriptions, err := createSubscriptions(instance, apis, logData)
	if err != nil {
		handleSubscriptionCreateStoreErrors(instance.ApplicationID, instance.ID, logData)
		return err
	}
	err = storeSubscriptions(subscriptions)
	if err != nil {
		log.Error("unable to store subscriptions", err, logData)
		handleSubscriptionCreateStoreErrors(instance.ApplicationID, instance.ID, logData)
		return err
	}

	return nil
}

func addServiceInstanceToDB(ID string, appData *apim.ApplicationMetadata, spaceID string, orgID string, paramHash string, logData *log.Data) (*model.ServiceInstance, error) {
	svcInstance := &model.ServiceInstance{
		ID:              ID,
		ApplicationID:   appData.ID,
		ApplicationName: appData.Name,
		SpaceID:         spaceID,
		OrgID:           orgID,
		ConsumerKey:     appData.Keys.ConsumerKey,
		ConsumerSecret:  appData.Keys.ConsumerSecret,
		ParameterHash:   paramHash,
	}
	err := storeServiceInstance(svcInstance, logData)
	if err != nil {
		return nil, errors.Wrap(&ConfigApplicationError{
			applicationID: appData.ID,
			logData:       logData,
			err:           err,
		}, "unable to store service instance")
	}
	return svcInstance, nil
}

func createApplicationAndGenerateKeys(id string, logData *log.Data) (*apim.ApplicationMetadata, error) {
	appName := generateApplicationName(id)

	logData.Add(LogKeyApplicationName, appName)
	appID, appDashboardURL, err := createApplication(appName, logData)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create application")
	}
	logData.Add(LogKeyAppID, appID).
		Add(ApplicationDashboardURL, appDashboardURL)

	keys, err := generateKeysForApplication(appID, logData)
	if err != nil {
		return nil, errors.Wrap(&ConfigApplicationError{
			applicationID: appID,
			logData:       logData,
			err:           err,
		}, "unable to store service instance")
	}

	return &apim.ApplicationMetadata{
		Name:         appName,
		ID:           appID,
		Keys:         keys,
		DashboardURL: appDashboardURL,
	}, nil

}

func handleSubscriptionCreateStoreErrors(appID string, svcInstanceID string, logData *log.Data) error {
	return errors.Wrap(&ConfigApplicationError{
		applicationID: appID,
		logData:       logData,
	}, "delete service instance from database")
}

func deleteServiceInstanceAndLogError(svcInstanceID string, logData *log.Data) {
	err := db.Delete(&model.ServiceInstance{
		ID: svcInstanceID,
	})
	if err != nil {
		log.Error(ErrMsgUnableDelInstance, err, logData)
	}
}

func generateHashForServiceParam(appID string, svcParams ServiceParams, logData *log.Data) (string, error) {
	generatedHash, err := hashstructure.Hash(svcParams, nil)
	if err != nil {
		log.Error("unable to generate hash value for service parameters", err, logData)
		err := &InternalServerFailureResponseError{
			errorMessage: "unable to generate hash value for service parameters",
			loggerAction: "generate hash for service parameter",
			revertApplication: &ConfigApplicationError{
				applicationID: appID,
				logData:       logData,
			},
			sysError: err,
		}
		return "", errors.Wrap(err, "unable to generate hash for parameters")
	}
	return strconv.FormatUint(generatedHash, 10), nil
}

func storeSubscriptions(subscriptions []model.Subscription) error {
	var entities []model.Entity
	for _, val := range subscriptions {
		entities = append(entities, val)
	}
	err := db.BulkInsert(entities)
	if err != nil {
		log.Error("unable to store subscriptions", err, nil)
		return &InternalServerFailureResponseError{
			errorMessage: "unable to store subscriptions in the database",
			loggerAction: "store subscriptions in the database",
		}
	}
	return nil
}

func getSubscriptionList(svcInstanceID string, subsResponses []apim.SubscriptionResp) []model.Subscription {
	var subscriptions []model.Subscription
	for _, subsResponse := range subsResponses {
		apiIdentifier := strings.Split(subsResponse.APIIdentifier, "-")
		subs := model.Subscription{
			ID:            subsResponse.SubscriptionID,
			ApplicationID: subsResponse.ApplicationID,
			User:          apiIdentifier[0],
			APIName:       apiIdentifier[1],
			APIVersion:    apiIdentifier[2],
			SVCInstanceID: svcInstanceID,
		}
		subscriptions = append(subscriptions, subs)
	}
	return subscriptions
}

// storeServiceInstance stores service instance in the database returns an error type apiresponses.FailureResponse.
func storeServiceInstance(i *model.ServiceInstance, logData *log.Data) error {
	err := db.Store(i)
	if err != nil {
		log.Error(ErrMsgUnableToStoreInstance, err, logData)
		//revertApplication(i.ApplicationID, logData)
		return &InternalServerFailureResponseError{
			errorMessage: ErrMsgUnableToStoreInstance,
			loggerAction: ErrActionStoreInstance,
			revertApplication: &ConfigApplicationError{
				applicationID: i.ApplicationID,
				logData:       logData,
			},
		}
	}
	return nil
}

func getAPIIDs(apis []API) ([]string, error) {
	var apiIDs []string
	for _, api := range apis {
		apiID, err := apim.SearchAPIByNameVersion(api.Name, api.Version)
		if err != nil {
			return nil, err
		}
		apiIDs = append(apiIDs, apiID)
	}
	return apiIDs, nil
}

func createSubscriptions(svcInstance *model.ServiceInstance, apis []API, logData *log.Data) ([]model.Subscription, error) {

	apiIDs, err := getAPIIDs(apis)
	if err != nil {
		log.Error("unable to get API ids", err, logData)
		return nil, &InternalServerFailureResponseError{
			errorMessage: "unable to search API's",
			loggerAction: "get API ID's",
		}
	}
	var subscriptionRequests []apim.SubscriptionReq
	for _, apiID := range apiIDs {
		subReq := apim.SubscriptionReq{
			ApplicationID: svcInstance.ApplicationID,
			APIIdentifier: apiID,
			Tier:          "Unlimited",
		}
		subscriptionRequests = append(subscriptionRequests, subReq)
	}
	subscriptionCreateResp, err := apim.CreateMultipleSubscriptions(subscriptionRequests)
	if err != nil {
		log.Error("unable to create subscriptions", err, logData)
		//revertApplication(svcInstance.ApplicationID, logData)
		return nil, &InternalServerFailureResponseError{
			errorMessage: "unable to create subscriptions ",
			loggerAction: "get API ID's",
			revertApplication: &ConfigApplicationError{
				applicationID: svcInstance.ApplicationID,
				logData:       logData,
			},
		}
	}
	return getSubscriptionList(svcInstance.ID, subscriptionCreateResp), nil
}

func generateApplicationName(svcInstanceID string) string {
	return ApplicationPrefix + svcInstanceID
}

func checkErrorResponse(err error) error {

	baseError := errors.Cause(err)
	switch baseError.(type) {
	case *InternalServerFailureResponseError:
		errStruct := baseError.(*InternalServerFailureResponseError)
		if errStruct.revertApplication != nil {
			revertApplication(errStruct.revertApplication.applicationID, errStruct.revertApplication.logData)
		}
		return apiresponses.NewFailureResponse(errors.New(errStruct.errorMessage), http.StatusInternalServerError, errStruct.loggerAction)
	case *BadRequestFailureResponseError:
		errStruct := baseError.(*BadRequestFailureResponseError)
		return apiresponses.NewFailureResponse(errors.New(errStruct.errorMessage), http.StatusBadRequest, errStruct.loggerAction)
	case *ConfigApplicationError:
		errStruct := baseError.(*ConfigApplicationError)
		revertApplication(errStruct.applicationID, errStruct.logData)
		return errStruct.err
	case *BindingNotExistsError:
		return apiresponses.ErrBindingDoesNotExist
	default:
		return err
	}

}

func (apimBroker *APIM) Provision(ctx context.Context, svcInstanceID string,
	serviceDetails domain.ProvisionDetails, asyncAllowed bool) (domain.ProvisionedServiceSpec, error) {
	if !hasValidSpaceIDAndOrgID(serviceDetails.SpaceGUID, serviceDetails.OrganizationGUID) {
		return domain.ProvisionedServiceSpec{}, invalidParamFailureResponse("check space ID and org ID")
	}

	logData := createCommonLogData(svcInstanceID, serviceDetails.ServiceID, serviceDetails.PlanID)

	svcParams, err := getValidServiceParams(serviceDetails.RawParameters, logData)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, checkErrorResponse(err)
	}

	svcAttributes := &ServiceAtrributes{
		serviceParams:  svcParams,
		organizationID: serviceDetails.OrganizationGUID,
		spaceID:        serviceDetails.SpaceGUID,
	}
	responseStatus, err := checkServiceInstanceWithAttributes(svcInstanceID, svcAttributes, logData)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, checkErrorResponse(err)
	}

	if responseStatus == StatusInstanceAlreadyExists {
		return domain.ProvisionedServiceSpec{
			AlreadyExists: true,
		}, nil
	}

	appMetadata, err := createApplicationAndGenerateKeys(svcInstanceID, logData)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, checkErrorResponse(err)
	}

	parameterHash, err := generateHashForServiceParam(appMetadata.ID, svcAttributes.serviceParams, logData)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, checkErrorResponse(err)
	}

	svcInstance, err := addServiceInstanceToDB(svcInstanceID, appMetadata, svcAttributes.spaceID, svcAttributes.organizationID, parameterHash, logData)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, checkErrorResponse(err)
	}

	err = createAndAddSubscriptionsToDB(svcInstance, svcAttributes.serviceParams.APIs, logData)
	if err != nil {
		deleteServiceInstanceAndLogError(svcInstanceID, logData)
		return domain.ProvisionedServiceSpec{}, checkErrorResponse(err)
	}

	return domain.ProvisionedServiceSpec{
		DashboardURL: appMetadata.DashboardURL,
	}, nil
}

func (apimBroker *APIM) Deprovision(ctx context.Context, svcInstanceID string,
	serviceDetails domain.DeprovisionDetails, asyncAllowed bool) (domain.DeprovisionServiceSpec, error) {
	logData := createCommonLogData(svcInstanceID, serviceDetails.ServiceID, serviceDetails.PlanID)

	svcInstance, err := getServiceInstanceFromDB(svcInstanceID, logData)
	if err != nil {
		return domain.DeprovisionServiceSpec{}, checkErrorResponse(err)
	}
	if svcInstance == nil {
		return domain.DeprovisionServiceSpec{}, apiresponses.ErrInstanceAlreadyExists //check: conflict
	}

	logData.
		Add(LogKeyAppID, svcInstance.ApplicationID).
		Add(LogKeyApplicationName, svcInstance.ApplicationName)

	log.Debug("delete the application", logData)
	err = apim.DeleteApplication(svcInstance.ApplicationID)
	if err != nil {
		log.Error("unable to delete the Application", err, logData)
		return domain.DeprovisionServiceSpec{},
			&InternalServerFailureResponseError{
				errorMessage: ErrMsgUnableDelInstance,
				loggerAction: ErrActionDelAPP,
			}
	}
	log.Debug(DebugMsgDelInstanceFromDB, logData)

	err = deleteInstanceFromDB(&model.ServiceInstance{ID: svcInstanceID}, logData)
	if err != nil {
		return domain.DeprovisionServiceSpec{}, checkErrorResponse(err)
	}

	return domain.DeprovisionServiceSpec{}, nil
}

func unmarshalServiceParams(rawParam json.RawMessage) (ServiceParams, error) {
	var s ServiceParams
	err := json.Unmarshal(rawParam, &s)
	if err != nil {
		return s, err
	}
	return s, nil
}

// getBindFromDB returns the whether Bind exists, initialized Bind struct and any error encountered.
func getBindFromDB(bindingID string, logData *log.Data) (bool, *model.Bind, error) {
	bind := &model.Bind{
		ID: bindingID,
	}
	exists, err := db.Retrieve(bind)
	if err != nil {
		log.Error(ErrMsgUnableToGetBindFromDB, err, logData)
		err := &InternalServerFailureResponseError{
			errorMessage: ErrMsgUnableToGetBindFromDB,
			loggerAction: "get bind from DB",
		}
		return false, nil, errors.Wrap(err, "retrieve bind infomation from database failed")
	}
	return exists, bind, nil
}

// isBindWithSameAttributes returns true of the Bind is already exists and attached with the given instance ID,attributes.
func isBindWithSameAttributes(bind *model.Bind, svcInstanceID string, bindResource *domain.BindResource) bool {
	var isSameAttributes = svcInstanceID == bind.SVCInstanceID
	if !isCreateServiceKey(bindResource) {
		isSameAttributes = isSameAttributes && (bindResource.AppGuid == bind.PlatformAppID)
	}
	return isSameAttributes
}

// Bind method creates a Bind between given Service instance and the App.
func (apimBroker *APIM) Bind(ctx context.Context, svcInstanceID, bindingID string,
	details domain.BindDetails, asyncAllowed bool) (domain.Binding, error) {

	logData := createCommonLogData(svcInstanceID, details.ServiceID, details.PlanID)
	logData.Add(LogKeyBindID, bindingID)

	exists, bind, err := getBindFromDB(bindingID, logData)
	if err != nil {
		return domain.Binding{}, checkErrorResponse(err)
	}
	var isWithSameAttr = false
	if exists {
		isWithSameAttr = isBindWithSameAttributes(bind, svcInstanceID, details.BindResource)
		if !isWithSameAttr {
			return domain.Binding{}, apiresponses.ErrBindingAlreadyExists
		}
	}

	log.Debug("get instance from DB", logData)
	svcInstance, err := getServiceInstanceFromDB(svcInstanceID, logData)
	if err != nil {
		return domain.Binding{}, checkErrorResponse(err)
	}
	if svcInstance == nil {
		return domain.Binding{}, apiresponses.ErrInstanceAlreadyExists //check: conflict
	}

	if isWithSameAttr {
		credentialsMap := credentialsMap(svcInstance.ApplicationName, svcInstance.ConsumerKey, svcInstance.ConsumerSecret)
		return domain.Binding{
			Credentials:   credentialsMap,
			AlreadyExists: true,
		}, nil
	}

	platformAppID := getPlatformAppID(details.BindResource)
	logData.Add(LogKeyPlatformApplicationName, platformAppID)

	bind = &model.Bind{
		ID:            bindingID,
		PlatformAppID: platformAppID,
		SVCInstanceID: svcInstanceID,
	}
	err = storeBind(bind, logData)
	if err != nil {
		return domain.Binding{}, checkErrorResponse(err)
	}
	log.Debug("successfully stored the Bind in the DB", logData)
	credentialsMap := credentialsMap(svcInstance.ApplicationName, svcInstance.ConsumerKey, svcInstance.ConsumerSecret)
	return domain.Binding{
		Credentials: credentialsMap,
	}, nil
}

func isApplicationPlan(planID string) bool {
	return planID == ApplicationPlanID
}

// func generateUUID() (string, error) {
// 	u, err := uuid.NewUUID()
// 	if err != nil {
// 		return "", err
// 	}
// 	return u.String(), nil
// }

// getPlatformAppID returns platform app ID from the given domain.BindResource and any error encountered.
// An UUID is returned if the domain.BindResource is nil(create-service-key flow).
// Error type is apiresponses.FailureResponse.
func getPlatformAppID(b *domain.BindResource) string {
	var cfAppID string
	if isCreateServiceKey(b) {
		// u, err := generateUUID()
		// if err != nil {
		// 	return "", err
		// }
		cfAppID = ""
	} else {
		cfAppID = b.AppGuid
	}
	return cfAppID
}

func revertApplication(appID string, logData *log.Data) {
	err := apim.DeleteApplication(appID)
	if err != nil {
		log.Error("unable to delete application", err, logData)
	}
	log.Debug("Delete Application", logData)
}

// generateKeysForApplication function generates keys for the given Subscription.
// Returns initialized *apim.ApplicationKeyResp and type apiresponses.FailureResponse error if encountered.
func generateKeysForApplication(appID string, logData *log.Data) (*apim.ApplicationKeyResp, error) {
	appKeys, err := apim.GenerateKeys(appID)
	if err != nil {
		log.Error(ErrMsgUnableGenerateKeys, err, logData)
		//revertApplication(appID, logData)
		return appKeys, &InternalServerFailureResponseError{
			errorMessage: ErrMsgUnableGenerateKeys,
			loggerAction: "generate keys for application",
			revertApplication: &ConfigApplicationError{
				applicationID: appID,
				logData:       logData,
			},
		}
	}
	return appKeys, nil
}

// storeBind function stores the given Bind in the database.
// Returns type apiresponses.FailureResponse error if encountered.
func storeBind(b *model.Bind, logData *log.Data) error {
	err := db.Store(b)
	if err != nil {
		log.Error("unable to store bind", err, logData)
		err := &InternalServerFailureResponseError{
			errorMessage: "unable to store Bind in DB",
			loggerAction: "store Bind in DB",
		}
		return errors.Wrap(err, "unable to store bind in the database")
	}
	return nil
}

// createServiceKey check whether the command is a "create-service-key".
// BindResources or BindResource.AppGuid is nil only if the it is a "create-service-key" command.
func isCreateServiceKey(b *domain.BindResource) bool {
	return b == nil || b.AppGuid == ""
}

func credentialsMap(appName, consumerKey, consumerSecret string) map[string]interface{} {
	return map[string]interface{}{
		"ApplicationName": appName,
		"ConsumerKey":     consumerKey,
		"ConsumerSecret":  consumerSecret,
	}
}

// Unbind deletes the Bind from database and returns domain.UnbindSpec struct and any error encountered.
func (apimBroker *APIM) Unbind(ctx context.Context, svcInstanceID, bindingID string,
	details domain.UnbindDetails, asyncAllowed bool) (domain.UnbindSpec, error) {

	logData := createCommonLogData(svcInstanceID, details.ServiceID, details.PlanID)

	domainUnbindSpec := domain.UnbindSpec{}
	if !isApplicationPlan(details.PlanID) {
		log.Error(ErrMsgInvalidPlanID, ErrInvalidSVCPlan, logData)
		return domainUnbindSpec, checkErrorResponse(errors.Wrap(invalidPlanFailureResponse("unbinding"), "plan id does not match"))
	}
	bind, err := getBindForDeletion(bindingID, logData)
	if err != nil {
		return domainUnbindSpec, checkErrorResponse(err)
	}
	logData.Add("cf-app-id", bind.PlatformAppID)

	err = deleteBindFromDB(bind, logData)

	if err != nil {
		return domainUnbindSpec, checkErrorResponse(err)
	}

	return domainUnbindSpec, nil
}

func deleteBindFromDB(bind *model.Bind, logData *log.Data) error {
	err := db.Delete(bind)
	if err != nil {
		log.Error("unable to delete the bind from the database", err, logData)
		err := &InternalServerFailureResponseError{
			errorMessage: "unable to unbind",
			loggerAction: "delete Bind",
		}
		return errors.Wrap(err, "unable to delete bind from database")
	}
	return nil
}

// getBindForDeletion checks whether the given BindID in the database. If not exists, an error type apiresponses.FailureResponse
// is returned.
func getBindForDeletion(bindingID string, logData *log.Data) (*model.Bind, error) {
	exists, bind, err := getBindFromDB(bindingID, logData)
	if err != nil {
		return nil, err
	}
	// Bind not exists
	if !exists {
		return nil, &BindingNotExistsError{}
	}
	return bind, nil
}

// LastOperation ...
// If the broker provisions asynchronously, the Cloud Controller will poll this endpoint
// for the status of the provisioning operation.
func (apimBroker *APIM) LastOperation(ctx context.Context, svcInstanceID string,
	details domain.PollDetails) (domain.LastOperation, error) {
	return domain.LastOperation{}, errors.New("not supported")
}

func updateServiceForAddedAPIs(params []API, svcInstance *model.ServiceInstance, logData *log.Data) ([]API, error) {

	existingAPIs, err := getExistingAPIsForAppID(svcInstance.ApplicationID, logData)
	if err != nil {
		return nil, err
	}

	addedAPIs := getAddedAPIs(existingAPIs, params, logData) //updatesvcforaddedapis
	if addedAPIs == nil {
		log.Debug("No new APIs found", logData)
		return addedAPIs, nil
	}

	addedSubscriptions, err := createSubscriptions(svcInstance, addedAPIs, logData)
	if err != nil {
		return nil, err
	}

	err = storeSubscriptions(addedSubscriptions)
	if err != nil {
		return nil, err
	}
	return existingAPIs, nil
}

func updateServiceForRemovedAPIs(svcInstance *model.ServiceInstance, instanceID string, existingAPIs []API, paramAPIs []API, logData *log.Data) error {

	removeSubscriptionIDs, err := getRemovedSubscriptionsIDs(svcInstance.ApplicationID, existingAPIs, paramAPIs, logData) //updatesvcforremovedapis
	if err != nil {
		return err
	}

	err = removeSubscriptions(removeSubscriptionIDs, instanceID)
	if err != nil {
		return err
	}

	return nil
}

func restorePreviousState(svcInstance *model.ServiceInstance, apis []API, instanceID string, logData *log.Data) {
	log.Debug("Started restoring previous state of the service", logData)
	existingAPIs, err := updateServiceForAddedAPIs(apis, svcInstance, logData)
	if err != nil {
		log.Error("unable to restore previous state", err, logData)
	}

	err = updateServiceForRemovedAPIs(svcInstance, instanceID, existingAPIs, apis, logData)
	if err != nil {
		log.Error("unable to restore previous state", err, logData)
	}
	log.Debug("previous service state restored", logData)

}

func (apimBroker *APIM) Update(cxt context.Context, svcInstanceID string,
	serviceDetails domain.UpdateDetails, asyncAllowed bool) (domain.UpdateServiceSpec, error) {

	logData := createCommonLogData(svcInstanceID, serviceDetails.ServiceID, serviceDetails.PlanID)
	log.Debug("update service instance", logData)

	svcParams, err := getValidServiceParams(serviceDetails.RawParameters, logData)
	if err != nil {
		return domain.UpdateServiceSpec{}, checkErrorResponse(err)
	}
	// getServiceInstanceFromDB returns 410 if the instance doesn't exists.
	// In this case status code 410 is interpreted as a failure.
	svcInstance, err := getServiceInstanceFromDB(svcInstanceID, logData)
	if err != nil {
		return domain.UpdateServiceSpec{}, checkErrorResponse(err)
	}
	if svcInstance == nil {
		return domain.UpdateServiceSpec{}, apiresponses.ErrInstanceAlreadyExists
	}

	existingAPIs, err := updateServiceForAddedAPIs(svcParams.APIs, svcInstance, logData)
	if err != nil {
		restorePreviousState(svcInstance, existingAPIs, svcInstanceID, logData)
		return domain.UpdateServiceSpec{}, checkErrorResponse(err)
	}

	err = updateServiceForRemovedAPIs(svcInstance, svcInstanceID, existingAPIs, svcParams.APIs, logData)
	if err != nil {
		restorePreviousState(svcInstance, existingAPIs, svcInstanceID, logData)
		return domain.UpdateServiceSpec{}, checkErrorResponse(err)
	}

	log.Debug("Instace successfully updated", logData)

	return domain.UpdateServiceSpec{}, err
}
