package pkg

import (
	"encoding/json"
	"sync"

	"github.com/tik-choco-lab/mistnet-signaling/pkg/logger"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type NodeId string
type SignalingType string

const (
	SignalingTypeRequest SignalingType = "Request"
)

type SignalingData struct {
	Type       SignalingType `json:"Type"`
	Data       string        `json:"Data,omitempty"`
	SenderId   NodeId        `json:"SenderId"`
	ReceiverId NodeId        `json:"ReceiverId"`
	RoomId     string        `json:"RoomId"`
}

type NodeIdWithData struct {
	NodeId NodeId
	Data   SignalingData
}

type SafeConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (sc *SafeConn) Write(msgType int, data []byte) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.conn.WriteMessage(msgType, data)
}

type SignalingService struct {
	server        *MistServer
	config        MistConfig
	nodeToConn    map[NodeId]*SafeConn
	sessionToNode map[string]NodeId
	roomNodesMu   sync.Mutex
	roomNodes     map[string][]NodeId
	nodeToRoom    map[NodeId]string
	nodeToConnMu  sync.RWMutex
	sessionMu     sync.RWMutex
}

func NewSignalingServer(server *MistServer, config MistConfig) *SignalingService {
	return &SignalingService{
		server:        server,
		config:        config,
		nodeToConn:    make(map[NodeId]*SafeConn),
		sessionToNode: make(map[string]NodeId),
		roomNodes:     make(map[string][]NodeId),
		nodeToRoom:    make(map[NodeId]string),
	}
}

func (s *SignalingService) HandleMessage(conn *websocket.Conn, sessionID string, message []byte) {
	logger.Debug("[SERVER][RECV]", zap.String("message", string(message)))

	var data SignalingData
	if err := json.Unmarshal(message, &data); err != nil {
		logger.Debug("[SERVER][ERROR]", zap.Error(err))
		return
	}

	// sessionIdを保存（並行安全）
	s.sessionMu.Lock()
	s.sessionToNode[sessionID] = data.SenderId
	s.sessionMu.Unlock()

	s.nodeToConnMu.Lock()
	if _, ok := s.nodeToConn[data.SenderId]; !ok {
		s.nodeToConn[data.SenderId] = &SafeConn{conn: conn}
	}
	s.nodeToConnMu.Unlock()

	if data.Type == SignalingTypeRequest {
		addNodeToRoom(s, data)

		s.roomNodesMu.Lock()
		roomNodesLen := len(s.roomNodes[data.RoomId])
		s.roomNodesMu.Unlock()

		if roomNodesLen >= 2 {
			s.sendRequest(data.RoomId)
		}
		return
	}

	s.send(data.ReceiverId, message)
}

func addNodeToRoom(s *SignalingService, data SignalingData) {
	s.roomNodesMu.Lock()
	defer s.roomNodesMu.Unlock()

	if _, ok := s.roomNodes[data.RoomId]; !ok {
		s.roomNodes[data.RoomId] = make([]NodeId, 0)
	}
	for _, node := range s.roomNodes[data.RoomId] {
		// 既にある場合は追加しない
		if node == data.SenderId {
			return
		}
	}
	roomNodes := s.roomNodes[data.RoomId]
	roomNodes = append(roomNodes, data.SenderId)
	s.nodeToRoom[data.SenderId] = data.RoomId
	s.roomNodes[data.RoomId] = roomNodes
}

func (s *SignalingService) send(nodeID NodeId, message []byte) {
	logger.Debug("[SERVER][SEND]", zap.String("nodeID", string(nodeID)))

	s.nodeToConnMu.RLock()
	sc, ok := s.nodeToConn[nodeID]
	s.nodeToConnMu.RUnlock()
	if !ok {
		return
	}
	_ = sc.Write(websocket.TextMessage, message)
}

func (s *SignalingService) sendRequest(roomId string) {
	s.roomNodesMu.Lock()
	defer s.roomNodesMu.Unlock()

	nodes := s.roomNodes[roomId]
	count := len(nodes)

	if count == 2 {
		// 1人目: A-B
		s.sendPair(nodes[0], nodes[1], roomId)
	} else if count >= 3 {
		// 新しい人と直前の2人をペア
		newNode := nodes[count-1]
		prev1 := nodes[count-2]
		prev2 := nodes[count-3]

		s.sendPair(prev2, newNode, roomId)
		s.sendPair(prev1, newNode, roomId)
	}
}

func (s *SignalingService) sendPair(nodeA, nodeB NodeId, roomId string) {
	nodeAData := SignalingData{
		Type:       SignalingTypeRequest,
		ReceiverId: nodeA,
		SenderId:   nodeB,
		RoomId:     roomId,
	}
	nodeBData := SignalingData{
		Type:       SignalingTypeRequest,
		ReceiverId: nodeB,
		SenderId:   nodeA,
		RoomId:     roomId,
	}

	nodeAJSON, _ := json.Marshal(nodeAData)
	nodeBJSON, _ := json.Marshal(nodeBData)

	s.send(nodeA, nodeAJSON)
	s.send(nodeB, nodeBJSON)
}

func (s *SignalingService) HandleClose(conn *websocket.Conn, sessionID string) {
	s.sessionMu.RLock()
	nodeID, exists := s.sessionToNode[sessionID]
	s.sessionMu.RUnlock()

	if exists {
		s.roomNodesMu.Lock()
		roomId := s.nodeToRoom[nodeID]
		s.roomNodesMu.Unlock()

		s.removeNodeFromRoom(nodeID, roomId)

		s.nodeToConnMu.Lock()
		delete(s.nodeToConn, nodeID)
		s.nodeToConnMu.Unlock()
	}

	s.sessionMu.Lock()
	delete(s.sessionToNode, sessionID)
	s.sessionMu.Unlock()
}

func (s *SignalingService) removeNodeFromRoom(nodeID NodeId, roomId string) {
	s.roomNodesMu.Lock()
	defer s.roomNodesMu.Unlock()

	delete(s.nodeToRoom, nodeID)

	// 全nodeを表示
	logger.Debug("[SERVER][REMOVE NODE FROM ROOM]", zap.String("roomId", roomId))
	logger.Debug("[SERVER][REMOVE NODE FROM ROOM]", zap.Any("nodes", s.roomNodes[roomId]))

	nodes := s.roomNodes[roomId]
	for i, node := range nodes {
		if node == nodeID {
			s.roomNodes[roomId] = append(nodes[:i], nodes[i+1:]...)
			break
		}
	}

	if len(s.roomNodes[roomId]) == 0 {
		delete(s.roomNodes, roomId)
	}

	// 全nodeを表示
	logger.Debug("[SERVER][REMOVE NODE FROM ROOM]", zap.String("roomId", roomId))
	logger.Debug("[SERVER][REMOVE NODE FROM ROOM]", zap.Any("nodes", s.roomNodes[roomId]))
}
