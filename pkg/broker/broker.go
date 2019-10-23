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

// Package broker holds the implementation of brokerapi.ServiceBroker interface.
package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
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
	"net/http"
	"strconv"
	"strings"
)

const (
	LogKeyAppID                   = "application-id"
	LogKeyServiceID               = "service-id"
	LogKeyPlanID                  = "plan-id"
	LogKeyInstanceID              = "instance-id"
	LogKeyBindID                  = "bind-id"
	LogKeyApplicationName         = "application-name"
	LogKeyPlatformApplicationName = "platform-application-name"
	ApplicationDashboardURL       = "application-dashboard-url"
	ErrMsgUnableToStoreInstance   = "unable to store service instance in database"
	ErrActionStoreInstance        = "store service instance in DB"
	ErrActionDelAPP               = "delete Application"
	ErrActionDelInstanceFromDB    = "delete service instance from DB"
	ErrActionCreateAPIMResource   = "creating API-M resource"
	ErrActionUpdateAPIMResource   = "update API-M resource"
	ErrMsgUnableDelInstance       = "unable to delete service instance"
	ErrMsgUnableToGetBindFromDB   = "unable to retrieve Bind from the database"
	ErrMsgUnableToGenInputSchema  = "unable to generate %s plan input Schema"
	ErrMsgInvalidPlanID           = "invalid plan id"
	ErrMsgUnableGenerateKeys      = "unable generate keys for application"
	ServiceID                     = "460F28F9-4D05-4889-970A-6BF5FB7D3CF8"
	ServiceName                   = "wso2apim-service"
	ServiceDescription            = "Manages WSO2 API Manager artifacts"
	ApplicationPlanID             = "00e851cd-ce8b-43eb-bc27-ac4d4fbb3204"
	ApplicationPlanName           = "app"
	ApplicationPlanDescription    = "Creates an Application with a set of subscription for a given set of APIs in WSO2 API Manager"
	DebugMsgDelInstanceFromDB     = "delete instance from DB"
	ApplicationPreFix             = "ServiceBroker_"
)

var (
	ErrNotSupported             = errors.New("not supported")
	ErrInvalidSVCPlan           = errors.New("invalid service or getServices")
	applicationPlanBindable     = true
	appPlanInputParameterSchema map[string]interface{}
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

// Init method initialize the API-M broker. If there is an error it will cause a panic.
func (apimBroker *APIM) Init() {
	var err error
	appPlanInputParameterSchema, err = utils.GetJSONSchema(apim.AppPlanInputParameterSchemaRaw)
	if err != nil {
		log.HandleErrorAndExit(fmt.Sprintf(ErrMsgUnableToGenInputSchema, "API"), err)
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

func (apimBroker *APIM) Provision(ctx context.Context, svcInstanceID string,
	serviceDetails domain.ProvisionDetails, asyncAllowed bool) (domain.ProvisionedServiceSpec, error) {
	if !hasValidSpaceOrgID(serviceDetails.SpaceGUID, serviceDetails.OrganizationGUID) {
		return domain.ProvisionedServiceSpec{}, invalidParamFailureResponse("check space ID and org ID")
	}

	logData := createCommonLogData(svcInstanceID, serviceDetails.ServiceID, serviceDetails.PlanID)

	svcParam, err := getValidServiceParams(serviceDetails.RawParameters)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}

	provisionedServiceSpec, err := checkInstanceWithAttributes(svcInstanceID, svcParam, serviceDetails.SpaceGUID, serviceDetails.OrganizationGUID)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}
	if provisionedServiceSpec.AlreadyExists {
		return provisionedServiceSpec, nil
	}

	appName, err := generateApplicationName()
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}

	logData.Add(LogKeyApplicationName, appName)
	appID, appDashboardURL, err := createApplication(appName)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}
	logData.Add(LogKeyAppID, appID).
		Add(ApplicationDashboardURL, appDashboardURL)

	keys, err := generateKeysForApplication(appID)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}

	parameterHash, err := generateHashAPIs(svcParam)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}

	svcInstance := &model.ServiceInstance{
		ID:              svcInstanceID,
		ApplicationID:   appID,
		ApplicationName: appName,
		SpaceID:         serviceDetails.SpaceGUID,
		OrgID:           serviceDetails.OrganizationGUID,
		ConsumerKey:     keys.ConsumerKey,
		ConsumerSecret:  keys.ConsumerSecret,
		ParameterHash:   parameterHash,
	}
	err = storeServiceInstance(svcInstance)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}

	subscriptions, err := createSubscriptions(svcInstance, svcParam.APIs)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}
	err = storeSubscriptions(subscriptions)
	if err != nil {
		log.Error("unable to store subscriptions", err, logData)
		handleStoreSubscriptionError(appID, svcInstanceID)
		return domain.ProvisionedServiceSpec{}, err
	}

	return domain.ProvisionedServiceSpec{
		DashboardURL: appDashboardURL,
	}, nil
}

