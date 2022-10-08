package main

import (
	"github.com/ide70/ide70/config"
	"github.com/ide70/ide70/util/log"
)

func main() {
	as := config.LoadServer()
	as.Start()

	log.Info("finished")
}
