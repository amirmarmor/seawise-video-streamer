package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"www.seawise.com/client/channels"
	"www.seawise.com/client/core"
	"www.seawise.com/client/log"
)

type DeviceInfo struct {
	Sn       string `json:"sn"`
	Owner    string `json:"owner"`
	Ip       string `json:"ip"`
	Channels int    `json:"channels"`
	Loop     int    `json:"loop"`
	Port     int    `json:"port"`
}

type Server struct {
	Started     bool
	Registering bool
	Backend     string
	DeviceInfo  *DeviceInfo
	Server      *http.Server
	Channels    *channels.Channels
	Streamers   []*Streamer
	Platform    string
	Problems    chan string
	ticker      *time.Ticker
	health      bool
}

func Produce(chs *channels.Channels) *Server {
	backend := fmt.Sprintf("http://%v:%v", core.Config.BackendHost, core.Config.BackendPort)

	server := &Server{
		Backend:  backend,
		Problems: make(chan string),
		ticker:   time.NewTicker(1 * time.Second),
		Channels: chs,
	}

	log.V5("REGISTERING DEVICE - " + server.Backend)

	router := mux.NewRouter()
	router.HandleFunc("/start/{port}", server.StartHandler).Methods("GET")
	router.HandleFunc("/action/{action}", server.ActionHandler).Methods("GET")
	router.HandleFunc("/getchannels", server.GetChannelsHandler).Methods("GET")
	router.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		_, err := writer.Write([]byte("ok"))
		if err != nil {
			log.Warn(fmt.Sprintf("unable to write response on health check: %v", err))
		}
	})

	server.Server = &http.Server{
		Addr:    core.Config.Port,
		Handler: router,
	}

	server.TryRegister()

	return server
}

func (s *Server) handleHealthCheck() {
	s.health = true
	for s.health {
		select {
		case <-s.ticker.C:
			err := s.checkHealth()
			if err != nil {
				s.gracefullyShutdown()
				s.TryRegister()
			}
		}
	}
}

func (s *Server) checkHealth() error {
	url := s.Backend + "/health/" + s.DeviceInfo.Sn
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed health check: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed health check: %v", err)
	}

	if string(body) != "ok" {
		return fmt.Errorf("failed health check: %v", err)
	}

	return nil
}

func (s *Server) Register() error {
	url := s.Backend + "/register"
	err := s.getPlatform()
	if err != nil {
		return fmt.Errorf("failed to register: %v", err)
	}

	ip, err := s.getIp()
	if err != nil {
		return fmt.Errorf("failed to register: %v", err)
	}

	sn, err := s.getSN()
	if err != nil {
		return fmt.Errorf("failed to register: %v", err)
	}

	s.DeviceInfo = &DeviceInfo{
		Sn:    sn,
		Ip:    ip,
		Owner: "echo",
	}

	postBody, err := json.Marshal(s.DeviceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal register requets: %v", err)
	}

	log.V5(url)
	var body []byte
	body, err = s.post(url, postBody)
	if err != nil {
		log.Warn(fmt.Sprintf("got error on register: %v", err))
		return fmt.Errorf("failed to register device no connectivity: %v", err)
	}

	err = json.Unmarshal(body, s.DeviceInfo)
	if err != nil {
		return fmt.Errorf("failed to unmarshal register response: %v", err)
	}

	return nil
}

func (s *Server) post(url string, postBody []byte) ([]byte, error) {
	respBody := bytes.NewBuffer(postBody)
	resp, err := http.Post(url, "application/json", respBody)
	if err != nil {
		return nil, fmt.Errorf("failed to post: %v", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to post: %v", err)
	}

	return body, nil
}

func (s *Server) getPlatform() error {
	out, err := exec.Command("/bin/sh", "-c", "uname -m").Output()
	if err != nil {
		return fmt.Errorf("failed to identify platform: %v", err)
	}
	platform := strings.ReplaceAll(string(out), "\n", "")
	if platform == "aarch64" || platform == "armv7l" {
		s.Platform = "pi"
	} else {
		s.Platform = "other"
	}
	log.V5(fmt.Sprintf("MY PLATFORM IS - %v", s.Platform))
	return nil
}

func (s *Server) getSN() (string, error) {
	log.V5("GETTING SERIAL NUMBER")
	var out *exec.Cmd
	if s.Platform == "pi" {
		out = exec.Command("/bin/sh", "-c", "sudo cat /proc/cpuinfo | grep Serial | cut -d ' ' -f 2")
	} else {
		out = exec.Command("/bin/sh", "-c", "sudo cat /sys/class/dmi/id/board_serial")
	}
	res, err := out.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get S/N: %v", err)
	}
	sn := strings.ReplaceAll(string(res), "\n", "")
	log.V5(fmt.Sprintf("SERIAL NUMBER IS - %v", sn))
	return sn, nil
}