func handleStoreSubscriptionError(appID string, svcInstanceID string) {
	logData := log.NewData().
		Add(LogKeyAppID, appID).
		Add(LogKeyInstanceID, svcInstanceID)
	log.Debug("deleting the application", logData)
	deleteApplicationAndLogError(appID)
	log.Debug("deleting the instance from database", logData)
	deleteServiceInstanceAndLogError(svcInstanceID)
}

func deleteServiceInstanceAndLogError(svcInstanceID string) {
	logData := log.NewData().
		Add(LogKeyInstanceID, svcInstanceID)
	err := db.Delete(&model.ServiceInstance{
		ID: svcInstanceID,
	})
	if err != nil {
		log.Error(ErrMsgUnableDelInstance, err, logData)
	}
}

func generateHashAPIs(svcParam ServiceParams) (string, error) {
	generatedHash, err := hashstructure.Hash(svcParam, nil)
	if err != nil {
		return "", internalServerFailureResponse("unable to generate hash value for service parameters", "generate hash for service parameter")
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
		return internalServerFailureResponse("unable to store subscriptions in the database", "store subscriptions in the database")
	}
	return nil
}

func getSubscriptionList(svcInstanceID string, subsResponses []apim.SubscriptionResp) []model.Subscription {
	var subscriptions []model.Subscription
	for _, subsResponse := range subsResponses {
		apiIdentifier := strings.Split(subsResponse.APIIdentifier, "-")
		subscriptions = append(subscriptions, model.Subscription{
			ID:            subsResponse.SubscriptionID,
			ApplicationID: subsResponse.ApplicationID,
			User:          apiIdentifier[0],
			APIName:       apiIdentifier[1],
			APIVersion:    apiIdentifier[2],
			SVCInstanceID: svcInstanceID,
		})
	}
	return subscriptions
}

