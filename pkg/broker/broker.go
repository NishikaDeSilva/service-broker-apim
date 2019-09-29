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
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/apim"
	"github.com/wso2/service-broker-apim/pkg/client"
	"github.com/wso2/service-broker-apim/pkg/db"
	"github.com/wso2/service-broker-apim/pkg/log"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"net/http"
)

const (
	LogKeyAPIName                   = "api-name"
	LogKeyAPIID                     = "api-id"
	LogKeyAppID                     = "app-id"
	LogKeyAPPName                   = "app-name"
	LogKeySubsID                    = "subs-id"
	LogKeyServiceID                 = "service-id"
	LogKeyPlanID                    = "plan-id"
	LogKeyInstanceID                = "instance-id"
	LogKeyBindID                    = "bind-id"
	LogKeyApplicationName           = "application-name"
	ErrMsgUnableToStoreInstance     = "unable to store instance in DB"
	ErrActionStoreInstance          = "store instance in DB"
	ErrActionDelAPI                 = "delete API"
	ErrActionDelAPP                 = "delete Application"
	ErrActionDelSubs                = "delete Subscription"
	ErrActionDelInstanceFromDB      = "delete service instance from DB"
	ErrActionCreateAPIMResource     = "creating API-M resource"
	ErrActionParseSubscriptionParam = "parse Subscription parameters"
	ErrActionParseAppParam          = "parse Application parameters"
	ErrActionUpdateAPIMResource     = "update API-M resource"
	ErrMsgUnableDelInstance         = "unable to delete service instance"
	ErrMsgUnableToGetBindFromDB     = "unable to retrieve Bind from the database"
	ErrMsgFailedToCleanUp           = "failed to cleanup, unable to delete the %s"
	ServiceID                       = "460F28F9-4D05-4889-970A-6BF5FB7D3CF8"
	ServiceName                     = "wso2apim-service"
	ServiceDescription              = "WSO2 API Manager Services"
	APIPlanID                       = "4CA4F2AF-EADF-4E76-A5CA-732FBC625593"
	APIPlanName                     = "api"
	APIPlanDescription              = "Create an API in WSO2 API Manager"
	ApplicationPlanID               = "00e851cd-ce8b-43eb-bc27-ac4d4fbb3204"
	ApplicationPlanName             = "app"
	ApplicationPlanDescription      = "Create an Application in WSO2 API Manager"
	SubscriptionPlanID              = "00e851cd-ce8b-43eb-bc27-ac4d4fbb3298"
	SubscriptionPlanName            = "subs"
	SubscriptionPlanDescription     = "Create a Subscription in WSO2 API Manager"
	DebugMsgDelInstanceFromDB       = "delete instance from DB"
)

var (
	ErrNotSupported        = errors.New("not supported")
	ErrInvalidSVCPlan      = errors.New("invalid service or services")
	ErrInvalidRawParameter = errors.New("invalid raw parameter")
)

// APIM struct implements the interface brokerapi.ServiceBroker.
type APIM struct {
}

// Services returns the services offered by this broker.
func (apimBroker *APIM) Services(ctx context.Context) ([]domain.Service, error) {
	return services(), nil
}

func createCommonLogData(instanceID, serviceID, planID string) *log.Data {
	return log.NewData().
		Add(LogKeyServiceID, serviceID).
		Add(LogKeyPlanID, planID).
		Add(LogKeyInstanceID, instanceID)
}

func (apimBroker *APIM) Provision(ctx context.Context, instanceID string,
	serviceDetails domain.ProvisionDetails, asyncAllowed bool) (domain.ProvisionedServiceSpec, error) {
	if !hasValidSpaceOrgID(serviceDetails.SpaceGUID, serviceDetails.OrganizationGUID) {
		return domain.ProvisionedServiceSpec{}, invalidParamFailureResponse("check space ID and org ID")
	}
	if isAPIPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return createAPIServiceInstance(instanceID, serviceDetails)
	} else if isApplicationPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return createAppServiceInstance(instanceID, serviceDetails)
	} else if isSubscriptionPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return createSubscriptionServiceInstance(instanceID, serviceDetails)
	} else {
		logData := createCommonLogData(instanceID, serviceDetails.ServiceID, serviceDetails.PlanID)
		log.Error("invalid instance id or plan id", ErrInvalidSVCPlan, logData)
		return domain.ProvisionedServiceSpec{}, invalidServiceOrPlanFailureResponse("provisioning")
	}
}

func hasValidSpaceOrgID(spaceID string, orgID string) bool {
	return spaceID != "" && orgID != ""
}

func (apimBroker *APIM) Deprovision(ctx context.Context, instanceID string,
	serviceDetails domain.DeprovisionDetails, asyncAllowed bool) (domain.DeprovisionServiceSpec, error) {

	if isAPIPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return deleteAPIService(instanceID, serviceDetails.ServiceID, serviceDetails.PlanID)
	} else if isApplicationPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return deleteAppService(instanceID, serviceDetails.ServiceID, serviceDetails.PlanID)
	} else if isSubscriptionPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return deleteSubscriptionService(instanceID, serviceDetails.ServiceID, serviceDetails.PlanID)
	} else {
		logData := createCommonLogData(instanceID, serviceDetails.ServiceID, serviceDetails.PlanID)
		log.Error("invalid instance id or plan id", ErrInvalidSVCPlan, logData)
		return domain.DeprovisionServiceSpec{}, invalidServiceOrPlanFailureResponse("provisioning")
	}
}

