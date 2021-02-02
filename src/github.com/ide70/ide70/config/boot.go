package config

import (
	"fmt"
	"github.com/ide70/ide70/server"
)

func LoadServer() *server.AppServer {
	app := server.LoadApplication("app")
	addr := ":7070"
	if app.Config["port"] != nil {
		addr = fmt.Sprintf(":%v", app.Config["port"])
	}
	secure := true
	if app.Config["secure"] != nil {
		secure = app.Config["secure"].(bool)
	}
	as := server.NewAppServer(addr, secure)
	as.SetApplication(app)
	return as
}
