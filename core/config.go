package core

import (
	"encoding/json"
	"github.com/namsral/flag"
	"www.seawise.com/client/log"
)

type Configuration = struct {
	Port        string
	StreamHost  string
	StreamPort  int
	BackendHost string
	BackendPort string
}

var Config Configuration

func InitFlags() {
	flag.StringVar(&Config.BackendHost, "backend-host", "localhost", "The backend host")
	flag.StringVar(&Config.BackendPort, "backend-port", "5000", "The backend port")
	flag.StringVar(&Config.StreamHost, "stream-host", "localhost", "The stream host")
	flag.IntVar(&Config.StreamPort, "stream-port", 8000, "The stream port")
	flag.StringVar(&Config.Port, "port", ":3000", "port")

	log.AddNotify(postParse)
}

func postParse() {
	marshal, err := json.Marshal(Config)
	if err != nil {
		log.Fatal("marshal config failed: %v", err)
	}

	log.V5("configuration loaded: %v", string(marshal))
}