// getBindFromDB returns the whether Bind exists, initialized Bind struct and any error encountered.
func getBindFromDB(bindingID string) (bool, *db.Bind, error) {
	logData := log.NewData().Add(LogKeyBindID, bindingID)
	bind := &db.Bind{
		ID: bindingID,
	}
	exists, err := db.Retrieve(bind)
	if err != nil {
		log.Error(ErrMsgUnableToGetBindFromDB, err, logData)
		return false, nil, internalServerFailureResponse(ErrMsgUnableToGetBindFromDB, "get bind from DB")
	}
	return exists, bind, nil
}

// isBindWithAttributes returns true of the Bind is already exists and attached with the given instance ID,attributes.
func isBindWithAttributes(bind *db.Bind, instanceID string, bindResource *domain.BindResource) bool {
	var isSameAttributes = instanceID == bind.InstanceID
	if !isCreateServiceKey(bindResource) {
		isSameAttributes = isSameAttributes && (bindResource.AppGuid == bind.PlatformAppID)
		fmt.Println("d")
	}
	fmt.Println("d")
	return isSameAttributes
}

// Bind method creates a Bind between given Service instance and the App.
func (apimBroker *APIM) Bind(ctx context.Context, instanceID, bindingID string,
	details domain.BindDetails, asyncAllowed bool) (domain.Binding, error) {

	logData := createCommonLogData(instanceID, details.ServiceID, details.PlanID)
	logData.Add(LogKeyBindID, bindingID)

	if !isApplicationPlan(details.ServiceID, details.PlanID) {
		log.Error("invalid instance id or plan id", ErrInvalidSVCPlan, logData)
		return domain.Binding{}, invalidServiceOrPlanFailureResponse("binding")
	}

	exists, bind, err := getBindFromDB(bindingID)
	if err != nil {
		return domain.Binding{}, err
	}
	var isWithSameAttr = false
	if exists {
		isWithSameAttr = isBindWithAttributes(bind, instanceID, details.BindResource)
		if !isWithSameAttr {
			return domain.Binding{}, apiresponses.ErrBindingAlreadyExists
		}
	}

	createServiceKey := isCreateServiceKey(details.BindResource)
	logData.Add("is-create-service-key", createServiceKey)

	platformAppID, err := getPlatformAppID(details.BindResource)
	if err != nil {
		return domain.Binding{}, err
	}
	logData.Add(LogKeyApplicationName, platformAppID)

	log.Debug("get instance from DB", logData)
	instance, err := getInstanceFromDB(instanceID)
	if err != nil {
		return domain.Binding{}, err
	}
	application := &db.Application{
		ID: instance.APIMResourceID,
	}
	// Not the first bind if application already in DB.
	hasBind, errDB := db.Retrieve(application)
	if errDB != nil {
		log.Error("unable to get application from DB", errDB, logData)
		return domain.Binding{}, internalServerFailureResponse("unable get application from DB",
			"get application information from DB")
	}

	if isWithSameAttr {
		credentialsMap := credentialsMap(application)
		return domain.Binding{
			Credentials:   credentialsMap,
			AlreadyExists: true,
		}, nil
	}

	if !hasBind {
		application, err = initApplication(instance.APIMResourceID)
		if err != nil {
			return domain.Binding{}, err
		}
		log.Debug("successfully stored the application in the DB", logData)
	}

	bind = &db.Bind{
		ID:                 bindingID,
		PlatformAppID:      platformAppID,
		InstanceID:         instanceID,
		ServiceID:          details.ServiceID,
		PlanID:             details.PlanID,
		IsCreateServiceKey: createServiceKey,
	}
	err = storeBind(bind)
	if err != nil {
		return domain.Binding{}, err
	}
	log.Debug("successfully stored the Bind in the DB", logData)
	credentialsMap := credentialsMap(application)
	return domain.Binding{
		Credentials: credentialsMap,
	}, nil
}

// getPlatformAppID returns platform app ID from the given domain.BindResource and any error encountered.
// Error type is apiresponses.FailureResponse.
func getPlatformAppID(b *domain.BindResource) (string, error) {
	var cfAppID string
	if isCreateServiceKey(b) {
		u, err := uuid.NewUUID()
		if err != nil {
			log.Error("unable to generate UUID for CF app", err, nil)
			return "", internalServerFailureResponse("unable to generate UUID for CF app", "generate UUID for CF app name")
		}
		cfAppID = u.String()
	} else {
		cfAppID = b.AppGuid
	}
	return cfAppID, nil
}