// storeServiceInstance stores service instance in the database returns an error type apiresponses.FailureResponse.
func storeServiceInstance(i *model.ServiceInstance) error {
	logData := log.NewData().
		Add(LogKeyAppID, i.ApplicationID).
		Add(LogKeyInstanceID, i.ID)
	err := db.Store(i)
	if err != nil {
		log.Error(ErrMsgUnableToStoreInstance, err, logData)
		log.Debug("deleting the application", logData)
		deleteApplicationAndLogError(i.ApplicationID)
		return internalServerFailureResponse(ErrMsgUnableToStoreInstance, ErrActionStoreInstance)
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

func createSubscriptions(svcInstance *model.ServiceInstance, apis []API) ([]model.Subscription, error) {
	logData := log.NewData().Add(LogKeyAppID, svcInstance.ApplicationID)

	apiIDs, err := getAPIIDs(apis)
	if err != nil {
		log.Error("unable to get API ids", err, logData)
		return nil, internalServerFailureResponse("unable to search API's", "get API ID's")
	}
	var subsCreateRequests []apim.SubscriptionReq
	for _, apiID := range apiIDs {
		subsCreateRequests = append(subsCreateRequests, apim.SubscriptionReq{
			ApplicationID: svcInstance.ApplicationID,
			APIIdentifier: apiID,
			Tier:          "Unlimited",
		})
	}
	subscriptionCreateResp, err := apim.CreateMultipleSubscription(subsCreateRequests)
	if err != nil {
		log.Error("unable to create subscriptions", err, logData)
		deleteApplicationAndLogError(svcInstance.ApplicationID)
		return nil, internalServerFailureResponse("unable to create subscriptions ", "get API ID's")
	}
	return getSubscriptionList(svcInstance.ID, subscriptionCreateResp), nil
}

func generateApplicationName() (string, error) {
	u, err := generateUUID()
	if err != nil {
		log.Error("unable to generate an Subscription name", err, nil)
		return "", internalServerFailureResponse("unable to generate an Subscription name", "generate application name")
	}
	return ApplicationPreFix + u, nil
}

func hasValidSpaceOrgID(spaceID string, orgID string) bool {
	return spaceID != "" && orgID != ""
}

func (apimBroker *APIM) Deprovision(ctx context.Context, svcInstanceID string,
	serviceDetails domain.DeprovisionDetails, asyncAllowed bool) (domain.DeprovisionServiceSpec, error) {
	logData := createCommonLogData(svcInstanceID, serviceDetails.ServiceID, serviceDetails.PlanID)

	svcInstance, err := getServiceInstanceFromDB(svcInstanceID)
	if err != nil {
		return domain.DeprovisionServiceSpec{}, err
	}

	logData.
		Add(LogKeyAppID, svcInstance.ApplicationID).
		Add(LogKeyApplicationName, svcInstance.ApplicationName)

	log.Debug("delete the application", logData)
	err = apim.DeleteApplication(svcInstance.ApplicationID)
	if err != nil {
		log.Error("unable to delete the Application", err, logData)
		return domain.DeprovisionServiceSpec{}, internalServerFailureResponse(ErrMsgUnableDelInstance, ErrActionDelAPP)
	}
	log.Debug(DebugMsgDelInstanceFromDB, logData)
	return deleteInstanceFromDB(&model.ServiceInstance{ID: svcInstanceID})
}

func getValidServiceParams(rawParam json.RawMessage) (ServiceParams, error) {
	s, err := toServiceParams(rawParam)
	if err != nil {
		log.Error("unable to parse parameters", err, nil)
		return s, invalidParamFailureResponse("parse ServiceParams")
	}

	if len(s.APIs) == 0 {
		log.Error("unable to parse parameters", err, nil)
		return s, invalidParamFailureResponse("parse ServiceParams")
	}

	return s, nil
}

func toServiceParams(rawParam json.RawMessage) (ServiceParams, error) {
	var s ServiceParams
	err := json.Unmarshal(rawParam, &s)
	if err != nil {
		return s, err
	}
	return s, nil
}

// getBindFromDB returns the whether Bind exists, initialized Bind struct and any error encountered.
func getBindFromDB(bindingID string) (bool, *model.Bind, error) {
	logData := log.NewData().Add(LogKeyBindID, bindingID)
	bind := &model.Bind{
		ID: bindingID,
	}
	exists, err := db.Retrieve(bind)
	if err != nil {
		log.Error(ErrMsgUnableToGetBindFromDB, err, logData)
		return false, nil, internalServerFailureResponse(ErrMsgUnableToGetBindFromDB, "get bind from DB")
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

	exists, bind, err := getBindFromDB(bindingID)
	if err != nil {
		return domain.Binding{}, err
	}
	var isWithSameAttr = false
	if exists {
		isWithSameAttr = isBindWithSameAttributes(bind, svcInstanceID, details.BindResource)
		if !isWithSameAttr {
			return domain.Binding{}, apiresponses.ErrBindingAlreadyExists
		}
	}

	log.Debug("get instance from DB", logData)
	svcInstance, err := getServiceInstanceFromDB(svcInstanceID)
	if err != nil {
		return domain.Binding{}, err
	}

	if isWithSameAttr {
		credentialsMap := credentialsMap(svcInstance.ApplicationName, svcInstance.ConsumerKey, svcInstance.ConsumerSecret)
		return domain.Binding{
			Credentials:   credentialsMap,
			AlreadyExists: true,
		}, nil
	}

	createServiceKey := isCreateServiceKey(details.BindResource)
	logData.Add("is-create-service-key", createServiceKey)

	platformAppID, err := getPlatformAppID(details.BindResource)
	if err != nil {
		log.Error("unable to generate UUID for CF app", err, nil)
		return domain.Binding{}, internalServerFailureResponse("unable to generate UUID for CF app", "generate UUID for CF app name")
	}
	logData.Add(LogKeyPlatformApplicationName, platformAppID)

	bind = &model.Bind{
		ID:                 bindingID,
		PlatformAppID:      platformAppID,
		SVCInstanceID:      svcInstanceID,
		IsCreateServiceKey: createServiceKey,
	}
	err = storeBind(bind)
	if err != nil {
		return domain.Binding{}, err
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

func generateUUID() (string, error) {
	u, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// getPlatformAppID returns platform app ID from the given domain.BindResource and any error encountered.
// An UUID is returned if the domain.BindResource is nil(create-service-key flow).
// Error type is apiresponses.FailureResponse.
func getPlatformAppID(b *domain.BindResource) (string, error) {
	var cfAppID string
	if isCreateServiceKey(b) {
		u, err := generateUUID()
		if err != nil {
			return "", err
		}
		cfAppID = u
	} else {
		cfAppID = b.AppGuid
	}
	return cfAppID, nil
}

func deleteApplicationAndLogError(appID string) {
	logData := log.NewData().Add(LogKeyAppID, appID)
	err := apim.DeleteApplication(appID)
	if err != nil {
		log.Error("unable to delete application", err, logData)
	}
}

// generateKeysForApplication function generates keys for the given Subscription.
// Returns initialized *apim.ApplicationKeyResp and type apiresponses.FailureResponse error if encountered.
func generateKeysForApplication(appID string) (*apim.ApplicationKeyResp, error) {
	logData := log.NewData().Add(LogKeyAppID, appID)
	appKeys, err := apim.GenerateKeys(appID)
	if err != nil {
		log.Error(ErrMsgUnableGenerateKeys, err, logData)
		log.Debug("deleting the application", logData)
		deleteApplicationAndLogError(appID)
		return appKeys, internalServerFailureResponse(ErrMsgUnableGenerateKeys,
			"generate keys for application")
	}
	return appKeys, nil
}

// storeBind function stores the given Bind in the database.
// Returns type apiresponses.FailureResponse error if encountered.
func storeBind(b *model.Bind) error {
	logData := log.NewData().Add(LogKeyBindID, b.ID)
	err := db.Store(b)
	if err != nil {
		log.Error("unable to store bind", err, logData)
		return internalServerFailureResponse("unable to store Bind in DB", "store Bind in DB")
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
		return domainUnbindSpec, invalidPlanFailureResponse("unbinding")
	}
	bind, err := validateBindIDForDel(bindingID)
	if err != nil {
		return domainUnbindSpec, err
	}
	logData.Add("cf-app-id", bind.PlatformAppID)
	errDB := db.Delete(bind)
	if errDB != nil {
		log.Error("unable to delete the bind from the database", errDB, logData)
		return domainUnbindSpec, internalServerFailureResponse("unable to unbind", "delete Bind")
	}
	return domainUnbindSpec, nil
}

// validateBindIDForDel checks whether the given BindID in the database. If not exists, an error type apiresponses.FailureResponse
// is returned.
func validateBindIDForDel(bindingID string) (*model.Bind, error) {
	exists, bind, err := getBindFromDB(bindingID)
	if err != nil {
		return nil, err
	}
	// Bind not exists
	if !exists {
		return nil, apiresponses.ErrBindingDoesNotExist
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

func (apimBroker *APIM) Update(cxt context.Context, svcInstanceID string,
	serviceDetails domain.UpdateDetails, asyncAllowed bool) (domain.UpdateServiceSpec, error) {

	logData := createCommonLogData(svcInstanceID, serviceDetails.ServiceID, serviceDetails.PlanID)
	log.Debug("update service instance", logData)

	svcParam, err := getValidServiceParams(serviceDetails.RawParameters)
	if err != nil {
		return domain.UpdateServiceSpec{}, err
	}
	// getServiceInstanceFromDB returns 410 if the instance doesn't exists.
	// In this case status code 410 is interpreted as a failure.
	svcInstance, err := getServiceInstanceFromDB(svcInstanceID)
	if err != nil {
		return domain.UpdateServiceSpec{}, err
	}

	existingAPIs, err := getExistingAPIsForAppID(svcInstance.ApplicationID)

	addedAPIs := getAddedAPIs(existingAPIs, svcParam.APIs)
	//subscriptions
	_, err = createSubscriptions(svcInstance, addedAPIs)
	if err != nil {
		return domain.UpdateServiceSpec{}, err
	}
	//removedSubsIDs
	_, err = getRemovedSubscriptionsIDs(svcInstance.ApplicationID, existingAPIs, svcParam.APIs)
	if err != nil {
		return domain.UpdateServiceSpec{}, err
	}

	return domain.UpdateServiceSpec{}, invalidPlanFailureResponse("updating")
}

func getRemovedSubscriptionsIDs(applicationID string, existingAPIs, newAPIs []API) ([]string, error) {
	rmAPIs := getRemovedAPIs(existingAPIs, newAPIs)
	var rmSubsIDs []string
	for _, rAPI := range rmAPIs {
		rSub, err := getSubscriptionForAppAndAPI(applicationID, rAPI)
		if err != nil {
			return nil, err
		}
		rmSubsIDs = append(rmSubsIDs, rSub.ID)
	}
	return rmSubsIDs, nil
}

func getRemovedAPIs(existingAPIs, newAPIs []API) []API {
	var removedAPIs []API
	for _, api := range existingAPIs {
		if !isArrayContainAPI(newAPIs, api) {
			removedAPIs = append(removedAPIs, api)
		}
	}
	return removedAPIs
}

func getAddedAPIs(existingAPIs, newAPIs []API) []API {
	var addedAPIs []API
	for _, api := range newAPIs {
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
								Parameters: map[string]interface{}{},
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
func createApplication(appName string) (string, string, error) {
	ld := log.NewData().Add(LogKeyApplicationName, appName)
	req := &apim.ApplicationCreateReq{
		Name:           appName,
		ThrottlingTier: "Unlimited",
	}
	appID, err := apim.CreateApplication(req)
	if err != nil {
		log.Error("unable to create application", err, ld)
		return "", "", handleAPIMResourceCreateError(err, appName)
	}
	dashboardURL := apim.GetAPPDashboardURL(appName)
	return appID, dashboardURL, nil
}

// handleAPIMResourceCreateError handles the API-M resource creation error. Returns an error type apiresponses.FailureResponse.
func handleAPIMResourceCreateError(e error, resourceName string) error {
	invokeErr, ok := e.(*client.InvokeError)
	if ok && invokeErr.StatusCode == http.StatusConflict {
		return internalServerFailureResponse(fmt.Sprintf("API-M resource %s already exist", resourceName), ErrActionCreateAPIMResource)
	}
	return internalServerFailureResponse("unable to create the API-M resource", ErrActionCreateAPIMResource)
}

// getServiceInstanceFromDB function check whether the given instance already exists. If the given instance exists then an initialized instance and
// if the given instance doesn't exist or unable to retrieve it from database, an error type apiresponses.FailureResponse is returned.
func getServiceInstanceFromDB(svcInstanceID string) (*model.ServiceInstance, error) {
	ld := log.NewData().Add(LogKeyInstanceID, svcInstanceID)
	instance := &model.ServiceInstance{
		ID: svcInstanceID,
	}
	exists, err := db.Retrieve(instance)
	if err != nil {
		log.Error("unable to retrieve the service instance from database", err, ld)
		return nil, internalServerFailureResponse("unable to get Service instance from the database", "get instance from the database")
	}
	if !exists {
		log.Error("instance doesn't exists", err, ld)
		return nil, apiresponses.ErrInstanceDoesNotExist
	}
	return instance, nil
}

// deleteInstanceFromDB function deletes the given instance from database. A domain.DeprovisionServiceSpec{} and an error type apiresponses.FailureResponse is returned.
func deleteInstanceFromDB(i *model.ServiceInstance) (domain.DeprovisionServiceSpec, error) {
	ld := log.NewData().Add(LogKeyInstanceID, i.ID)
	err := db.Delete(i)
	if err != nil {
		log.Error("unable to delete the instance from the database", err, ld)
		return domain.DeprovisionServiceSpec{}, internalServerFailureResponse(ErrMsgUnableDelInstance, ErrActionDelInstanceFromDB)
	}
	return domain.DeprovisionServiceSpec{}, nil
}

// handleAPIMResourceUpdateError handles the API-M resource update errors. Returns an error type apiresponses.FailureResponse.
func handleAPIMResourceUpdateError(err error, resourceName string) error {
	e, ok := err.(*client.InvokeError)
	if ok && e.StatusCode == http.StatusNotFound {
		return internalServerFailureResponse(fmt.Sprintf("API-M resource %s not found !", resourceName), ErrActionUpdateAPIMResource)
	}
	return internalServerFailureResponse("unable to update the API-M resource ", ErrActionUpdateAPIMResource)
}

func internalServerFailureResponse(m, a string) error {
	return apiresponses.NewFailureResponse(errors.New(m), http.StatusInternalServerError, a)
}

func invalidPlanFailureResponse(a string) error {
	return badRequestFailureResponse(a, "invalid getServices or Service")
}

func badRequestFailureResponse(action, msg string) error {
	return apiresponses.NewFailureResponse(errors.New(msg),
		http.StatusBadRequest, action)
}

func invalidParamFailureResponse(a string) error {
	return badRequestFailureResponse(a, "invalid parameters")
}

// checkInstanceWithAttributes function returns domain.ProvisionedServiceSpec and error type apiresponses.FailureResponse.
// If there is a instance with the same attributes the domain.ProvisionedServiceSpec.AlreadyExists will be set to "true".
// and if the instance exist with different attributes apiresponses.ErrInstanceAlreadyExists is returned.
func checkInstanceWithAttributes(svcInstanceID string, svcParam ServiceParams, spaceID, orgID string) (domain.ProvisionedServiceSpec, error) {
	ld := createCommonLogData(svcInstanceID, spaceID, orgID)
	i := &model.ServiceInstance{
		ID: svcInstanceID,
	}

	svcInstanceExists, err := db.Retrieve(i)
	if err != nil {
		log.Error("unable to get instance from DB", err, ld)
		return domain.ProvisionedServiceSpec{}, internalServerFailureResponse("unable to query database",
			"get the instance from DB")
	}
	if svcInstanceExists {
		parameterHash, err := generateHashAPIs(svcParam)
		if err != nil {
			return domain.ProvisionedServiceSpec{}, err
		}
		if (i.ParameterHash == parameterHash) && (spaceID == i.SpaceID) && (orgID == i.OrgID) {
			existingAPIs, err := getExistingAPIsForAppID(i.ApplicationID)
			if err != nil {
				return domain.ProvisionedServiceSpec{}, err
			}
			// If existing parameter hash and the new parameter has matches then check each field again to avoid hash collision.
			if isSameAPIs(existingAPIs, svcParam.APIs) {
				return domain.ProvisionedServiceSpec{
					AlreadyExists: true,
				}, nil
			}
		}
		return domain.ProvisionedServiceSpec{}, apiresponses.ErrInstanceAlreadyExists
	}
	return domain.ProvisionedServiceSpec{}, nil
}

func getSubscriptionsForAppID(applicationID string) ([]model.Subscription, error) {
	ld := log.NewData().Add(LogKeyAppID, applicationID)
	subscription := &model.Subscription{
		ApplicationID: applicationID,
	}
	var subscriptionsList []model.Subscription
	hasSubscriptions, err := db.RetrieveList(subscription, &subscriptionsList)
	if err != nil {
		log.Error("unable to get subscription from DB", err, ld)
		return nil, internalServerFailureResponse("unable to query database",
			"get the subscription from DB")
	}
	if !hasSubscriptions {
		log.Error("no subscriptions are available", err, ld)
		return nil, internalServerFailureResponse("no subscriptions are available",
			"get the subscription from DB")
	}
	return subscriptionsList, nil
}

func getSubscriptionForAppAndAPI(applicationID string, api API) (*model.Subscription, error) {
	ld := log.NewData().Add(LogKeyAppID, applicationID)
	subscription := &model.Subscription{
		ApplicationID: applicationID,
		APIName:       api.Name,
		APIVersion:    api.Version,
	}
	hasSubscription, err := db.Retrieve(subscription)
	if err != nil {
		log.Error("unable to get subscription from DB", err, ld)
		return nil, internalServerFailureResponse("unable to query database",
			"get the subscription from DB")
	}
	if !hasSubscription {
		log.Error("no subscription is available", err, ld)
		//ToDO check err msg
		return nil, internalServerFailureResponse("no subscription is available",
			"get the subscription from DB")
	}
	return subscription, nil
}

func getExistingAPIsForAppID(applicationID string) ([]API, error) {
	subscriptionsList, err := getSubscriptionsForAppID(applicationID)
	if err != nil {
		return nil, err
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

func isSameAPIs(existingAPIs, newAPIs []API) bool {
	if len(existingAPIs) != len(newAPIs) {
		return false
	}
	for _, existingAPI := range existingAPIs {
		if !isArrayContainAPI(newAPIs, existingAPI) {
			return false
		}
	}
	return true
}

func isArrayContainAPI(apis []API, api API) bool {
	for _, a := range apis {
		if a.Name == api.Name && a.Version == api.Version {
			return true
		}
	}
	return false
}
