package app

import (
	"bytes"
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/store"
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
	Name        string
	Path        string
	URLString   string   // Application URL string
	URL         *url.URL // Application URL, parsed
	Description string
	Connectors  Connectors
	Access      user.Access
	Config      map[string]interface{}
}

type Connectors struct {
	MainDB *store.DatabaseContext
	FileContext *file.FileContext
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
		app.Config = dataxform.InterfaceMapToStringMap(tpUnitIf)
		app.Name = dataxform.SIMapGetByKeyAsString(app.Config, "name")
		app.Path = "/" + app.Name + "/"
		app.Access.LoginUnit = dataxform.SIMapGetByKeyAsString(app.Config, "loginUnit")
		connectors := dataxform.SIMapGetByKeyAsMap(app.Config, "connectors")
		mainDB := dataxform.SIMapGetByKeyAsMap(connectors, "mainDB")
		if len(mainDB) > 0 {
			app.Connectors.MainDB = &store.DatabaseContext{}
			app.Connectors.MainDB.Host = dataxform.SIMapGetByKeyAsString(mainDB, "host")
			app.Connectors.MainDB.Port = dataxform.SIMapGetByKeyAsInt(mainDB, "port")
			app.Connectors.MainDB.DBName = dataxform.SIMapGetByKeyAsString(mainDB, "dbName")
			app.Connectors.MainDB.User = dataxform.SIMapGetByKeyAsString(mainDB, "user")
			app.Connectors.MainDB.Password = dataxform.SIMapGetByKeyAsString(mainDB, "password")
		}
		app.Connectors.FileContext = &file.FileContext{}

		return app
	}

	logger.Error("Application", configFileName, "has invalid format")

	return nil
}