// initApplication function generates keys for the given Application and store the application in the database.
// Returns initialized *db.Application and any error encountered. Error type is apiresponses.FailureResponse.
func initApplication(appID string) (*db.Application, error) {
	logData := log.NewData().Add(LogKeyAppID, appID)
	appKeys, err := generateKeysForApplication(appID)
	if err != nil {
		return nil, err
	}
	log.Debug("successfully generated the keys", logData)
	application := &db.Application{
		ID:             appID,
		Token:          appKeys.Token.AccessToken,
		ConsumerSecret: appKeys.ConsumerSecret,
		ConsumerKey:    appKeys.ConsumerKey,
	}

	err = storeApplication(application)
	if err != nil {
		return nil, err
	}
	log.Debug("successfully stored the application in the DB", logData)
	return application, nil
}

// generateKeysForApplication function generates keys for the given Application.
// Returns initialized *apim.ApplicationKeyResp and type apiresponses.FailureResponse error if encountered.
func generateKeysForApplication(appID string) (*apim.ApplicationKeyResp, error) {
	logData := log.NewData().Add(LogKeyAppID, appID)
	appKeys, err := apim.GenerateKeys(appID)
	if err != nil {
		log.Error("unable generate keys for application", err, logData)
		return appKeys, internalServerFailureResponse("unable generate keys for application",
			"generate keys for application")
	}
	return appKeys, nil
}

// storeApplication function stores the given Application in the database.
// Returns type apiresponses.FailureResponse error if encountered.
func storeApplication(a *db.Application) error {
	logData := log.NewData().Add(LogKeyAppID, a.ID)
	err := db.Store(a)
	if err != nil {
		log.Error("unable store application in DB", err, logData)
		return internalServerFailureResponse("unable store application in DB",
			"store application in DB")
	}
	return nil
}

// storeBind function stores the given Bind in the database.
// Returns type apiresponses.FailureResponse error if encountered.
func storeBind(b *db.Bind) error {
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

func credentialsMap(app *db.Application) map[string]interface{} {
	return map[string]interface{}{
		"ConsumerKey":    app.ConsumerKey,
		"ConsumerSecret": app.ConsumerSecret,
		"AccessToken":    app.Token,
	}
}

// Unbind deletes the Bind from database and returns domain.UnbindSpec struct and any error encountered.
func (apimBroker *APIM) Unbind(ctx context.Context, instanceID, bindingID string,
	details domain.UnbindDetails, asyncAllowed bool) (domain.UnbindSpec, error) {

	logData := createCommonLogData(instanceID, details.ServiceID, details.PlanID)

	domainUnbindSpec := domain.UnbindSpec{}
	if !isApplicationPlan(details.ServiceID, details.PlanID) {
		log.Error("invalid instance id or plan id", ErrInvalidSVCPlan, logData)
		return domainUnbindSpec, invalidServiceOrPlanFailureResponse("unbinding")
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
func validateBindIDForDel(bindingID string) (*db.Bind, error) {
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
func (apimBroker *APIM) LastOperation(ctx context.Context, instanceID string,
	details domain.PollDetails) (domain.LastOperation, error) {
	return domain.LastOperation{}, errors.New("not supported")
}

func (apimBroker *APIM) Update(cxt context.Context, instanceID string,
	serviceDetails domain.UpdateDetails, asyncAllowed bool) (domain.UpdateServiceSpec, error) {
	logData := createCommonLogData(instanceID, serviceDetails.ServiceID, serviceDetails.PlanID)
	if isAPIPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return updateAPIService(instanceID, serviceDetails)
	} else if isApplicationPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return updateAppService(instanceID, serviceDetails)
	} else if isSubscriptionPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		// It is not possible to update a subscription
		return domain.UpdateServiceSpec{}, ErrNotSupported
	} else {
		log.Error("invalid instance id or plan id", ErrInvalidSVCPlan, logData)
		return domain.UpdateServiceSpec{}, invalidServiceOrPlanFailureResponse("updating")
	}
}

func (apimBroker *APIM) GetBinding(ctx context.Context, instanceID,
bindingID string) (domain.GetBindingSpec, error) {
	return domain.GetBindingSpec{}, ErrNotSupported
}

func (apimBroker *APIM) GetInstance(ctx context.Context,
	instanceID string) (domain.GetInstanceDetailsSpec, error) {
	return domain.GetInstanceDetailsSpec{}, ErrNotSupported
}

func (apimBroker *APIM) LastBindingOperation(ctx context.Context, instanceID,
bindingID string, details domain.PollDetails) (domain.LastOperation, error) {
	return domain.LastOperation{}, ErrNotSupported
}

// services returns an array of services offered by this service broker.
func services() []domain.Service {
	apiPlanBindable := false
	applicationPlanBindable := true
	subscriptionPlanBindable := false
	appPlanParameterSchema, err := utils.JSONSchema(apim.ApplicationInputSchemaRaw)
	if err != nil {
		log.HandleErrorAndExit("unable to get Input parameter JSON schema", err)
	}
	return []domain.Service{
		{
			ID:                   ServiceID,
			Name:                 ServiceName,
			Description:          ServiceDescription,
			Bindable:             false,
			InstancesRetrievable: false,
			PlanUpdatable:        true,

			Plans: []domain.ServicePlan{
				{
					ID:          APIPlanID,
					Name:        APIPlanName,
					Description: APIPlanDescription,
					Bindable:    &apiPlanBindable,
				},
				{
					ID:          ApplicationPlanID,
					Name:        ApplicationPlanName,
					Description: ApplicationPlanDescription,
					Bindable:    &applicationPlanBindable,
					Schemas: &domain.ServiceSchemas{
						Instance: domain.ServiceInstanceSchema{
							Create: domain.Schema{
								Parameters: appPlanParameterSchema,
							},
							Update: domain.Schema{
								Parameters: appPlanParameterSchema,
							},
						},
						Binding: domain.ServiceBindingSchema{
							Create: domain.Schema{
								Parameters: map[string]interface{}{},
							},
						},
					},
				},
				{
					ID:          SubscriptionPlanID,
					Name:        SubscriptionPlanName,
					Description: SubscriptionPlanDescription,
					Bindable:    &subscriptionPlanBindable,
				},
			},
		},
	}
}

// createAPIServiceInstance creates a API service instance and returns domain.ProvisionedServiceSpec and any error encountered.
func createAPIServiceInstance(instanceID string, serviceDetails domain.ProvisionDetails) (domain.ProvisionedServiceSpec, error) {
	logData := createCommonLogData(instanceID, serviceDetails.ServiceID, serviceDetails.PlanID)

	log.Debug("creating API service instance", logData)
	apiParam, err := getValidAPIParam(serviceDetails.RawParameters)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}
	logData.Add(LogKeyAPIName, apiParam.APISpec.Name)

	paramHash, err := utils.GenerateHash(apiParam)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, internalServerFailureResponse("unable to compare parameters",
			"compare parameters")
	}

	exists, provisionedServiceSpec, err := checkInstanceWithAttributes(instanceID, paramHash,
		serviceDetails.SpaceGUID, serviceDetails.OrganizationGUID)
	if err != nil || exists {
		return provisionedServiceSpec, err
	}

	apiID, err := createAPI(&apiParam.APISpec)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}
	log.Debug("successfully created API", logData)
	logData.Add(LogKeyAPIID, apiID)

	err = publishAPI(apiID)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}
	log.Debug("successfully published API", logData)

	// Store instance in the database
	i := &db.Instance{
		ServiceID:      serviceDetails.ServiceID,
		PlanID:         serviceDetails.PlanID,
		OrgID:          serviceDetails.OrganizationGUID,
		SpaceID:        serviceDetails.SpaceGUID,
		ID:             instanceID,
		APIMResourceID: apiID,
		ParameterHash:  paramHash,
	}
	err = db.Store(i)
	if err != nil {
		log.Error("unable to store API service instance", err, logData)
		return domain.ProvisionedServiceSpec{}, err
	}
	return domain.ProvisionedServiceSpec{}, nil
}

