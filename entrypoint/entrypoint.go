package entrypoint

import (
	"fmt"
	"net/http"
	"www.seawise.com/client/channels"
	"www.seawise.com/client/core"
	"www.seawise.com/client/log"
	"www.seawise.com/client/server"
)

type EntryPoint struct {
	//manager  *core.ConfigManager
	channels *channels.Channels
	server   *server.Server
}

func (p *EntryPoint) Run() {
	core.InitFlags()
	log.InitFlags()

	log.ParseFlags()
	log.Info("Starting")

	p.buildBlocks()

	err := p.server.Server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(fmt.Sprintf("Server down: %v", err))
	}
	log.V5("Finished")
}

func (p *EntryPoint) buildBlocks() {
	var err error

	p.channels, err = channels.Create(5)
	if err != nil {
		panic(err)
	}

	p.server = server.Produce(p.channels)
}
