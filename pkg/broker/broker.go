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
package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/client"
	"github.com/wso2/service-broker-apim/pkg/config"
	"github.com/wso2/service-broker-apim/pkg/db"
	"github.com/wso2/service-broker-apim/pkg/log"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"net/http"
	"strconv"
)

const (
	ScopeAPICreate              = "apim:api_create"
	ScopeSubscribe              = "apim:subscribe"
	ScopeAPIPublish             = "apim:api_publish"
	ScopeAPIView                = "apim:api_view"
	ErrMSGInvalidSVCPlan        = "Invalid service or Plan"
	LogKeyAPIName               = "api-name"
	LogKeyAPIID                 = "api-id"
	LogKeyAPPID                 = "app-id"
	LogKeyAPPName               = "app-name"
	LogKeySubsID                = "subs-id"
	LogKeyServiceID             = "service-id"
	LogKeyPlanID                = "plan-id"
	LogKeyInstanceID            = "instance-id"
	LogKeyBindID                = "bind-id"
	LogKeyApplicationName       = "application-name"
	ErrMSGUnableToStoreInstance = "unable to store instance in DB"
	ErrActionStoreInstance      = "store instance in DB"
	ErrActionDelAPI             = "delete API"
	ErrActionDelAPP             = "delete Application"
	ErrActionDelSubs            = "delete Application"
	ErrActionDelInstanceFromDB  = "delete service instance from DB"
	ErrMsgUnableDelInstance     = "unable to delete service instance"
	ServiceId                   = "460F28F9-4D05-4889-970A-6BF5FB7D3CF8"
	ServiceName                 = "wso2apim-service"
	ServiceDescription          = "WSO2 API Manager Services"
	APIPlanID                   = "4CA4F2AF-EADF-4E76-A5CA-732FBC625593"
	APIPlanName                 = "api"
	APIPlanDescription          = "Create an API in WSO2 API Manager"
	ApplicationPlanID           = "00e851cd-ce8b-43eb-bc27-ac4d4fbb3204"
	ApplicationPlanName         = "app"
	ApplicationPlanDescription  = "Create an Application in WSO2 API Manager"
	SubscriptionPlanID          = "00e851cd-ce8b-43eb-bc27-ac4d4fbb3298"
	SubscriptionPlanName        = "subs"
	SubscriptionPlanDescription = "Create a Subscription in WSO2 API Manager"
)

var ErrNotSupported = errors.New("not supported")

// APIMServiceBroker struct holds the concrete implementation of the interface brokerapi.ServiceBroker
type APIMServiceBroker struct {
	BrokerConfig *config.BrokerConfig
	//TokenManager *TokenManager
	APIMManager *APIMClient
}

func (asb *APIMServiceBroker) Services(ctx context.Context) ([]brokerapi.Service, error) {
	return Plan(), nil
}

func (asb *APIMServiceBroker) Provision(ctx context.Context, instanceID string,
	serviceDetails domain.ProvisionDetails, asyncAllowed bool) (spec domain.ProvisionedServiceSpec, err error) {
	logData := log.NewData().
		Add(LogKeyServiceID, serviceDetails.ServiceID).
		Add(LogKeyPlanID, serviceDetails.PlanID).
		Add(LogKeyInstanceID, instanceID)
	if isAPIPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return asb.createAPIService(instanceID, serviceDetails)
	} else if isApplicationPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return asb.createAppService(instanceID, serviceDetails)
	} else if isSubscriptionPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return asb.createSubscriptionService(instanceID, serviceDetails)
	} else {
		log.Error("invalid instance id or plan id", errors.New(ErrMSGInvalidSVCPlan), logData)
		return spec, apiresponses.NewFailureResponse(errors.New("invalid Plan or Service"),
			http.StatusBadRequest, "provisioning")
	}
}

func (asb *APIMServiceBroker) Deprovision(ctx context.Context, instanceID string,
	serviceDetails domain.DeprovisionDetails, asyncAllowed bool) (domain.DeprovisionServiceSpec, error) {
	logData := log.NewData().
		Add(LogKeyServiceID, serviceDetails.ServiceID).
		Add(LogKeyPlanID, serviceDetails.PlanID).
		Add(LogKeyInstanceID, instanceID)
	if isAPIPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return asb.delAPIService(instanceID, serviceDetails)
	} else if isApplicationPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return asb.delAppService(instanceID, serviceDetails)
	} else if isSubscriptionPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return asb.delSubscriptionService(instanceID, serviceDetails)
	} else {
		log.Error("invalid instance id or plan id", errors.New(ErrMSGInvalidSVCPlan), logData)
		return domain.DeprovisionServiceSpec{}, invalidServiceOrPlanFailureResponse("provisioning")
	}
}