// createAPI creates the given API and returns an error type apiresponses.FailureResponse if encountered.
func createAPI(a *apim.APIReqBody) (string, error) {
	logData := log.NewData().Add(LogKeyAPIName, a.Name)
	apiID, err := apim.CreateAPI(a)
	if err != nil {
		log.Error("unable to create the API", err, logData)
		return "", handleAPIMResourceCreateError(err, a.Name)
	}
	return apiID, nil
}

// publishAPI publishes the given API and returns an error type apiresponses.FailureResponse if encountered.
func publishAPI(apiID string) error {
	logData := log.NewData().Add(LogKeyAPIID, apiID)
	if err := apim.PublishAPI(apiID); err != nil {
		log.Error("unable to publish API", err, logData)
		log.Debug("deleting the API", logData)
		errDel := apim.DeleteAPI(apiID)
		if errDel != nil {
			log.Error(fmt.Sprintf(ErrMsgFailedToCleanUp, "API"), errDel, logData)
		}
		log.Debug("deleted the API", logData)
		return internalServerFailureResponse("unable to publish the API", "publish the API")
	}
	return nil
}

// storeAPIServiceInstance store API service instance in the database and if any error encountered, deletes
// the API and returns an error type apiresponses.FailureResponse.
func storeAPIServiceInstance(i *db.Instance) error {
	logData := log.NewData().Add(LogKeyInstanceID, i.ID)
	err := db.Store(i)
	if err != nil {
		log.Error("unable to store instance", err, logData)
		log.Debug("deleting the API", logData)
		errDel := apim.DeleteAPI(i.APIMResourceID)
		if errDel != nil {
			log.Error(fmt.Sprintf(ErrMsgFailedToCleanUp, "API"), errDel, logData)
		}
		log.Debug("deleted the API", logData)
		return internalServerFailureResponse(ErrMsgUnableToStoreInstance, ErrActionStoreInstance)
	}
	return nil
}

func hasValidRawParameters(m *json.RawMessage) (bool, error) {
	s, err := utils.RawMsgToString(m)
	if err != nil {
		return false, errors.Wrap(err, "unable to get string value")
	}
	if s == "{}" {
		return false, nil
	}
	return true, nil
}

