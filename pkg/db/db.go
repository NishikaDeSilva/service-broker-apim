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

// Package db handles the DB connections and ORM.
// Should be initialized with "Init" function before using.
package db

import (
	"fmt"
	"sync"

	// mysql driver is blank import for grom
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/config"
	"github.com/wso2/service-broker-apim/pkg/log"
	logPkg "log"
	"math"
	"strconv"
	"time"
)

const (
	MySQL                        = "mysql"
	ErrMsgUnableToOpenDBCon      = "unable to open a DB connect"
	TableInstance                = "instances"
	TableBind                    = "binds"
	TableApplication             = "applications"
	TableInstanceAPIMIDFieldName = "apim_resource_id"
	ForeignKeyDestAPIMID         = TableInstance + "(" + TableInstanceAPIMIDFieldName + ")"
	ForeignKeyDestInstanceID     = TableInstance + "(id)"
)

// Entity represents a table in the database.
type Entity interface {
	TableName() string
}

// TODO change name and pkg
// Instance represents the Instance model in the Database.
type Instance struct {
	ID             string `gorm:"primary_key;type:varchar(100)"`
	ServiceID      string `gorm:"type:varchar(100);not null"`
	PlanID         string `gorm:"type:varchar(100);not null"`
	APIMResourceID string `gorm:"type:varchar(100);not null;unique;column:apim_resource_id"`
	ParameterHash  string `gorm:"type:varchar(100);not null"`
	SpaceID        string `gorm:"type:varchar(100);not null"`
	OrgID          string `gorm:"type:varchar(100);not null"`
}

// Application represents the Application model in the database.
type Application struct {
	ID             string `gorm:"primary_key;type:varchar(100);not null;unique"`
	Token          string `gorm:"type:varchar(100)"`
	ConsumerKey    string `gorm:"type:varchar(100)"`
	ConsumerSecret string `gorm:"type:varchar(100)"`
}

// Bind represents the Bind model in the Database
type Bind struct {
	ID                 string `gorm:"primary_key;type:varchar(100)"`
	InstanceID         string `gorm:"type:varchar(100);not null"`
	PlatformAppID      string `gorm:"type:varchar(100);not null"`
	ServiceID          string `gorm:"type:varchar(100);not null"`
	PlanID             string `gorm:"type:varchar(100);not null"`
	IsCreateServiceKey bool   `gorm:"type:BOOLEAN;not null;default:false"`
}

var (
	url        string
	logMode    bool
	maxRetries int
	db         *gorm.DB
	once       sync.Once
)

func (Instance) TableName() string {
	return TableInstance
}

func (Bind) TableName() string {
	return TableBind
}

func (Application) TableName() string {
	return TableApplication
}

func backOff(min, max time.Duration, attempt int) time.Duration {
	du := math.Pow(2, float64(attempt))
	sleep := time.Duration(du) * time.Second
	if sleep < min {
		return min
	}
	if sleep > max {
		return max
	}
	return sleep
}

// Init initialize database parameters and open a DB connection.
func Init(conf *config.DB) {
	once.Do(func() {
		url = conf.Username + ":" + conf.Password + "@tcp(" + conf.Host + ":" + strconv.Itoa(conf.Port) + ")/" +
			conf.Database + "?charset=utf8"
		logMode = conf.LogMode
		maxRetries = conf.MaxRetries
		err := connect()
		if err != nil {
			log.HandleErrorAndExit(ErrMsgUnableToOpenDBCon, err)
		}
	})
}

// CreateTable creates a table for the given model only if table already not exists.
// Program will be closed if any error encountered.
func CreateTable(e Entity) {
	var ld = &log.Data{}
	ld.Add("table", e.TableName())

	if !db.HasTable(e.TableName()) {
		log.Debug("creating a table in the DB", ld)
		if err := db.CreateTable(e).Error; err != nil {
			log.HandleErrorAndExit(fmt.Sprintf("couldn't create the table :%s", e.TableName()), err)
		}
	} else {
		log.Debug("database already has the table", ld)
	}
}

// connect start a DB connection and returns any error occurred.
func connect() error {
	var ld = log.NewData().
		Add("logMode", logMode)
	var err error
	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(MySQL, url)
		if err == nil {
			break
		}
		bt := backOff(1*time.Second, 60*time.Second, i)
		ld.Add("attempt", i).
			Add("back-off time(seconds)", bt/time.Second)
		log.Debug(fmt.Sprintf("retrying the DB connection. err: %v", err), ld)
		time.Sleep(bt)
	}
	if err != nil {
		return errors.Wrap(err, "cannot initiate database connection")
	}
	if logMode {
		log.Debug("debug logs are enabled for Database", ld)
		db.LogMode(logMode)
		ioWriter := log.IoWriterLog()
		db.SetLogger(gorm.Logger{LogWriter: logPkg.New(ioWriter, "database", 0)})
	}
	return nil
}

// CloseDBCon function closes the open DB connections.
func CloseDBCon() {
	log.Debug("closing DB connection", nil)
	if err := db.Close(); err != nil {
		log.Error("unable to close the DB connection", err, nil)
	}
}

// Store saves the given Instance in the Database.
// Returns any error encountered.
func Store(e Entity) error {
	return db.Table(e.TableName()).Create(e).Error
}

// Update updates the given Instance in the Database.
// Returns any error encountered.
func Update(e Entity) error {
	return db.Table(e.TableName()).Save(e).Error
}

// Delete deletes the given Instance from the Database.
// Returns any error encountered.
func Delete(e Entity) error {
	return db.Table(e.TableName()).Delete(e).Error
}

// Retrieve function initialize the given Instance from the database if exists.
// Returns true if the instance exists and any error encountered.
func Retrieve(e Entity) (bool, error) {
	result := db.Table(e.TableName()).Where(e).Find(e)
	if result.RecordNotFound() {
		return false, nil
	}
	if result.Error != nil {
		return false, result.Error
	}
	return true, nil
}

// AddForeignKey adds a Foreign Key and returns any error encountered.
// Ex: db.AddForeignKey(&User{}).AddForeignKey("city_id", "cities(id)", "RESTRICT", "RESTRICT").
func AddForeignKey(e Entity, field string, dest string, onDelete string, onUpdate string) error {
	return db.Model(e).AddForeignKey(field, dest, onDelete, onUpdate).Error
}
