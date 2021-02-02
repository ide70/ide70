package server

import (
	"bytes"
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/user"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/url"
	"strings"
)

const APP_PATH = "ide70/app/"

type Application struct {
	Name        string
	Path        string
	URLString   string   // Application URL string
	URL         *url.URL // Application URL, parsed
	Description string
	//	Connectors  Connectors
	Access user.Access
	Config map[string]interface{}
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
		app.Name = app.Config["name"].(string)
		app.Path = "/" + app.Name + "/"
		app.Access.LoginUnit = app.Config["loginUnit"].(string)
		return app
	}

	logger.Error("Application", configFileName, "has invalid format")

	return nil
}

func (app *Application) registerServer(server *AppServer) {
	addr := server.Addr
	if strings.HasPrefix(addr, ":") {
		addr = "localhost" + addr
	}
	if server.Secure {
		app.URLString = "https://" + addr + app.Path
	} else {
		app.URLString = "http://" + addr + app.Path
	}

	var err error
	if app.URL, err = url.Parse(app.URLString); err != nil {
		logger.Error("Parse", app.URLString, err)
	}
}