func createAppServiceInstance(instanceID string, serviceDetails domain.ProvisionDetails) (domain.ProvisionedServiceSpec, error) {
	logData := createCommonLogData(instanceID, serviceDetails.ServiceID, serviceDetails.PlanID)

	appParam, err := getValidApplicationParam(serviceDetails.RawParameters)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}
	paramHash, err := utils.GenerateHash(appParam)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, internalServerFailureResponse("unable to compare parameters",
			"compare parameters")
	}

	exists, provisionedServiceSpec, err := checkInstanceWithAttributes(instanceID, paramHash,
		serviceDetails.SpaceGUID, serviceDetails.OrganizationGUID)
	if err != nil || exists {
		return provisionedServiceSpec, err
	}

	logData.
		Add("throttlingTier", appParam.AppSpec.ThrottlingTier).
		Add("description", appParam.AppSpec.Description).
		Add("callbackUrl", appParam.AppSpec.CallbackURL).
		Add(LogKeyAPPName, appParam.AppSpec.Name)

	appID, err := createApplication(&appParam.AppSpec)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}
	logData.Add(LogKeyAppID, appID)
	log.Debug("application created", logData)

	// Store instance in the database
	i := &db.Instance{
		ServiceID:      serviceDetails.ServiceID,
		PlanID:         serviceDetails.PlanID,
		OrgID:          serviceDetails.OrganizationGUID,
		SpaceID:        serviceDetails.SpaceGUID,
		ID:             instanceID,
		APIMResourceID: appID,
		ParameterHash:  paramHash,
	}
	err = storeApplicationServiceInstance(i)
	if err != nil {
		log.Error("unable to store instance", err, logData)
		return domain.ProvisionedServiceSpec{}, err
	}
	return domain.ProvisionedServiceSpec{}, nil
}

// createApplication creates Application in API-M and returns App ID and an error type apiresponses.FailureResponse if encountered.
func createApplication(a *apim.ApplicationCreateReq) (string, error) {
	ld := log.NewData().Add(LogKeyApplicationName, a.Name)
	appID, err := apim.CreateApplication(a)
	if err != nil {
		log.Error("unable to create application", err, ld)
		return "", handleAPIMResourceCreateError(err, a.Name)
	}
	return appID, nil
}

// storeApplicationServiceInstance store Application service instance in the database and if any error encountered, deletes
// the Application and returns an error type apiresponses.FailureResponse.
func storeApplicationServiceInstance(i *db.Instance) error {
	logData := log.NewData().Add(LogKeyAPIID, i.ID)
	err := db.Store(i)
	if err != nil {
		log.Error("unable to store instance", err, logData)
		log.Debug("deleting the Application", logData)
		errDel := apim.DeleteApplication(i.APIMResourceID)
		if errDel != nil {
			log.Error("failed to cleanup, unable to delete the application", errDel, logData)
		}
		log.Debug("deleted the Application", logData)
		return internalServerFailureResponse("unable to store instance in DB", "storing instance in DB")
	}
	return nil
}

// handleAPIMResourceCreateError handles the API-M resource creation error. Returns an error type apiresponses.FailureResponse.
func handleAPIMResourceCreateError(e error, resourceName string) error {
	invokeErr, ok := e.(*client.InvokeError)
	if ok && invokeErr.StatusCode == http.StatusConflict {
		return internalServerFailureResponse(fmt.Sprintf("API-M resource %s already exist", resourceName), ErrActionCreateAPIMResource)
	}
	return internalServerFailureResponse("unable to create the API-M resource", ErrActionCreateAPIMResource)
}

func createSubscriptionServiceInstance(instanceID string, serviceDetails domain.ProvisionDetails) (domain.ProvisionedServiceSpec, error) {
	logData := createCommonLogData(instanceID, serviceDetails.ServiceID, serviceDetails.PlanID)

	subParam, err := getValidSubscriptionParam(serviceDetails.RawParameters)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}

	paramHash, err := utils.GenerateHash(subParam)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, internalServerFailureResponse("unable to compare parameters",
			"compare parameters")
	}

	exists, provisionedServiceSpec, err := checkInstanceWithAttributes(instanceID, paramHash,
		serviceDetails.SpaceGUID, serviceDetails.OrganizationGUID)
	if err != nil || exists {
		return provisionedServiceSpec, err
	}

	logData.Add(LogKeyAPIName, subParam.SubsSpec.APIName)
	log.Debug("search API", logData)
	apiID, err := searchAPI(subParam.SubsSpec.APIName)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}
	logData.
		Add(LogKeyAPIID, apiID).
		Add(LogKeyApplicationName, subParam.SubsSpec.AppName)

	log.Debug("search Application", logData)
	appID, err := searchApplication(subParam.SubsSpec.AppName)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}
	logData.Add(LogKeyAppID, appID)

	log.Debug("creating the subscription", logData)
	subsID, err := createSubscription(appID, apiID, subParam.SubsSpec.SubscriptionTier)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}
	logData.Add(LogKeySubsID, subsID)

	// Store instance in the database
	log.Debug("store Subscription service instance in DB", logData)
	i := &db.Instance{
		ServiceID:      serviceDetails.ServiceID,
		PlanID:         serviceDetails.PlanID,
		OrgID:          serviceDetails.OrganizationGUID,
		SpaceID:        serviceDetails.SpaceGUID,
		ID:             instanceID,
		APIMResourceID: subsID,
		ParameterHash:  paramHash,
	}
	err = storeSubscriptionServiceInstance(i)
	if err != nil {
		log.Error("unable to store instance", err, logData)
		return domain.ProvisionedServiceSpec{}, err
	}
	return domain.ProvisionedServiceSpec{}, nil
}