func (s *Server) getIp() (string, error) {
	if s.Platform != "pi" {
		return "127.0.0.1", nil
	}

	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("failed to get IP: %v", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String(), nil
}

func (s *Server) StartHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portString := vars["port"]

	port, err := strconv.Atoi(portString)
	if err != nil {
		log.Warn(fmt.Sprintf("invalid port %v: %v", portString, err))
		sendErrorMessage(w)
		return
	}

	if port < core.Config.BackendPort+10 {
		log.Warn(fmt.Sprintf("invalid port %v: %v", port, err))
		sendErrorMessage(w)
		return
	}

	response := "Already started"
	if s.Started {
		_, err = w.Write([]byte(response))
		if err != nil {
			panic(err)
		}
		return
	}

	response = "ok"
	s.Started = true

	for i, channel := range s.Channels.Array {
		streamer := CreateStreamer(channel.Queue, &s.Problems)
		streamer.Connect(port + i)
		s.Streamers = append(s.Streamers, CreateStreamer(streamer.Queue, &s.Problems))
	}

	go s.Channels.Start()

	_, err = w.Write([]byte(response))
	if err != nil {
		panic(err)
	}
}

func (s *Server) ActionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	action := vars["action"]

	go s.gracefullyShutdown()
	s.health = false

	if action == "restart" {
		time.Sleep(3 * time.Second)
		go s.TryRegister()
	}

	if action == "shutdown" {
		go func() {
			time.Sleep(1 * time.Second)
			err := s.Server.Shutdown(context.Background())
			if err != nil {
				log.Fatal(fmt.Sprintf("Failed to shutdown server - %v", err))
			}
		}()
	}

	if action == "reboot" {
		go func() {
			log.V5("Rebooting....")
			time.Sleep(2 * time.Second)
			exec.Command("sudo reboot")
		}()
	}

	response := fmt.Sprintf("Did %v", action)
	_, err := w.Write([]byte(response))
	if err != nil {
		panic(err)
	}
}

func (s *Server) GetChannelsHandler(w http.ResponseWriter, r *http.Request) {
	err := s.Channels.DetectCameras()
	if err != nil {
		panic("failed to create channels!!!!!")
	}

	_, err = w.Write([]byte(strconv.Itoa(len(s.Channels.Array))))
	if err != nil {
		panic(err)
	}
}

func (s *Server) handleProblems() {
	for s.health {
		select {
		case problem := <-s.Problems:
			go s.problemHandler(problem)
		}
	}
}

func (s *Server) problemHandler(problem string) {
	log.V5("Problem - %v", problem)
}

func (s *Server) TryRegister() {
	attempt := 0
	trying := true

	for trying {
		if attempt > core.Config.Retries {
			trying = false
			panic(fmt.Sprintf("failed to register after all attempts, stopping"))
		}

		err := s.Register()
		if err != nil {
			log.Warn(fmt.Sprintf("failed to register: %v", err))
			log.Warn(fmt.Sprintf("Retrying to register - attempt %v", strconv.Itoa(attempt)))
			attempt++
			time.Sleep(3 * time.Second)
		} else {
			trying = false
		}
	}

	go s.handleHealthCheck()
	go s.handleProblems()

	return
}

func (s *Server) gracefullyShutdown() {
	if s.Started {
		s.Started = false
		s.Channels.StopChannel <- "stop"
		go func() {
			time.Sleep(1 * time.Second)
			s.Channels.Close()
		}()
	}

	if len(s.Streamers) > 0 {
		for _, streamer := range s.Streamers {
			streamer.Stop()
		}
		s.Streamers = s.Streamers[:0]
	}

	s.health = false
	log.V5("shut down complete")
}

func sendErrorMessage(w http.ResponseWriter) {
	w.WriteHeader(500)
	_, err := w.Write([]byte("an error occured"))
	if err != nil {
		log.Warn("failed to write response")
	}
}
