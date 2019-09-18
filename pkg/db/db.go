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

// db package handles the DB connections and ORM
package db

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/config"
	"github.com/wso2/service-broker-apim/pkg/log"
	logPkg "log"
	"math"
	"strconv"
	"time"
)
import "github.com/jinzhu/gorm"

const (
	MySQL                        = "mysql"
	ErrMsgUnableToOpenDBCon      = "unable to open a DB connection"
	ErrMsgUnableToAddForeignKeys = "unable to add foreign keys"
	TableInstance                = "instances"
	TableBind                    = "binds"
	TableApplication             = "applications"
)

type Entity interface {
	TableName() string
}

// Instance represents the Instance model in the Database
type Instance struct {
	Id             string `gorm:"primary_key;type:varchar(100)"`
	ServiceID      string `gorm:"type:varchar(100);not null"`
	PlanID         string `gorm:"type:varchar(100);not null"`
	APIMResourceID string `gorm:"type:varchar(100);not null;unique;column:apim_resource_id"`
}

// Application represents the Application model in the database
type Application struct {
	ID             string `gorm:"primary_key;type:varchar(100);not null;unique"`
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

var (
	url        string
	logMode    bool
	maxRetries int
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

// backOff waits until attempt^2 or (min,max)
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

// Initialize database ORM parameters
func InitDB(conf *config.DBConfig) {
	url = conf.Username + ":" + conf.Password + "@tcp(" + conf.Host + ":" + strconv.Itoa(conf.Port) + ")/" +
		conf.Database + "?charset=utf8"
	logMode = conf.LogMode
	maxRetries = conf.MaxRetries
}

// Creates a table for the given model only if table already not exists
func CreateTable(model interface{}, table string) {
	db, err := dbCon()
	if err != nil {
		log.HandleErrorWithLoggerAndExit(ErrMsgUnableToOpenDBCon, err)
	}
	defer closeDBCon(db)

	var ld = &log.Data{}
	ld.Add("table", table)

	if ! db.HasTable(table) {
		log.Debug("creating a table in the DB", ld)
		if err := db.CreateTable(model).Error; err != nil {
			log.HandleErrorWithLoggerAndExit(fmt.Sprintf("couldn't create the table :%s", table), err)
		}
	} else {
		log.Debug("database already has the table", ld)
	}
}

// dbCon returns a DB connection and any error occurred
func dbCon() (*gorm.DB, error) {
	var ld = log.NewData().
		Add("dbURL", url).
		Add("logMode", logMode)
	var db *gorm.DB
	var err error
	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(MySQL, url)
		if err == nil {
			break
		}
		bt := backOff(1*time.Second, 10*time.Second, i)
		ld.Add("attempt", i).
			Add("back-off time(seconds)", bt/time.Second)
		log.Debug(fmt.Sprintf("retrying the DB connection err: %v", err), ld)
		time.Sleep(bt)
	}

	if err != nil {
		return nil, errors.Wrap(err, "cannot initiate ORM")
	}
	if logMode {
		db.LogMode(logMode)
		ioWriter := log.IoWriterLog()
		if ioWriter == nil {
			return nil, errors.New("IoWriter for logging is not initialized")
		}
		db.SetLogger(gorm.Logger{LogWriter: logPkg.New(ioWriter, "database", 0)})
	}
	return db, nil
}

func closeDBCon(d *gorm.DB) {
	if err := d.Close(); err != nil {
		log.Error("unable to close DB connection", err, nil)
	}
}

// store save the given Instance in the Database
func Store(e Entity) error {
	db, err := dbCon()
	if err != nil {
		return errors.Wrap(err, ErrMsgUnableToOpenDBCon)
	}
	defer closeDBCon(db)
	return db.Table(e.TableName()).Create(e).Error
}

// update updates the given Instance in the Database
func update(e Entity) error {
	db, err := dbCon()
	if err != nil {
		return errors.Wrap(err, ErrMsgUnableToOpenDBCon)
	}
	defer closeDBCon(db)
	return db.Table(e.TableName()).Save(e).Error
}

// deleteEntry deletes the given Instance in the Database
// returns an error occurred
func Delete(e Entity) error {
	db, err := dbCon()
	if err != nil {
		return errors.Wrap(err, ErrMsgUnableToOpenDBCon)
	}
	defer closeDBCon(db)
	return db.Table(e.TableName()).Delete(e).Error
}

// Retrieve function returns the given Instance from the Database if exists and any error occurred
// returns true if the instance exists and an error if occurred
func Retrieve(e Entity) (bool, error) {
	db, err := dbCon()
	if err != nil {
		return false, errors.Wrap(err, ErrMsgUnableToOpenDBCon)
	}
	defer closeDBCon(db)
	result := db.Table(e.TableName()).Where(e).Find(e)
	if result.RecordNotFound() {
		return false, nil
	}

	if result.Error != nil {
		return false, err
	}
	return true, nil
}

// addForeignKeys configures foreign keys for Bind and Application tables
func addForeignKeys() {
	db, err := dbCon()
	if err != nil {
		log.HandleErrorWithLoggerAndExit(ErrMsgUnableToOpenDBCon, err)
	}
	defer closeDBCon(db)
	err = db.Model(&Bind{}).
		AddForeignKey("instance_id", TableInstance+"(id)", "CASCADE",
			"CASCADE").Error
	if err != nil {
		log.HandleErrorWithLoggerAndExit(ErrMsgUnableToAddForeignKeys, err)
	}
	err = db.Model(&Application{}).
		AddForeignKey("id", TableInstance+"(apim_resource_id)", "CASCADE",
			"CASCADE").Error
	if err != nil {
		log.HandleErrorWithLoggerAndExit(ErrMsgUnableToAddForeignKeys, err)
	}
}

// CreateTables creates the tables and add foreign keys
func CreateTables() {
	CreateTable(&Instance{}, TableInstance)
	CreateTable(&Application{}, TableApplication)
	CreateTable(&Bind{}, TableBind)
	addForeignKeys()
}
