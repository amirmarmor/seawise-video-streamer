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
	//p.addHandlers()
	cleanSigTerm := Produce()
	err := http.ListenAndServe(core.Config.Port, p.server.Router)
	if err != nil {
		log.Fatal(fmt.Sprintf("Server down: %v", err))
	}
	//go p.capt.Start()

	cleanSigTerm.WaitForTermination()
}

func (p *EntryPoint) buildBlocks() {
	var err error
	//p.manager, err = core.Produce()
	//if err != nil {
	//	panic(err)
	//}

	p.channels, err = channels.Create(5)
	if err != nil {
		panic(err)
	}

	p.server = server.Produce(p.channels)
}
