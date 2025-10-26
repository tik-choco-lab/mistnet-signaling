package pkg

import (
	"fmt"
	"log"
	"net/http"

	"github.com/tik-choco-lab/mistnet-signaling/pkg/logger"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type MistServer struct {
	config           MistConfig
	upgrader         websocket.Upgrader
	signalingService *SignalingService
}

func NewMistServer(config MistConfig) *MistServer {
	server := &MistServer{
		config: config,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 全オリジンからの接続を許可
			},
		},
		signalingService: nil,
	}
	signalingService := NewSignalingServer(server, config)
	server.signalingService = signalingService
	return server

}

func (s *MistServer) Start() {
	if !s.config.GlobalNode.Enable {
		return
	}

	http.HandleFunc("/signaling", s.handleWebSocket)

	port := s.config.GlobalNode.Port
	addr := fmt.Sprintf(":%d", port)
	logger.Debug("[MistSignalingServer] Start", zap.Int("port", port))
	log.Fatal(http.ListenAndServe(addr, nil))
}

func (s *MistServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("PANIC recovered", zap.Any("error", r), zap.Stack("stack"))
		}
	}()

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	sessionID := conn.RemoteAddr().String()
	logger.Debug("[SERVER][OPEN]", zap.String("sessionID", sessionID))

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			logger.Debug("[SERVER][CLOSE]", zap.String("sessionID", sessionID))
			s.handleClose(conn, sessionID)
			break
		}

		s.signalingService.HandleMessage(conn, sessionID, message)
	}
}

func (s *MistServer) handleClose(conn *websocket.Conn, sessionID string) {
	s.signalingService.HandleClose(conn, sessionID)
}