// createSubscription function creates a Subscription for the given API and Application. Returns an any error encountered.
// The error is type apiresponses.FailureResponse.
func createSubscription(appID, apiID, tier string) (string, error) {
	ld := log.NewData().
		Add(LogKeyAppID, appID).
		Add(LogKeyAPIID, apiID).
		Add("tier", tier)
	subsID, err := apim.Subscribe(appID, apiID, tier)
	if err != nil {
		log.Error("unable to create the subscription", err, ld)
		return "", internalServerFailureResponse("unable to create the subscription", "create subscription")
	}
	return subsID, nil
}

// searchApplication function returns Application ID for the given Application name and an any error encountered.
// The error is type apiresponses.FailureResponse.
func searchApplication(appName string) (string, error) {
	ld := log.NewData().Add(LogKeyApplicationName, appName)
	appID, err := apim.SearchApplication(appName)
	if err != nil {
		log.Error("unable to search Application", err, ld)
		return "", internalServerFailureResponse(fmt.Sprintf("couldn't find the Application: %s", appName), "searching application")
	}
	return appID, nil
}

// searchAPI function returns API ID for the given API name and an any error encountered.
// The error is type apiresponses.FailureResponse.
func searchAPI(apiName string) (string, error) {
	ld := log.NewData().Add(LogKeyAPIName, apiName)
	apiID, err := apim.SearchAPI(apiName)
	if err != nil {
		log.Error("unable to search API", err, ld)
		return "", internalServerFailureResponse(fmt.Sprintf("couldn't find the API: %s", apiName), "searching API")
	}
	return apiID, nil
}

// getValidSubscriptionParam function returns a valid apim.SubscriptionParam struct constructed from the given rawParameter and an any error encountered. The error is type apiresponses.FailureResponse.
func getValidSubscriptionParam(rawParameter json.RawMessage) (apim.SubscriptionParam, error) {
	var subParam apim.SubscriptionParam
	if rawParameter == nil {
		log.Error("invalid raw parameter", ErrInvalidRawParameter, nil)
		return subParam, apiresponses.ErrRawParamsInvalid
	}
	subParam, err := toSubscriptionParam(rawParameter)
	if err != nil {
		log.Error("unable to parse Subscription parameters", err, nil)
		return subParam, invalidParamFailureResponse(ErrActionParseSubscriptionParam)
	}
	if !utils.IsValidParams(subParam.SubsSpec.APIName, subParam.SubsSpec.AppName) {
		return subParam, invalidParamFailureResponse(ErrActionParseSubscriptionParam)
	}
	return subParam, nil
}

// storeSubscriptionServiceInstance stores Subscription service instance in the database and if any error encountered, revert
// the subscription and returns an error type apiresponses.FailureResponse.
func storeSubscriptionServiceInstance(i *db.Instance) error {
	logData := log.NewData().Add(LogKeyInstanceID, i.ID)
	err := db.Store(i)
	if err != nil {
		log.Error("unable to store instance", err, logData)
		log.Debug("revert subscription", logData)
		errDel := apim.UnSubscribe(i.APIMResourceID)
		if errDel != nil {
			log.Error("failed to cleanup, unable to delete the subscription", errDel, logData)
		}
		log.Debug("subscription is reverted", logData)
		return internalServerFailureResponse(ErrMsgUnableToStoreInstance, ErrActionStoreInstance)
	}
	return nil
}

func deleteAPIService(instanceID, serviceID, planID string) (domain.DeprovisionServiceSpec, error) {
	logData := createCommonLogData(instanceID, serviceID, planID)

	deprovisionServiceSpec := domain.DeprovisionServiceSpec{}
	instance, err := getInstanceFromDB(instanceID)
	if err != nil {
		return deprovisionServiceSpec, err
	}
	logData.Add(LogKeyAPIID, instance.APIMResourceID)
	log.Debug("delete the API", logData)
	errDel := apim.DeleteAPI(instance.APIMResourceID)
	if errDel != nil {
		log.Error("unable to delete the API", errDel, logData)
		return deprovisionServiceSpec, internalServerFailureResponse(ErrMsgUnableDelInstance, ErrActionDelAPI)
	}
	log.Debug(DebugMsgDelInstanceFromDB, logData)
	return deleteInstanceFromDB(instance)
}

