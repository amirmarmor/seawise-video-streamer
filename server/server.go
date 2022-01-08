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
	Backend    string
	DeviceInfo *DeviceInfo
	Server     *http.Server
	Channels   *channels.Channels
	Streamers  []*Streamer
	Platform   string
}

func Produce(channels *channels.Channels) *Server {
	attempt := 0
	backend := fmt.Sprintf("http://%v:%v", core.Config.BackendHost, core.Config.BackendPort)

	server := &Server{
		Backend:  backend,
		Channels: channels,
	}

	log.V5("REGISTERING DEVICE - " + server.Backend)

	for attempt < core.Config.Retries {
		err := server.Register(len(channels.Array))
		if err != nil {
			log.Warn(fmt.Sprintf("failed to register: %v", err))
			log.Warn(fmt.Sprintf("Retrying to register - attempt %v", strconv.Itoa(attempt)))
			attempt++
			time.Sleep(3 * time.Second)
		} else {
			attempt = core.Config.Retries + 1
		}
	}

	for i, channel := range server.Channels.Array {
		server.Streamers = append(server.Streamers, CreateStreamer(server.DeviceInfo.Port+i, channel.Queue))
	}

	router := mux.NewRouter()
	router.HandleFunc("/shutdown", server.ShutdownHandler)
	router.HandleFunc("/start/{ch}", server.StartHandler)
	router.HandleFunc("/stop/{num}", server.StopHandler)
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

	return server
}

func (s *Server) Register(channels int) error {
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
		Sn:       sn,
		Ip:       ip,
		Owner:    "echo",
		Channels: channels,
	}

	postBody, err := json.Marshal(s.DeviceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal register requets: %v", err)
	}

	log.V5(url)
	var body []byte
	body, err = s.post(url, postBody)
	if err != nil {
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
	var response string
	vars := mux.Vars(r)
	ch := vars["ch"]
	if ch == "" {
		log.Warn(fmt.Sprintf("invalid address"))
		sendErrorMessage(w)
	}

	channel, err := strconv.Atoi(ch)
	if err != nil {
		log.Warn(fmt.Sprintf("invalid address"))
		sendErrorMessage(w)
	}

	s.Channels.Start(channel)
	response = "starting..."

	_, err = w.Write([]byte(response))
	if err != nil {
		panic(err)
	}
}

func (s *Server) StopHandler(w http.ResponseWriter, r *http.Request) {
	var response string
	vars := mux.Vars(r)
	cam, err := strconv.Atoi(vars["num"])
	if err != nil || cam > s.DeviceInfo.Channels {
		response = fmt.Sprintf("Invalid camera number - %v", cam)
		log.Warn(response)
	} else {
		go s.Channels.Stop(cam)
		response = "stopping..."
	}
	_, err = w.Write([]byte(response))
	if err != nil {
		panic(err)
	}
}

func (s *Server) ShutdownHandler(w http.ResponseWriter, r *http.Request) {
	response := "Shutting down..."

	err := s.Channels.Close()
	if err != nil {
		log.Warn(fmt.Sprintf("failed to close - %v", err))
		sendErrorMessage(w)
		return
	}

	for _, streamer := range s.Streamers {
		streamer.Stop()
	}

	log.V5("HERERERRRR")
	//err = s.Channels.DetectCameras()
	//if err != nil {
	//	log.Warn(fmt.Sprintf("failed to re-detect cameras - %v", err))
	//	sendErrorMessage(w)
	//	return
	//}

	//err = s.Register(len(s.Channels.Array))
	//if err != nil {
	//	log.Warn(fmt.Sprintf("failed to re-register - %v", err))
	//	sendErrorMessage(w)
	//	return
	//}

	_, err = w.Write([]byte(response))
	if err != nil {
		panic(err)
	}

	time.Sleep(1 * time.Second)
	go func() {
		err = s.Server.Shutdown(context.Background())
		if err != nil {
			log.Fatal(fmt.Sprintf("Failed to shutdown server - %v", err))
		}
	}()
}

func sendErrorMessage(w http.ResponseWriter) {
	w.WriteHeader(500)
	_, err := w.Write([]byte("an error occured"))
	if err != nil {
		log.Warn("failed to write response")
	}
}
