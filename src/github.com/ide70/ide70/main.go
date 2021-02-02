package main

import (
	"github.com/ide70/ide70/config"
	"github.com/ide70/ide70/server"
	"github.com/ide70/ide70/util/log"
	//"github.com/ide70/ide70/comp"
	"os"
	//"fmt"
)

func main() {
	log.SetKeyLevel("", log.INFO)
	log.SetKeyLevel("comp", log.ERROR)
	cwd, err := os.Getwd()
	if err != nil {
		log.Error(err.Error())
		return
	}
	log.Info("ide70 serve at", cwd)
	server.Cwd = cwd

	as := config.LoadServer()
	as.Start()

	log.Info("finished")
}