// getInstanceFromDB function check whether the given instance already exists. If the given instance exists then an initialized instance and
// if the given instance doesn't exist or unable to retrieve it from database, an error type apiresponses.FailureResponse is returned.
func getInstanceFromDB(instanceID string) (*db.Instance, error) {
	ld := log.NewData().Add(LogKeyInstanceID, instanceID)
	instance := &db.Instance{
		ID: instanceID,
	}
	exists, err := db.Retrieve(instance)
	if err != nil {
		log.Error("unable to retrieve the instance from database", err, ld)
		return nil, internalServerFailureResponse("unable to query database", "get instance from the database")
	}
	if !exists {
		log.Error("instance doesn't exists", err, ld)
		return nil, apiresponses.ErrInstanceDoesNotExist
	}
	return instance, nil
}

// deleteInstanceFromDB function deletes the given instance from database. A domain.DeprovisionServiceSpec{} and an error type apiresponses.FailureResponse is returned.
func deleteInstanceFromDB(i *db.Instance) (domain.DeprovisionServiceSpec, error) {
	ld := log.NewData().Add(LogKeyInstanceID, i.ID)
	err := db.Delete(i)
	if err != nil {
		log.Error("unable to delete the instance from the database", err, ld)
		return domain.DeprovisionServiceSpec{}, internalServerFailureResponse(ErrMsgUnableDelInstance, ErrActionDelInstanceFromDB)
	}
	return domain.DeprovisionServiceSpec{}, nil
}

func deleteAppService(instanceID, serviceID, planID string) (domain.DeprovisionServiceSpec, error) {
	logData := createCommonLogData(instanceID, serviceID, planID)

	deprovisionServiceSpec := domain.DeprovisionServiceSpec{}
	instance, err := getInstanceFromDB(instanceID)
	if err != nil {
		return deprovisionServiceSpec, err
	}

	logData.Add(LogKeyAppID, instance.APIMResourceID)
	log.Debug("delete the application", logData)
	errDelApp := apim.DeleteApplication(instance.APIMResourceID)
	if errDelApp != nil {
		log.Error("unable to delete the Application", errDelApp, logData)
		return deprovisionServiceSpec, internalServerFailureResponse(ErrMsgUnableDelInstance, ErrActionDelAPP)
	}
	log.Debug(DebugMsgDelInstanceFromDB, logData)
	return deleteInstanceFromDB(instance)
}

func deleteSubscriptionService(instanceID, serviceID, planID string) (domain.DeprovisionServiceSpec, error) {
	logData := createCommonLogData(instanceID, serviceID, planID)

	deprovisionServiceSpec := domain.DeprovisionServiceSpec{}
	instance, err := getInstanceFromDB(instanceID)
	if err != nil {
		return domain.DeprovisionServiceSpec{}, err
	}

	logData.Add(LogKeySubsID, instance.APIMResourceID)
	log.Debug("remove subscription", logData)
	errUnSubscribe := apim.UnSubscribe(instance.APIMResourceID)
	if errUnSubscribe != nil {
		log.Error("unable to delete the Subscription", errUnSubscribe, logData)
		return deprovisionServiceSpec, internalServerFailureResponse(ErrMsgUnableDelInstance, ErrActionDelSubs)
	}
	log.Debug(DebugMsgDelInstanceFromDB, logData)
	return deleteInstanceFromDB(instance)
}

// getValidAPIParam function return a valid apim.APIParam struct constructed from the given rawParameter and an any error encountered.
// The error is type apiresponses.FailureResponse.
func getValidAPIParam(rawParameter json.RawMessage) (apim.APIParam, error) {
	var apiParam apim.APIParam
	if rawParameter == nil {
		log.Error("invalid raw parameter", ErrInvalidRawParameter, nil)
		return apiParam, apiresponses.ErrRawParamsInvalid
	}
	apiParam, err := toAPIParam(rawParameter)
	if err != nil {
		log.Error("unable to parse API parameters", err, nil)
		return apiParam, invalidParamFailureResponse("parse API JSON Spec parameter")
	}

	var has bool
	if has, err = hasValidRawParameters(&rawParameter); err != nil {
		log.Error("couldn't validate API parameters", err, nil)
		return apiParam, internalServerFailureResponse("couldn't validate API parameters", "validate API parameters")
	}
	if !has {
		log.Error("empty API parameters", err, nil)
		return apiParam, invalidParamFailureResponse("validate API parameters")
	}
	return apiParam, nil
}

func updateAPIService(instanceID string, serviceDetails domain.UpdateDetails) (domain.UpdateServiceSpec, error) {
	logData := createCommonLogData(instanceID, serviceDetails.ServiceID, serviceDetails.PlanID)
	log.Debug("updating API service instance", logData)

	updateServiceSpec := domain.UpdateServiceSpec{}
	instance, err := getInstanceFromDB(instanceID)
	if err != nil {
		return updateServiceSpec, err
	}
	apiParam, err := getValidAPIParam(serviceDetails.RawParameters)
	if err != nil {
		return updateServiceSpec, err
	}
	log.Debug("update the API", logData)
	apiParam.APISpec.ID = instance.APIMResourceID
	errUpdateAPI := apim.UpdateAPI(instance.APIMResourceID, &apiParam.APISpec)
	if errUpdateAPI != nil {
		log.Error("unable to update API", errUpdateAPI, logData)
		return updateServiceSpec, handleAPIMResourceUpdateError(errUpdateAPI, apiParam.APISpec.Name)
	}
	return updateServiceSpec, nil
}