func (asb *APIMServiceBroker) Bind(ctx context.Context, instanceID, bindingID string,
	details domain.BindDetails, asyncAllowed bool) (domain.Binding, error) {
	logData := log.NewData().
		Add(LogKeyServiceID, details.ServiceID).
		Add(LogKeyPlanID, details.PlanID).
		Add(LogKeyInstanceID, instanceID).
		Add(LogKeyBindID, bindingID)
	log.Debug("binding invoked", logData)
	if !isApplicationPlan(details.ServiceID, details.PlanID) {
		log.Error("invalid instance id or plan id", errors.New(ErrMSGInvalidSVCPlan), logData)
		return domain.Binding{}, invalidServiceOrPlanFailureResponse("binding")
	}

	bind := &db.Bind{
		Id: bindingID,
	}
	exists, err := db.Retrieve(bind)
	if err != nil {
		log.Error("unable to retrieve Bind from the database", err, logData)
		return domain.Binding{}, failureResponse500("unable to retrieve Bind from the database", "get binding info from DB")
	}
	// Bind exists
	if exists {
		return domain.Binding{}, apiresponses.ErrBindingAlreadyExists
	}

	instance := &db.Instance{
		Id: instanceID,
	}
	exists, err = db.Retrieve(instance)
	if err != nil {
		log.Error("unable to get instance from DB", err, logData)
		return domain.Binding{}, failureResponse500("unable to query database",
			"getting the instance information from DB")
	}
	if !exists {
		log.Error("instance does not not exists", err, logData)
		return domain.Binding{}, apiresponses.ErrInstanceDoesNotExist
	}

	var (
		// Is the create service key flow
		isCreateServiceKey = isCreateServiceKey(details)
		// Application name
		cfAppName string
	)
	if isCreateServiceKey {
		u, err := uuid.NewUUID()
		if err != nil {
			return domain.Binding{}, failureResponse500("unable to generate UUID for CF app", "generate UUID for CF app name")
		}
		cfAppName = u.String()
	} else {
		cfAppName = details.BindResource.AppGuid
	}
	logData.Add(LogKeyApplicationName, cfAppName).Add("create-service-key", isCreateServiceKey)

	application := &db.Application{
		ID: instance.APIMResourceID,
	}
	// Not the first bind if application already in DB
	hasBind, err := db.Retrieve(application)
	if err != nil {
		log.Error("unable to get application from DB", err, logData)
		return domain.Binding{}, failureResponse500("unable get application from DB",
			"get application information from DB")
	}

	if !hasBind {
		appKeys, err := asb.APIMManager.GenerateKeys(instance.APIMResourceID)
		if err != nil {
			log.Error("unable generate keys for application", err, logData)
			return domain.Binding{}, failureResponse500("unable generate keys for application",
				"generate keys for application")
		}
		application.Token = appKeys.Token.AccessToken
		application.ConsumerSecret = appKeys.ConsumerSecret
		application.ConsumerKey = appKeys.ConsumerKey

		err = db.Store(application)
		if err != nil {
			log.Error("unable store application in DB", err, logData)
			return domain.Binding{}, failureResponse500("unable store application in DB",
				"store application in DB")
		}

	}

	// Construct Bind struct and store
	bind.AppName = cfAppName
	bind.InstanceID = instanceID
	bind.ServiceID = details.ServiceID
	bind.PlanID = details.PlanID
	bind.IsCreateServiceKey = isCreateServiceKey

	err = db.Store(bind)
	if err != nil {
		log.Error("unable to store bind", err, logData)
		return domain.Binding{}, failureResponse500("unable to store Bind in DB", "store Bind in DB")
	}

	credentialsMap := credentialsMap(application)
	return domain.Binding{
		Credentials: credentialsMap,
	}, nil
}

func isCreateServiceKey(d domain.BindDetails) bool {
	return d.BindResource == nil || d.BindResource.AppGuid == ""
}

