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

// dbutil package handles the DB connections and ORM
package dbutil

import (
	"fmt"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/config"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"log"
	"net/http"
	"strconv"
	"time"
)
import "github.com/jinzhu/gorm"

const (
	MySQL                    = "mysql"
	ErrMsgUnableToCloseDBCon = "unable to open a DB connection"
)

// Instance represents the Instance model in the Database
type Instance struct {
	Id               string `gorm:"primary_key;type:varchar(100)"`
	ServiceID        string `gorm:"type:varchar(100);not null"`
	PlanID           string `gorm:"type:varchar(100);not null"`
	APIMResourceID   string `gorm:"type:varchar(100);not null;unique"`
	APIMResourceName string `gorm:"type:varchar(100);not null;unique"`
}

// Application represents the Application model in the database
type Application struct {
	Name           string `gorm:"primary_key;type:varchar(100)"`
	InstanceID     string `gorm:"type:varchar(100);not null;unique"`
	Token          string `gorm:"type:varchar(100)"`
	ConsumerKey    string `gorm:"type:varchar(100)"`
	ConsumerSecret string `gorm:"type:varchar(100)"`
}

// Bind represents the Bind model in the Database
type Bind struct {
	Id                 string `gorm:"primary_key;type:varchar(100)"`
	InstanceID         string `gorm:"type:varchar(100);not null"`
	AppName            string `gorm:"type:varchar(100);not null"`
	ServiceID          string `gorm:"type:varchar(100);not null"`
	PlanID             string `gorm:"type:varchar(100);not null"`
	IsCreateServiceKey bool   `gorm:"type:BOOLEAN;not null;default:false"`
}

const (
	TableInstance        = "instances"
	TableBind            = "binds"
	TableApplication     = "applications"
	ErrTableExists       = 1050
	ErrMSGAPPNameMissing = "Name is missing"
	ErrMSGIDMissing      = "Id is missing"
)

// RetryPolicy defines a function which validate the response and apply desired policy
// to determine whether to retry the particular request or not
type RetryPolicy func(resp *http.Response) bool

// BackOffPolicy policy determines the duration between two retires
type BackOffPolicy func(min, max time.Duration, attempt int) time.Duration

var (
	url     string
	logMode bool
)

// Initialize database ORM parameters
func InitDB(conf *config.DBConfig) {
	url = conf.Username + ":" + conf.Password + "@tcp(" + conf.Host + ":" + strconv.Itoa(conf.Port) + ")/" +
		conf.Database + "?charset=utf8"
	logMode = conf.LogMode
}

// Creates a table for the given model only if table already not exists
func CreateTable(model interface{}, table string) {
	db, err := dbCon()
	if err != nil {
		utils.HandleErrorWithLoggerAndExit(ErrMsgUnableToCloseDBCon, err)
	}
	defer closeDBCon(db)
	if err := db.CreateTable(model).Error; err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			if mysqlErr.Number != ErrTableExists {
				utils.HandleErrorWithLoggerAndExit(fmt.Sprintf("couldn't create the table :%s", table), err)
			}
		} else {
			utils.HandleErrorWithLoggerAndExit(fmt.Sprintf("couldn't create the table :%s", table), err)
		}
	}
}

// dbCon returns a DB connection and any error occurred
func dbCon() (*gorm.DB, error) {
	var ld = &utils.LogData{}
	ld.AddData("dbURL", url).
		AddData("logMode", logMode)
	db, err := gorm.Open(MySQL, url)
	if err != nil {
		return nil, errors.Wrap(err, "cannot initiate ORM")
	}
	db.LogMode(logMode)
	ioWriter := utils.IoWriterLog()
	if ioWriter == nil {
		return nil, errors.New("IoWriter for logging is not initialized")
	}
	db.SetLogger(gorm.Logger{LogWriter: log.New(ioWriter, "database", 0)})
	return db, nil
}

func closeDBCon(d *gorm.DB) {
	if err := d.Close(); err != nil {
		utils.LogError("unable to close DB connection", err, nil)
	}
}

// store save the given Instance in the Database
func store(model interface{}, table string) error {
	db, err := dbCon()
	if err != nil {
		return errors.Wrap(err, ErrMsgUnableToCloseDBCon)
	}
	defer closeDBCon(db)
	return db.Table(table).Create(model).Error
}

