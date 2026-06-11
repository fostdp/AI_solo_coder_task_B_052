package lora

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"
)

type PacketHandler func(packet []byte, addr *net.UDPAddr)

type UDPServer struct {
	conn       *net.UDPConn
	addr       string
	handler    PacketHandler
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	bufferSize int
}

func NewUDPServer(addr string, handler PacketHandler, bufferSize int) *UDPServer {
	if bufferSize <= 0 {
		bufferSize = 4096
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &UDPServer{
		addr:       addr,
		handler:    handler,
		ctx:        ctx,
		cancel:     cancel,
		bufferSize: bufferSize,
	}
}

func (s *UDPServer) Start() error {
	addr, err := net.ResolveUDPAddr("udp", s.addr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	s.conn = conn

	s.wg.Add(1)
	go s.readLoop()

	log.Printf("[UDPServer] listening on %s", s.addr)
	return nil
}

func (s *UDPServer) readLoop() {
	defer s.wg.Done()

	buf := make([]byte, s.bufferSize)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		s.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, addr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			select {
			case <-s.ctx.Done():
				return
			default:
				log.Printf("[UDPServer] read error: %v", err)
				continue
			}
		}

		data := make([]byte, n)
		copy(data, buf[:n])

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handlePacket(data, addr)
		}()
	}
}

func (s *UDPServer) handlePacket(data []byte, addr *net.UDPAddr) {
	select {
	case <-s.ctx.Done():
		return
	default:
	}

	if s.handler != nil {
		s.handler(data, addr)
	}
}

type LoRaUDPPacket struct {
	PacketID   string                 `json:"packet_id"`
	DeviceType string                 `json:"device_type"`
	DeviceID   string                 `json:"device_id"`
	Timestamp  int64                  `json:"timestamp"`
	Data       map[string]interface{} `json:"data"`
}

func CreateLoRaPacketHandler(dedup *PacketDeduplicator, forwardURL string) PacketHandler {
	return func(raw []byte, addr *net.UDPAddr) {
		var packet LoRaUDPPacket
		if err := json.Unmarshal(raw, &packet); err != nil {
			log.Printf("[UDPServer] invalid packet from %s: %v", addr, err)
			return
		}

		if dedup != nil && packet.PacketID != "" {
			result := dedup.CheckPacket(packet.PacketID, packet.DeviceID, time.Unix(packet.Timestamp, 0))
			if result.IsDuplicate {
				return
			}
		}

		log.Printf("[UDPServer] received from %s: device=%s type=%s", addr, packet.DeviceID, packet.DeviceType)
	}
}

func (s *UDPServer) Stop() {
	s.cancel()
	if s.conn != nil {
		s.conn.Close()
	}

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("[UDPServer] stopped gracefully")
	case <-time.After(5 * time.Second):
		log.Println("[UDPServer] forced stop after timeout")
	}
}

func (s *UDPServer) Addr() string {
	if s.conn != nil {
		return s.conn.LocalAddr().String()
	}
	return s.addr
}
