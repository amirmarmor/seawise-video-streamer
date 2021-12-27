package server

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/gorilla/mux"
	"net"
	"time"
	"www.seawise.com/client/log"
)

type Streamer struct {
	TCPConn                 *net.TCPConn
	port                    int
	timeStampPacketSize     uint
	contentLengthPacketSize uint
	Router                  *mux.Router
	Queue                   *chan []byte
}

func CreateStreamer(port int) *Streamer {
	q := make(chan []byte)
	streamer := &Streamer{
		Queue:                   &q,
		port:                    port,
		timeStampPacketSize:     8,
		contentLengthPacketSize: 8,
	}

	streamer.connect()
	return streamer
}

func (s *Streamer) connect() {
	log.V5(fmt.Sprintf("opening socket on port: %v", s.port))
	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: s.port,
	})

	if err != nil {
		log.Warn(fmt.Sprintf("generate udp client failed! - %v", err))
		time.Sleep(time.Second * 3)
		go s.connect()
		return
	}

	s.TCPConn = conn
	go s.handleSend()
	return
}

func (s *Streamer) handleSend() {
	writer := bufio.NewWriter(s.TCPConn)
	for pkt := range *s.Queue {
		_, err := writer.Write(s.pack(pkt))
		if err != nil {
			log.Warn(fmt.Sprintf("Packet Send Failed! - %v", err))
			go s.connect()
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
	err := s.TCPConn.Close()
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to close socket: %v", err))
	}
}