// update updates the given Instance in the Database
func update(model interface{}, table string) error {
	db, err := dbCon()
	if err != nil {
		return errors.Wrap(err, ErrMsgUnableToCloseDBCon)
	}
	defer closeDBCon(db)
	return db.Table(table).Save(model).Error
}

// deleteEntry deletes the given Instance in the Database
func deleteEntry(model interface{}, table string) error {
	db, err := dbCon()
	if err != nil {
		return errors.Wrap(err, ErrMsgUnableToCloseDBCon)
	}
	defer closeDBCon(db)
	return db.Table(table).Delete(model).Error
}

// retrieve function returns the given Instance from the Database if exists and any error occurred
func retrieve(model interface{}, table string) (bool, error) {
	db, err := dbCon()
	if err != nil {
		return false, errors.Wrap(err, ErrMsgUnableToCloseDBCon)
	}
	defer closeDBCon(db)
	result := db.Table(table).Where(model).Find(model)
	if result.RecordNotFound() {
		return false, nil
	}
	return true, result.Error
}

// addForeignKeys configures foreign keys
func addForeignKeys() {
	db, err := dbCon()
	if err != nil {
		utils.HandleErrorWithLoggerAndExit(ErrMsgUnableToCloseDBCon, err)
	}
	defer closeDBCon(db)
	err = db.Model(&Bind{}).
		AddForeignKey("instance_id", TableInstance+"(id)", "CASCADE",
			"CASCADE").Error
	if err != nil {
		utils.HandleErrorWithLoggerAndExit("unable to add foreign keys", err)
	}
	err = db.Model(&Application{}).
		AddForeignKey("instance_id", TableInstance+"(id)", "CASCADE",
			"CASCADE").Error
	if err != nil {
		utils.HandleErrorWithLoggerAndExit("unable to add foreign keys", err)
	}
}

// CreateInstanceTable creates the Instance tables
func CreateTables() {
	CreateTable(&Instance{}, TableInstance)
	CreateTable(&Application{}, TableApplication)
	CreateTable(&Bind{}, TableBind)
	addForeignKeys()
}

// RetrieveInstance function returns the given Instance from the Database
func RetrieveInstance(i *Instance) (bool, error) {
	if i.Id == "" {
		return false, errors.New(ErrMSGIDMissing)
	}
	return retrieve(i, TableInstance)
}

// StoreInstance saves the Instance in the database
func StoreInstance(i *Instance) error {
	if i.Id == "" {
		return errors.New(ErrMSGIDMissing)
	}
	return store(i, TableInstance)
}

// DeleteInstance deletes the Instance in the database
func DeleteInstance(i *Instance) error {
	if i.Id == "" {
		return errors.New(ErrMSGIDMissing)
	}
	return deleteEntry(i, TableInstance)
}

// RetrieveBind function returns the given Bind from the Database
func RetrieveBind(b *Bind) (bool, error) {
	if b.Id == "" {
		return false, errors.New(ErrMSGIDMissing)
	}
	return retrieve(b, TableBind)
}

// StoreBind saves the Bind in the database
func StoreBind(b *Bind) error {
	if b.Id == "" {
		return errors.New(ErrMSGIDMissing)
	}
	return store(b, TableBind)
}

// DeleteBind deletes the Bind in the database
func DeleteBind(b *Bind) error {
	if b.Id == "" {
		return errors.New(ErrMSGIDMissing)
	}
	return deleteEntry(b, TableBind)
}

// RetrieveApp function returns the given Application from the Database
func RetrieveApp(a *Application) (bool, error) {
	if a.Name == "" {
		return false, errors.New(ErrMSGAPPNameMissing)
	}
	return retrieve(a, TableApplication)
}

// StoreApp saves the Application in the database
func StoreApp(b *Application) error {
	if b.Name == "" {
		return errors.New(ErrMSGAPPNameMissing)
	}
	return store(b, TableApplication)
}

// UpdateApp updates the application entry
func UpdateApp(b *Application) error {
	if b.Name == "" {
		return errors.New(ErrMSGAPPNameMissing)
	}
	return update(b, TableApplication)
}

// DeleteApp deletes the application entry
func DeleteApp(b *Application) error {
	if b.Name == "" {
		return errors.New(ErrMSGAPPNameMissing)
	}
	return deleteEntry(b, TableApplication)
}
