package server

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/gorilla/mux"
	"net"
	"time"
	"www.seawise.com/client/core"
	"www.seawise.com/client/log"
)

type Streamer struct {
	TCPConn                 *net.TCPConn
	port                    int
	timeStampPacketSize     uint
	contentLengthPacketSize uint
	Router                  *mux.Router
	Queue                   *chan []byte
	Problems                *chan string
}

func CreateStreamer(q *chan []byte, problems *chan string) *Streamer {
	streamer := &Streamer{
		timeStampPacketSize:     8,
		contentLengthPacketSize: 8,
		Queue:                   q,
		Problems:                problems,
	}

	return streamer
}

func (s *Streamer) Connect(port int) {
	s.port = port
	log.V5(fmt.Sprintf("opening socket on port: %v", s.port))
	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{
		IP:   net.ParseIP(core.Config.BackendHost),
		Port: s.port,
	})

	if err != nil {
		log.Warn(fmt.Sprintf("generate udp client failed! - %v", err))
		time.Sleep(time.Second * 3)
		*s.Problems <- "disconnect"
		return
	}

	s.TCPConn = conn
	go s.handleSend()
	log.V5("done connecting")
	return
}

func (s *Streamer) handleSend() {
	writer := bufio.NewWriter(s.TCPConn)
	for pkt := range *s.Queue {
		_, err := writer.Write(s.pack(pkt))
		if err != nil {
			log.Warn(fmt.Sprintf("Packet Send Failed! - %v", err))
			go s.Connect(s.port)
			return
		}
	}
}

func (s *Streamer) pack(frame []byte) []byte {
	// ------ Packet ------
	// timestamp (8 bytes)
	// content-length (8 bytes)
	// content (content-length bytes)
	// ------  End   ------

	timePkt := make([]byte, s.timeStampPacketSize)
	binary.LittleEndian.PutUint64(timePkt, uint64(time.Now().UnixNano()))

	contentLengthPkt := make([]byte, s.contentLengthPacketSize)
	binary.LittleEndian.PutUint64(contentLengthPkt, uint64(len(frame)))

	var pkt []byte
	pkt = append(pkt, timePkt...)
	pkt = append(pkt, contentLengthPkt...)
	pkt = append(pkt, frame...)

	return pkt
}

func (s *Streamer) Stop() {
	if s.TCPConn != nil {
		err := s.TCPConn.Close()
		if err != nil {
			log.Warn(fmt.Sprintf("Failed to close socket: %v", err))
		}
	}
}