func updateAppService(instanceID string, serviceDetails domain.UpdateDetails) (domain.UpdateServiceSpec, error) {
	logData := createCommonLogData(instanceID, serviceDetails.ServiceID, serviceDetails.PlanID)

	log.Debug("updating Application service instance", logData)
	updateServiceSpec := domain.UpdateServiceSpec{}
	instance, err := getInstanceFromDB(instanceID)
	if err != nil {
		return updateServiceSpec, err
	}
	applicationParam, err := getValidApplicationParam(serviceDetails.RawParameters)
	if err != nil {
		return updateServiceSpec, err
	}
	log.Debug("updating the application...", logData)
	errUpdate := apim.UpdateApplication(instance.APIMResourceID, &applicationParam.AppSpec)
	if errUpdate != nil {
		log.Error("unable to update Application", errUpdate, logData)
		return updateServiceSpec, handleAPIMResourceUpdateError(err, applicationParam.AppSpec.Name)
	}
	return updateServiceSpec, nil
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

func invalidServiceOrPlanFailureResponse(a string) error {
	return badRequestFailureResponse(a, "invalid services or Service")
}

func badRequestFailureResponse(action, msg string) error {
	return apiresponses.NewFailureResponse(errors.New(msg),
		http.StatusBadRequest, action)
}

func invalidParamFailureResponse(a string) error {
	return badRequestFailureResponse(a, "invalid parameters")
}

func isAPIPlan(serviceID, planID string) bool {
	return (serviceID == ServiceID) && (planID == APIPlanID)
}

func isApplicationPlan(serviceID, planID string) bool {
	return (serviceID == ServiceID) && (planID == ApplicationPlanID)
}

func isSubscriptionPlan(serviceID, planID string) bool {
	return (serviceID == ServiceID) && (planID == SubscriptionPlanID)
}

// toApplicationParam parses application parameters.
func toApplicationParam(params json.RawMessage) (apim.ApplicationParam, error) {
	var a apim.ApplicationParam
	err := json.Unmarshal(params, &a)
	if err != nil {
		return a, err
	}
	return a, nil
}

// toSubscriptionParam parses subscription parameters.
func toSubscriptionParam(params json.RawMessage) (apim.SubscriptionParam, error) {
	var s apim.SubscriptionParam
	err := json.Unmarshal(params, &s)
	if err != nil {
		return s, err
	}
	return s, nil
}

// toAPIParam parses API spec parameter.
func toAPIParam(params json.RawMessage) (apim.APIParam, error) {
	var a apim.APIParam
	err := json.Unmarshal(params, &a)
	if err != nil {
		return a, err
	}
	return a, nil
}

// checkInstanceWithAttributes function returns whether the given instanceID already exist in the database with the given attributes
// , domain.ProvisionedServiceSpec struct and error type apiresponses.FailureResponse.
func checkInstanceWithAttributes(instanceID, paramHash, spaceID, orgID string) (bool, domain.ProvisionedServiceSpec, error) {
	ld := log.NewData().Add(LogKeyInstanceID, instanceID)
	i := &db.Instance{
		ID: instanceID,
	}
	exists, err := db.Retrieve(i)
	if err != nil {
		log.Error("unable to get instance from DB", err, ld)
		return false, domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(errors.New("unable to query database"),
			http.StatusInternalServerError, "get the instance from DB")
	}
	if exists {
		if (paramHash == i.ParameterHash) && (spaceID == i.SpaceID) && (orgID == i.OrgID) {
			return exists, domain.ProvisionedServiceSpec{
				AlreadyExists: true,
			}, nil
		}
		return exists, domain.ProvisionedServiceSpec{}, apiresponses.ErrInstanceAlreadyExists

	}
	return false, domain.ProvisionedServiceSpec{}, nil
}

// getValidApplicationParam returns a valid apim.ApplicationParam struct and an error type apiresponses.FailureResponse if any
// error encountered.
func getValidApplicationParam(rawParameter json.RawMessage) (apim.ApplicationParam, error) {
	var appParam apim.ApplicationParam
	if rawParameter == nil {
		log.Error("invalid raw parameter", ErrInvalidRawParameter, nil)
		return appParam, apiresponses.ErrRawParamsInvalid
	}
	appParam, err := toApplicationParam(rawParameter)
	if err != nil {
		log.Error("invalid parameter in application json", err, nil)
		return appParam, invalidParamFailureResponse(ErrActionParseAppParam)
	}
	if !utils.IsValidParams(appParam.AppSpec.ThrottlingTier, appParam.AppSpec.Description,
		appParam.AppSpec.CallbackURL, appParam.AppSpec.Name) {
		return appParam, invalidParamFailureResponse(ErrActionParseAppParam)
	}
	return appParam, nil
}