func credentialsMap(app *db.Application) map[string]interface{} {
	return map[string]interface{}{
		"ConsumerKey":    app.ConsumerKey,
		"ConsumerSecret": app.ConsumerSecret,
		"AccessToken":    app.Token,
	}
}

func (asb *APIMServiceBroker) Unbind(ctx context.Context, instanceID, bindingID string,
	details domain.UnbindDetails, asyncAllowed bool) (domain.UnbindSpec, error) {
	logData := log.NewData().
		Add(LogKeyServiceID, details.ServiceID).
		Add(LogKeyPlanID, details.PlanID).
		Add(LogKeyInstanceID, instanceID)
	if !isApplicationPlan(details.ServiceID, details.PlanID) {
		log.Error("invalid instance id or plan id", errors.New(ErrMSGInvalidSVCPlan), logData)
		return domain.UnbindSpec{}, invalidServiceOrPlanFailureResponse("unbinding")
	}
	bind := &db.Bind{
		Id: bindingID,
	}
	exists, err := db.Retrieve(bind)
	if err != nil {
		log.Error("unable to retrieve Bind from the database", err, logData)
		return domain.UnbindSpec{}, err
	}
	// Bind not exists
	if !exists {
		return domain.UnbindSpec{}, apiresponses.ErrBindingDoesNotExist
	}

	logData.Add(LogKeyApplicationName, bind.AppName)
	err = db.Delete(bind)
	if err != nil {
		log.Error("unable to delete the bind from the database", err, logData)
		return domain.UnbindSpec{}, err
	}
	return domain.UnbindSpec{}, nil
}

// LastOperation ...
// If the broker provisions asynchronously, the Cloud Controller will poll this endpoint
// for the status of the provisioning operation.
func (apimServiceBroker *APIMServiceBroker) LastOperation(ctx context.Context, instanceID string,
	details domain.PollDetails) (domain.LastOperation, error) {
	return domain.LastOperation{}, errors.New("not supported")
}

func (asb *APIMServiceBroker) Update(cxt context.Context, instanceID string,
	serviceDetails domain.UpdateDetails, asyncAllowed bool) (domain.UpdateServiceSpec, error) {
	logData := log.NewData().
		Add(LogKeyServiceID, serviceDetails.ServiceID).
		Add(LogKeyPlanID, serviceDetails.PlanID).
		Add(LogKeyInstanceID, instanceID)
	if isAPIPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return asb.UpdateAPIService(instanceID, serviceDetails)
	} else if isApplicationPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return asb.UpdateAppService(instanceID, serviceDetails)
	} else if isSubscriptionPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		// It is not possible to update subscription
		return domain.UpdateServiceSpec{}, ErrNotSupported
	} else {
		log.Error("invalid instance id or plan id", errors.New(ErrMSGInvalidSVCPlan), logData)
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(errors.New("invalid Plan or Service"),
			http.StatusBadRequest, "updating")
	}
}

func (apimServiceBroker *APIMServiceBroker) GetBinding(ctx context.Context, instanceID,
bindingID string) (domain.GetBindingSpec, error) {
	return domain.GetBindingSpec{}, ErrNotSupported
}

func (apimServiceBroker *APIMServiceBroker) GetInstance(ctx context.Context,
	instanceID string) (domain.GetInstanceDetailsSpec, error) {
	return domain.GetInstanceDetailsSpec{}, ErrNotSupported
}

func (apimServiceBroker *APIMServiceBroker) LastBindingOperation(ctx context.Context, instanceID,
bindingID string, details domain.PollDetails) (domain.LastOperation, error) {
	return domain.LastOperation{}, ErrNotSupported
}

