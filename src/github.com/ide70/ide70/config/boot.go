package config

import (
	"fmt"
	"github.com/ide70/ide70/app"
	"github.com/ide70/ide70/server"
	"github.com/ide70/ide70/util/file"
	"github.com/ide70/ide70/util/log"
	"os"
	"path/filepath"
)

var logger = log.Logger{"app"}

func LoadServer() *server.AppServer {
	adjustWorkingDirectory()
	app := app.LoadApplication("app")
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

func adjustWorkingDirectory() {
	ex,_ := os.Executable();
	exPath := file.NormalizeFileName(filepath.Dir(ex))
	parentPath := file.TrimLastPathComponent(exPath)
	os.Chdir(parentPath)
	logger.Info("Server root set to", parentPath)
}
