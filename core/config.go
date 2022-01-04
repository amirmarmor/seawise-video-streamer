package core

import (
	"encoding/json"
	"github.com/namsral/flag"
	"www.seawise.com/client/log"
)

const DefaultLoop = 60

type Configuration = struct {
	Port        string
	BackendHost string
	BackendPort string
	VidsString  string
	Parallel    bool
	Retries     int
}

var Config Configuration

func InitFlags() {
	flag.StringVar(&Config.BackendHost, "behost", "localhost", "The backend host")
	flag.StringVar(&Config.BackendPort, "beport", "8080", "The backend port")
	flag.StringVar(&Config.Port, "port", ":4000", "port")
	flag.StringVar(&Config.VidsString, "vids", "[0,2]", "set known vid numbers")
	flag.BoolVar(&Config.Parallel, "parallel", true, "record videos in parallel or in a loop")
	flag.IntVar(&Config.Retries, "retries", 5000, "number of register retries")

	log.AddNotify(postParse)
}

func postParse() {
	marshal, err := json.Marshal(Config)
	if err != nil {
		log.Fatal("marshal config failed: %v", err)
	}

	log.V5("configuration loaded: %v", string(marshal))
}
