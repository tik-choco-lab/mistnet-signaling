package main

import (
	"fmt"

	"github.com/tik-choco-lab/mistnet-signaling/pkg"
	"github.com/tik-choco-lab/mistnet-signaling/pkg/logger"
)

func main() {
	logger.Init()
	config, err := pkg.LoadConfig()
	if err != nil {
		fmt.Println(err)
		return
	}
	server := pkg.NewMistServer(*config)
	server.Start()
}