// Plan returns an array of services offered by this service broker
func Plan() []domain.Service {
	apiPlanBindable := false
	applicationPlanBindable := true
	subscriptionPlanBindable := false
	return []domain.Service{
		{
			ID:                   ServiceId,
			Name:                 ServiceName,
			Description:          ServiceDescription,
			Bindable:             false,
			InstancesRetrievable: false,
			PlanUpdatable:        false,

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

func (asb *APIMServiceBroker) createAPIService(instanceID string, serviceDetails domain.ProvisionDetails) (spec domain.ProvisionedServiceSpec, err error) {
	logData := log.NewData().
		Add(LogKeyServiceID, serviceDetails.ServiceID).
		Add(LogKeyPlanID, serviceDetails.PlanID).
		Add(LogKeyInstanceID, instanceID)
	log.Debug("creating API service instance", logData)
	// Verifying whether the instance is already exists
	exists, err := isInstanceExists(instanceID)
	// Handling DB connection error
	if err != nil {
		log.Error("unable to get instance from DB", err, logData)
		return spec, apiresponses.NewFailureResponse(errors.New("unable to query database"),
			http.StatusInternalServerError, "getting the instance information from DB")
	}
	if exists {
		log.Error(CreateAPIContext, apiresponses.ErrInstanceAlreadyExists, logData)
		return spec, apiresponses.ErrInstanceAlreadyExists
	}
	// Parse API JSON Spec
	apiParam, err := toAPIParam(serviceDetails.RawParameters)
	if err != nil {
		log.Error("unable to parse API parameters", err, logData)
		return spec, invalidParamFailureResponse("parse API JSON Spec parameter")
	}
	if has, err := hasValidParameters(&serviceDetails.RawParameters); err != nil {
		log.Error("couldn't validate API parameters", err, logData)
		return spec, failureResponse500("couldn't validate API parameters", "validate API parameters")
	} else {
		if !has {
			log.Error("empty API parameters", err, logData)
			return spec, invalidParamFailureResponse("validate API parameters")
		}
	}

	logData.Add(LogKeyAPIName, apiParam.APISpec.Name)

	apiID, err := asb.APIMManager.CreateAPI(&apiParam.APISpec)
	if err != nil {
		log.Error("unable to create API", err, logData)
		e, ok := err.(*client.InvokeError)
		if ok && e.StatusCode == http.StatusConflict {
			return spec, failureResponse500(fmt.Sprintf("API %s already exist !", apiParam.APISpec.Name), "create API")
		}
		return spec, failureResponse500("unable to create the API", "create API")
	}

	logData.Add(LogKeyAPIID, apiID)
	log.Debug("created API in APIM", logData)

	i := &db.Instance{
		ServiceID:      serviceDetails.ServiceID,
		PlanID:         serviceDetails.PlanID,
		Id:             instanceID,
		APIMResourceID: apiID,
	}

	// Store instance in the database
	err = db.Store(i)
	if err != nil {
		log.Error("unable to store instance", err, logData)
		errDel := asb.APIMManager.DeleteAPI(apiID)
		if errDel != nil {
			log.Error("failed to cleanup, unable to delete the API", errDel, logData)
		}
		return spec, failureResponse500(ErrMSGUnableToStoreInstance, ErrActionStoreInstance)
	}
	if err = asb.APIMManager.PublishAPI(apiID); err != nil {
		log.Error("unable to publish API", err, logData)
		errDel := asb.APIMManager.DeleteAPI(apiID)
		if errDel != nil {
			log.Error("failed to cleanup, unable to delete the API", errDel, logData)
		}
		return spec, apiresponses.NewFailureResponse(errors.New("unable to publish the API"),
			http.StatusInternalServerError, "unable to publish the API")
	}
	log.Debug("published API in APIM", logData)
	return spec, nil
}

func hasValidParameters(m *json.RawMessage) (bool, error) {
	s, err := utils.RawMSGToString(m)
	if err != nil {
		return false, errors.Wrap(err, "unable to get string value")
	}
	if s == "{}" {
		return false, nil
	}
	return true, nil
}

func (asb *APIMServiceBroker) createAppService(instanceID string, serviceDetails domain.ProvisionDetails) (domain.ProvisionedServiceSpec, error) {
	logData := log.NewData().
		Add(LogKeyServiceID, serviceDetails.ServiceID).
		Add(LogKeyPlanID, serviceDetails.PlanID).
		Add(LogKeyInstanceID, instanceID)
	// Verifying whether the instance is already exists
	exists, err := isInstanceExists(instanceID)
	// Handling DB connection error
	if err != nil {
		log.Error("unable to get instance from DB", err, logData)
		return domain.ProvisionedServiceSpec{}, err
	}
	if exists {
		log.Error(CreateApplicationContext, apiresponses.ErrInstanceAlreadyExists, logData)
		return domain.ProvisionedServiceSpec{}, apiresponses.ErrInstanceAlreadyExists
	}
	// Validates the parameters before moving forward
	applicationParam, err := toApplicationParam(serviceDetails.RawParameters)
	if err != nil {
		log.Error("invalid parameter in application json", err, logData)
		return domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(errors.New("invalid parameter"),
			http.StatusBadRequest, "parsing ApplicationConfig JSON Spec parameter")
	}
	if !utils.IsValidParams(applicationParam.AppSpec.ThrottlingTier, applicationParam.AppSpec.Description,
		applicationParam.AppSpec.CallbackUrl, applicationParam.AppSpec.Name) {
		return domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(errors.New("invalid parameters"),
			http.StatusBadRequest, "Parsing Subscription parameters")
	}
	appCreateReq := &ApplicationCreateReq{
		ThrottlingTier: applicationParam.AppSpec.ThrottlingTier,
		Description:    applicationParam.AppSpec.Description,
		Name:           applicationParam.AppSpec.Name,
		CallbackUrl:    applicationParam.AppSpec.CallbackUrl,
	}

	logData.
		Add("throttlingTier", appCreateReq.ThrottlingTier).
		Add("description", appCreateReq.Description).
		Add("callbackUrl", appCreateReq.CallbackUrl).
		Add(LogKeyAPPName, appCreateReq.Name)
	log.Debug("creating a new application...", logData)
	appID, err := asb.APIMManager.CreateApplication(appCreateReq)
	if err != nil {
		e, ok := err.(*client.InvokeError)
		if ok {
			logData.Add("response code", strconv.Itoa(e.StatusCode))
		}
		log.Error("unable to create application", err, logData)
		return domain.ProvisionedServiceSpec{}, failureResponse500(ErrMSGUnableToStoreInstance, ErrActionStoreInstance)
	}
	i := &db.Instance{
		ServiceID:      serviceDetails.ServiceID,
		PlanID:         serviceDetails.PlanID,
		Id:             instanceID,
		APIMResourceID: appID,
	}
	logData.Add(LogKeyAPPID, appID)
	// Store instance in the database
	err = db.Store(i)
	if err != nil {
		log.Error("unable to store instance", err, logData)
		errDel := asb.APIMManager.DeleteApplication(appID)
		if errDel != nil {
			log.Error("failed to cleanup, unable to delete the application", errDel, logData)
		}
		return domain.ProvisionedServiceSpec{}, failureResponse500("unable to store instance in DB", "storing instance in DB")
	}
	return domain.ProvisionedServiceSpec{}, nil
}

func (asb *APIMServiceBroker) createSubscriptionService(instanceID string, serviceDetails domain.ProvisionDetails) (spec domain.ProvisionedServiceSpec, err error) {
	logData := log.NewData().
		Add(LogKeyServiceID, serviceDetails.ServiceID).
		Add(LogKeyPlanID, serviceDetails.PlanID).
		Add(LogKeyInstanceID, instanceID)
	// Verifying whether the instance is already exists
	exists, err := isInstanceExists(instanceID)
	// Handling DB connection error
	if err != nil {
		log.Error("unable to get instance from DB", err, logData)
		return domain.ProvisionedServiceSpec{}, failureResponse500("unable to query database", "getting the instance information from DB")
	}
	if exists {
		log.Error(CreateSubscriptionContext, apiresponses.ErrInstanceAlreadyExists, logData)
		return domain.ProvisionedServiceSpec{}, apiresponses.ErrInstanceAlreadyExists
	}
	subsInfo, err := toSubscriptionParam(serviceDetails.RawParameters)
	if err != nil {
		log.Error("unable to parse Subscription parameters", err, logData)
		return spec, invalidParamFailureResponse("parsing Subscription parameters")
	}
	if !utils.IsValidParams(subsInfo.SubsSpec.APIName, subsInfo.SubsSpec.AppName) {
		return domain.ProvisionedServiceSpec{}, invalidParamFailureResponse("parsing Subscription parameters")
	}
	logData.Add("API name", subsInfo.SubsSpec.APIName).Add("Application Name", subsInfo.SubsSpec.AppName)
	apiID, err := asb.APIMManager.SearchAPI(subsInfo.SubsSpec.APIName)
	if err != nil {
		log.Error("unable to search API", err, logData)
		return spec, failureResponse500(fmt.Sprintf("couldn't find the API: %s", subsInfo.SubsSpec.APIName), "searching API")
	}
	logData.Add(LogKeyAPIID, apiID)
	appID, err := asb.APIMManager.SearchApplication(subsInfo.SubsSpec.AppName)
	if err != nil {
		log.Error("unable to search Application", err, logData)
		return spec, failureResponse500(fmt.Sprintf("couldn't find the Application: %s", subsInfo.SubsSpec.AppName), "searching application")
	}
	logData.Add(LogKeyAPPID, appID)
	subsID, err := asb.APIMManager.Subscribe(appID, apiID, subsInfo.SubsSpec.SubscriptionTier)
	if err != nil {
		log.Error("unable to create the subscription", err, logData)
		return spec, failureResponse500("unable to create the subscription", "create subscription")
	}
	i := &db.Instance{
		ServiceID:      serviceDetails.ServiceID,
		PlanID:         serviceDetails.PlanID,
		Id:             instanceID,
		APIMResourceID: subsID,
	}
	logData.Add(LogKeySubsID, subsID)
	// Store instance in the database
	err = db.Store(i)
	if err != nil {
		log.Error("unable to store instance", err, logData)
		errDel := asb.APIMManager.UnSubscribe(subsID)
		if errDel != nil {
			log.Error("failed to cleanup, unable to delete the subscription", errDel, logData)
		}
		return spec, failureResponse500(ErrMSGUnableToStoreInstance, ErrActionStoreInstance)
	}
	return domain.ProvisionedServiceSpec{}, nil
}

func (asb *APIMServiceBroker) delAPIService(instanceID string, serviceDetails domain.DeprovisionDetails) (domain.DeprovisionServiceSpec, error) {
	logData := log.NewData().
		Add(LogKeyServiceID, serviceDetails.ServiceID).
		Add(LogKeyPlanID, serviceDetails.PlanID).
		Add(LogKeyInstanceID, instanceID)
	instance := &db.Instance{
		Id: instanceID,
	}
	exists, err := db.Retrieve(instance)
	if err != nil {
		log.Error("unable to get instance from DB", err, logData)
		return domain.DeprovisionServiceSpec{}, err
	}
	if !exists {
		return domain.DeprovisionServiceSpec{}, apiresponses.ErrInstanceDoesNotExist
	}

	logData.Add(LogKeyAPIID, instance.APIMResourceID)
	err = asb.APIMManager.DeleteAPI(instance.APIMResourceID)
	if err != nil {
		log.Error("unable to delete the API", err, logData)
		return domain.DeprovisionServiceSpec{}, failureResponse500(ErrMsgUnableDelInstance, ErrActionDelAPI)
	}
	err = db.Delete(instance)
	if err != nil {
		log.Error("unable to delete the instance from the database", err, logData)
		return domain.DeprovisionServiceSpec{}, failureResponse500(ErrMsgUnableDelInstance, ErrActionDelInstanceFromDB)
	}
	return domain.DeprovisionServiceSpec{}, nil
}

func (asb *APIMServiceBroker) delAppService(instanceID string, serviceDetails domain.DeprovisionDetails) (domain.DeprovisionServiceSpec, error) {
	logData := log.NewData().
		Add(LogKeyServiceID, serviceDetails.ServiceID).
		Add(LogKeyPlanID, serviceDetails.PlanID).
		Add(LogKeyInstanceID, instanceID)
	instance := &db.Instance{
		Id: instanceID,
	}
	exists, err := db.Retrieve(instance)
	if err != nil {
		log.Error("unable to get instance from DB", err, logData)
		return domain.DeprovisionServiceSpec{}, err
	}
	if !exists {
		return domain.DeprovisionServiceSpec{}, apiresponses.ErrInstanceDoesNotExist
	}

	logData.Add(LogKeyAPPID, instance.APIMResourceID)
	err = asb.APIMManager.DeleteApplication(instance.APIMResourceID)
	if err != nil {
		log.Error("unable to delete the Application", err, logData)
		return domain.DeprovisionServiceSpec{}, failureResponse500(ErrMsgUnableDelInstance, ErrActionDelAPP)
	}
	err = db.Delete(instance)
	if err != nil {
		log.Error("unable to delete the instance from the database", err, logData)
		return domain.DeprovisionServiceSpec{}, failureResponse500(ErrMsgUnableDelInstance, ErrActionDelInstanceFromDB)
	}
	return domain.DeprovisionServiceSpec{}, nil
}

func (asb *APIMServiceBroker) delSubscriptionService(instanceID string, serviceDetails domain.DeprovisionDetails) (domain.DeprovisionServiceSpec, error) {
	logData := log.NewData().
		Add(LogKeyServiceID, serviceDetails.ServiceID).
		Add(LogKeyPlanID, serviceDetails.PlanID).
		Add(LogKeyInstanceID, instanceID)
	instance := &db.Instance{
		Id: instanceID,
	}
	exists, err := db.Retrieve(instance)
	if err != nil {
		log.Error("unable to get instance from DB", err, logData)
		return domain.DeprovisionServiceSpec{}, err
	}
	if !exists {
		return domain.DeprovisionServiceSpec{}, apiresponses.ErrInstanceDoesNotExist
	}

	logData.Add(LogKeySubsID, instance.APIMResourceID)
	err = asb.APIMManager.UnSubscribe(instance.APIMResourceID)
	if err != nil {
		log.Error("unable to delete the Subscription", err, logData)
		return domain.DeprovisionServiceSpec{}, failureResponse500(ErrMsgUnableDelInstance, ErrActionDelSubs)
	}
	err = db.Delete(instance)
	if err != nil {
		log.Error("unable to delete the instance from the database", err, logData)
		return domain.DeprovisionServiceSpec{}, failureResponse500(ErrMsgUnableDelInstance, ErrActionDelInstanceFromDB)
	}
	return domain.DeprovisionServiceSpec{}, nil
}

func (asb *APIMServiceBroker) UpdateAPIService(instanceID string, serviceDetails domain.UpdateDetails) (domain.UpdateServiceSpec, error) {
	logData := log.NewData().
		Add(LogKeyServiceID, serviceDetails.ServiceID).
		Add(LogKeyPlanID, serviceDetails.PlanID).
		Add(LogKeyInstanceID, instanceID)
	log.Debug("updating API service instance", logData)
	instance := &db.Instance{
		Id: instanceID,
	}
	exists, err := db.Retrieve(instance)
	if err != nil {
		log.Error("unable to get instance from DB", err, logData)
		return domain.UpdateServiceSpec{}, err
	}
	if !exists {
		log.Error(UpdateAPIContext, apiresponses.ErrInstanceDoesNotExist, logData)
		return domain.UpdateServiceSpec{}, apiresponses.ErrInstanceDoesNotExist
	}
	// Parse API JSON Spec
	apiParam, err := toAPIParam(serviceDetails.RawParameters)
	if err != nil {
		log.Error("unable to parse API parameters", err, logData)
		return domain.UpdateServiceSpec{}, invalidParamFailureResponse("parse API JSON Spec parameter")
	}
	if has, err := hasValidParameters(&serviceDetails.RawParameters); err != nil {
		log.Error("couldn't validate API parameters", err, logData)
		return domain.UpdateServiceSpec{}, failureResponse500("couldn't validate API parameters", "validate API parameters")
	} else {
		if !has {
			log.Error("empty API parameters", err, logData)
			return domain.UpdateServiceSpec{}, invalidParamFailureResponse("validate API parameters")
		}
	}
	apiParam.APISpec.ID = instance.APIMResourceID
	err = asb.APIMManager.UpdateAPI(instance.APIMResourceID, &apiParam.APISpec)
	if err != nil {
		log.Error("unable to update API", err, logData)
		e, ok := err.(*client.InvokeError)
		if ok && e.StatusCode == http.StatusNotFound {
			return domain.UpdateServiceSpec{}, failureResponse500(fmt.Sprintf("API %s not found !", apiParam.APISpec.Name), "update API")
		}
		return domain.UpdateServiceSpec{}, failureResponse500("unable to update the API", "update API")
	}
	return domain.UpdateServiceSpec{}, nil
}

func (asb *APIMServiceBroker) UpdateAppService(instanceID string, serviceDetails domain.UpdateDetails) (domain.UpdateServiceSpec, error) {
	logData := log.NewData().
		Add(LogKeyServiceID, serviceDetails.ServiceID).
		Add(LogKeyPlanID, serviceDetails.PlanID).
		Add(LogKeyInstanceID, instanceID)
	log.Debug("updating Application service instance", logData)
	instance := &db.Instance{
		Id: instanceID,
	}
	exists, err := db.Retrieve(instance)
	if err != nil {
		log.Error("unable to get instance from DB", err, logData)
		return domain.UpdateServiceSpec{}, err
	}
	if !exists {
		log.Error(UpdateApplicationContext, apiresponses.ErrInstanceDoesNotExist, logData)
		return domain.UpdateServiceSpec{}, apiresponses.ErrInstanceDoesNotExist
	}
	// Validates the parameters before moving forward
	applicationParam, err := toApplicationParam(serviceDetails.RawParameters)
	if err != nil {
		log.Error("invalid parameter in application json", err, logData)
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(errors.New("invalid parameter"),
			http.StatusBadRequest, "parsing ApplicationConfig JSON Spec parameter")
	}
	if !utils.IsValidParams(applicationParam.AppSpec.ThrottlingTier, applicationParam.AppSpec.Description,
		applicationParam.AppSpec.CallbackUrl, applicationParam.AppSpec.Name) {
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(errors.New("invalid parameters"),
			http.StatusBadRequest, "parsing Subscription parameters")
	}
	appCreateReq := &ApplicationCreateReq{
		ThrottlingTier: applicationParam.AppSpec.ThrottlingTier,
		Description:    applicationParam.AppSpec.Description,
		Name:           applicationParam.AppSpec.Name,
		CallbackUrl:    applicationParam.AppSpec.CallbackUrl,
	}

	logData.
		Add("throttlingTier", appCreateReq.ThrottlingTier).
		Add("description", appCreateReq.Description).
		Add("callbackUrl", appCreateReq.CallbackUrl).
		Add(LogKeyAPPName, appCreateReq.Name)
	log.Debug("updating a new application...", logData)
	err = asb.APIMManager.UpdateApplication(instance.APIMResourceID, &applicationParam.AppSpec)
	if err != nil {
		log.Error("unable to update Application", err, logData)
		e, ok := err.(*client.InvokeError)
		if ok && e.StatusCode == http.StatusNotFound {
			return domain.UpdateServiceSpec{}, failureResponse500(fmt.Sprintf("Application %s not found !", applicationParam.AppSpec.Name), "update App")
		}
		return domain.UpdateServiceSpec{}, failureResponse500("unable to update the Application", "update App")
	}
	return domain.UpdateServiceSpec{}, nil
}

func failureResponse500(m, a string) *apiresponses.FailureResponse {
	return apiresponses.NewFailureResponse(errors.New(m), http.StatusInternalServerError, a)
}
func invalidServiceOrPlanFailureResponse(a string) *apiresponses.FailureResponse {
	return apiresponses.NewFailureResponse(errors.New("invalid Plan or Service"),
		http.StatusBadRequest, a)
}

func invalidParamFailureResponse(a string) *apiresponses.FailureResponse {
	return apiresponses.NewFailureResponse(errors.New("invalid parameters"),
		http.StatusBadRequest, a)
}

func isAPIPlan(serviceID, planID string) bool {
	return (serviceID == ServiceId) && (planID == APIPlanID)
}

func isApplicationPlan(serviceID, planID string) bool {
	return (serviceID == ServiceId) && (planID == ApplicationPlanID)
}

func isSubscriptionPlan(serviceID, planID string) bool {
	return (serviceID == ServiceId) && (planID == SubscriptionPlanID)
}

// toApplicationParam parses application parameters
func toApplicationParam(params json.RawMessage) (ApplicationParam, error) {
	var a ApplicationParam
	err := json.Unmarshal(params, &a)
	if err != nil {
		return a, errors.Wrap(err, "unable to parse application parameters")
	}
	return a, nil
}

// toSubscriptionParam parses subscription parameters
func toSubscriptionParam(params json.RawMessage) (SubscriptionParam, error) {
	var s SubscriptionParam
	err := json.Unmarshal(params, &s)
	if err != nil {
		return s, errors.Wrap(err, "unable to parse subscription parameters")
	}
	return s, nil
}

// toAPIParam parses API spec parameter
func toAPIParam(params json.RawMessage) (APIParam, error) {
	var a APIParam
	err := json.Unmarshal(params, &a)
	if err != nil {
		return a, errors.Wrap(err, "unable to parse API parameters")
	}
	return a, nil
}

// isInstanceExists returns true if the given instanceID already exists in the DB
func isInstanceExists(instanceID string) (bool, error) {
	i := &db.Instance{
		Id: instanceID,
	}
	exists, err := db.Retrieve(i)
	if err != nil {
		return false, err
	}
	return exists, nil
}
