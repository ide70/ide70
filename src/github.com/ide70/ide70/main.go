package main

import (
	"github.com/ide70/ide70/config"
	"github.com/ide70/ide70/util/log"
	"os"
	//"fmt"
)

func main() {
	log.SetKeyLevel("", log.INFO)
	log.SetKeyLevel("comp", log.INFO)

	cwd, err := os.Getwd()
	if err != nil {
		log.Error(err.Error())
		return
	}
	log.Info("ide70 serve at", cwd)

	as := config.LoadServer()
	as.Start()

	log.Info("finished")
}
