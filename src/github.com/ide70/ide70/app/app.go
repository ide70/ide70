package app

import (
	"bytes"
	"github.com/ide70/ide70/api"
	"github.com/ide70/ide70/user"
	"github.com/ide70/ide70/util/file"
	"github.com/ide70/ide70/util/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/url"
)

const APP_PATH = "ide70/app/"

var logger = log.Logger{"app"}

type Application struct {
	Name           string
	Path           string
	URLString      string   // Application URL string
	URL            *url.URL // Application URL, parsed
	Description    string
	Connectors     Connectors
	Access         user.Access
	Config         map[string]interface{}
	configFileName string
}

type Connectors struct {
	MainDB      *api.DatabaseContext
	FileContext *file.FileContext
	LoadContext *api.LoadContext
}

func NewApplication(appName string) *Application {
	app := &Application{}
	app.Name = appName
	app.Path = "/" + appName + "/"
	app.Config = map[string]interface{}{}
	return app
}

func LoadApplication(configFileName string) *Application {
	contentB, err := ioutil.ReadFile(APP_PATH + configFileName + ".yaml")
	if err != nil {
		logger.Error("Application", configFileName, "not found")
		return nil
	}

	decoder := yaml.NewDecoder(bytes.NewReader(contentB))

	var unitIf interface{}
	err = decoder.Decode(&unitIf)
	if err != nil {
		logger.Error("Application", configFileName, "failed to decode:", err.Error())
	}

	switch tpUnitIf := unitIf.(type) {
	case map[interface{}]interface{}:
		app := &Application{}
		app.Config = api.InterfaceMapToStringMap(tpUnitIf)
		app.Name = api.SIMapGetByKeyAsString(app.Config, "name")
		app.Path = "/" + app.Name + "/"
		loginUnitsArr := api.SIMapGetByKeyAsList(app.Config, "loginUnits")
		app.Access.LoginUnits = map[string]bool{}
		for _, loginUnitIf := range loginUnitsArr {
			loginUnitData := api.IAsSIMap(loginUnitIf)
			app.Access.LoginUnits[api.IAsString(loginUnitData["path"])] = true
		}
		connectors := api.SIMapGetByKeyAsMap(app.Config, "connectors")
		mainDB := api.SIMapGetByKeyAsMap(connectors, "mainDB")
		if len(mainDB) > 0 {
			app.Connectors.MainDB = &api.DatabaseContext{}
			app.Connectors.MainDB.Host = api.SIMapGetByKeyAsString(mainDB, "host")
			app.Connectors.MainDB.Port = api.SIMapGetByKeyAsInt(mainDB, "port")
			app.Connectors.MainDB.DBName = api.SIMapGetByKeyAsString(mainDB, "dbName")
			app.Connectors.MainDB.User = api.SIMapGetByKeyAsString(mainDB, "user")
			app.Connectors.MainDB.Password = api.SIMapGetByKeyAsString(mainDB, "password")
		}
		app.Connectors.FileContext = &file.FileContext{}
		app.Connectors.LoadContext = &api.LoadContext{}
		app.configFileName = configFileName

		log.ResetKeyLevels()
		logProps := api.SIMapGetByKeyAsMap(app.Config, "log")
		for logCategory, logCatgPropMapIf := range logProps {
			logCatgPropMap := api.IAsSIMap(logCatgPropMapIf)
			logLevelStr := api.SIMapGetByKeyAsString(logCatgPropMap, "level")
			logLevel := log.NameToLevel(logLevelStr)
			logCategoryInternal := logCategory
			if logCategoryInternal == "common" {
				logCategoryInternal = ""
			}
			if logLevel != -1 {
				if log.SetKeyLevel(logCategoryInternal, logLevel) {
					logger.Info("Log level of", logCategory, "set to:", logLevelStr)
				}
			}
		}
		return app
	}

	logger.Error("Application", configFileName, "has invalid format")

	return nil
}

func (a *Application) ReconfigureApplication() {
	logger.Info("Reload Application Config...")
	newApp := LoadApplication(a.configFileName)
	a.Config = newApp.Config
	a.Access.LoginUnits = newApp.Access.LoginUnits
	a.Connectors.MainDB = newApp.Connectors.MainDB
}
