/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */

// dbutil package handles the DB connections and ORM
package dbutil

import (
	"fmt"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/wso2/service-broker-apim/pkg/config"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"strconv"
	"sync"
)
import "github.com/jinzhu/gorm"

const MySQL = "mysql"

// Instance represents the Instance model in the Database
type Instance struct {
	InstanceID string `gorm:"primary_key;type:varchar(100)"`
	ServiceID  string `gorm:"type:varchar(100)"`
	PlanID     string `gorm:"type:varchar(100)"`
	ApiID      string `gorm:"type:varchar(100)"`
	APIName    string `gorm:"type:varchar(100)"`
}

// Application represents the Application model in the database
type Application struct {
	AppName        string `gorm:"primary_key;type:varchar(100)"`
	AppID          string `gorm:"type:varchar(100)"`
	Token          string `gorm:"type:varchar(100)"`
	ConsumerKey    string `gorm:"type:varchar(100)"`
	ConsumerSecret string `gorm:"type:varchar(100)"`
}

// Bind represents the Bind model in the Database
type Bind struct {
	InstanceID     string `gorm:"type:varchar(100)"`
	BindID         string `gorm:"primary_key;type:varchar(100)"`
	AppName        string `gorm:"type:varchar(100)"`
	ServiceID      string `gorm:"type:varchar(100)"`
	PlanID         string `gorm:"type:varchar(100)"`
	SubscriptionID string `gorm:"type:varchar(100)"`
}

const (
	TableInstance    = "instances"
	TableBind        = "binds"
	TableApplication = "applications"
	ErrTableExists   = 1050
)

var (
	onceInit sync.Once
	db       *gorm.DB
)

// Initialize database ORM
func InitDB(conf *config.DBConfig) {
	onceInit.Do(func() {
		url := conf.Username + ":" + conf.Password + "@tcp(" + conf.Host + ":" + strconv.Itoa(conf.Port) + ")/" +
			conf.Database + "?charset=utf8&parseTime=True&loc=Local"
		var err error
		db, err = gorm.Open(MySQL, url)
		if err != nil {
			utils.HandleErrorWithLoggerAndExit("cannot initiate ORM", err)
		}
		//db.AutoMigrate(&Instance{})
		db.LogMode(conf.LogMode)
	})
}

// Creates a table for the given model only if table already not exists
func CreateTable(model interface{}, table string) {
	if err := db.CreateTable(model).Error; err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			// Not throw error if the table already exists
			if mysqlErr.Number != ErrTableExists {
				utils.HandleErrorWithLoggerAndExit(fmt.Sprintf("couldn't create the table :%s", table), err)
			}
		}
	}
}

// store save the given Instance in the Database
func store(model interface{}, table string) error {
	return db.Table(table).Create(model).Error
}

// update update the given Instance in the Database
func update(model interface{}, table string) error {
	return db.Table(table).Save(model).Error
}

// retrieve function returns the given Instance from the Database
func retrieve(model interface{}, table string) (bool, error) {
	result := db.Table(table).Where(model).Find(model)
	if result.RecordNotFound() {
		return false, nil
	}
	return true, result.Error
}

// CreateInstanceTable creates the Instance table
func CreateTables() {
	CreateTable(&Instance{}, TableInstance)
	CreateTable(&Bind{}, TableBind)
	CreateTable(&Application{}, TableApplication)
}

// RetrieveInstance function returns the given Instance from the Database
func RetrieveInstance(i *Instance) (bool, error) {
	return retrieve(i, TableInstance)
}

// StoreInstance saves the Instance in the database
func StoreInstance(i *Instance) error {
	return store(i, TableInstance)
}

// RetrieveBind function returns the given Bind from the Database
func RetrieveBind(b *Bind) (bool, error) {
	return retrieve(b, TableBind)
}

// StoreBind saves the Bind in the database
func StoreBind(b *Bind) error {
	return store(b, TableBind)
}

// RetrieveApp function returns the given Application from the Database
func RetrieveApp(a *Application) (bool, error) {
	return retrieve(a, TableApplication)
}

// StoreApp saves the Application in the database
func StoreApp(b *Application) error {
	return store(b, TableApplication)
}

// UpdateApp update the application entry
func UpdateApp(b *Application) error {
	return update(b, TableApplication)
}
