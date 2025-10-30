package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"kahoot-game/internal/models"
	"kahoot-game/internal/services"
)

// Hub WebSocket 連線管理中心
type Hub struct {
	// 註冊的客戶端連線
	clients map[*Client]bool

	// 按房間分組的客戶端
	rooms map[string]map[*Client]bool

	// 客戶端註冊通道
	register chan *Client

	// 客戶端註銷通道
	unregister chan *Client

	// 廣播訊息通道
	broadcast chan []byte

	// 房間廣播通道
	roomBroadcast chan *RoomMessage

	// 服務層依賴
	roomService *services.RoomService
	gameService *services.GameService

	// 互斥鎖
	mutex sync.RWMutex
}

// RoomMessage 房間訊息結構
type RoomMessage struct {
	RoomID  string `json:"roomId"`
	Message []byte `json:"message"`
}

// NewHub 創建新的 Hub
func NewHub(roomService *services.RoomService, gameService *services.GameService) *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		rooms:         make(map[string]map[*Client]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		broadcast:     make(chan []byte),
		roomBroadcast: make(chan *RoomMessage),
		roomService:   roomService,
		gameService:   gameService,
	}
}

// Run 啟動 Hub
func (h *Hub) Run() {
	log.Println("🚀 WebSocket Hub 已啟動")

	for {
		select {
		case client := <-h.register:
			log.Printf("🔄 Hub 收到註冊請求: %s", client.ID)
			h.registerClient(client)
			log.Printf("✅ Hub 註冊完成: %s", client.ID)

		case client := <-h.unregister:
			log.Printf("🔄 Hub 收到註銷請求: %s", client.ID)
			h.unregisterClient(client)
			log.Printf("❌ Hub 註銷完成: %s", client.ID)

		case message := <-h.broadcast:
			log.Printf("📡 Hub 處理全域廣播")
			h.broadcastToAll(message)

		case roomMsg := <-h.roomBroadcast:
			log.Printf("📡 Hub 處理房間廣播: 房間=%s", roomMsg.RoomID)
			h.broadcastToRoom(roomMsg.RoomID, roomMsg.Message)
		}
	}
}

// registerClient 註冊客戶端
func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.clients[client] = true

	log.Printf("✅ 客戶端已註冊: %s (總計: %d)", client.ID, len(h.clients))

	// 發送歡迎訊息
	welcomeMsg := Message{
		Type: "CONNECTED",
		Data: map[string]interface{}{
			"clientId": client.ID,
			"message":  "歡迎來到 Ricky 遊戲小舖！",
		},
	}

	if msgBytes, err := json.Marshal(welcomeMsg); err == nil {
		client.send <- msgBytes
	}
}

// unregisterClient 註銷客戶端
func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, ok := h.clients[client]; ok {
		// 1. 先處理離開邏輯（在關閉通道前）
		if client.RoomID != "" {
			h.handlePlayerLeaveInternal(client)
		}

		// 2. 從特定房間移除（而不是遍歷所有房間）
		if client.RoomID != "" {
			h.removeClientFromRoom(client, client.RoomID)
		}

		// 3. 從全域客戶端列表移除
		delete(h.clients, client)

		// 4. 最後關閉通道
		close(client.send)

		log.Printf("❌ 客戶端已註銷: %s (剩餘: %d)", client.ID, len(h.clients))
	} else {
		log.Printf("⚠️ 嘗試註銷不存在的客戶端: %s", client.ID)
	}
}

// addClientToRoom 將客戶端加入房間
func (h *Hub) AddClientToRoom(client *Client, roomID string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[*Client]bool)
	}

	h.rooms[roomID][client] = true
	client.RoomID = roomID

	log.Printf("🏠 客戶端 %s 加入房間 %s", client.ID, roomID)
}

// removeClientFromRoom 從房間移除客戶端
func (h *Hub) removeClientFromRoom(client *Client, roomID string) {
	if h.rooms[roomID] != nil {
		delete(h.rooms[roomID], client)

		// 如果房間沒有客戶端了，刪除房間
		if len(h.rooms[roomID]) == 0 {
			delete(h.rooms, roomID)
			log.Printf("🗑️ 房間 %s 已清空並移除", roomID)
		}
	}
}

// broadcastToAll 廣播給所有客戶端
func (h *Hub) broadcastToAll(message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			delete(h.clients, client)
			close(client.send)
		}
	}
}

// BroadcastToRoom 廣播給特定房間
func (h *Hub) BroadcastToRoom(roomID string, message []byte) {
	h.roomBroadcast <- &RoomMessage{
		RoomID:  roomID,
		Message: message,
	}
}

// broadcastToRoom 內部廣播給房間
func (h *Hub) broadcastToRoom(roomID string, message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if roomClients, exists := h.rooms[roomID]; exists {
		for client := range roomClients {
			select {
			case client.send <- message:
			default:
				delete(roomClients, client)
				close(client.send)
			}
		}
	}
}

// broadcastToRoomExclude 內部廣播給房間（排除指定客戶端）
func (h *Hub) broadcastToRoomExclude(roomID string, message []byte, excludeClient *Client) {
	if roomClients, exists := h.rooms[roomID]; exists {
		for client := range roomClients {
			// 跳過已離開的客戶端
			if client == excludeClient {
				continue
			}

			select {
			case client.send <- message:
			default:
				// 如果發送失敗，標記為需要清理（但不在這裡直接刪除，避免併發問題）
				log.Printf("⚠️ 向客戶端 %s 發送消息失敗", client.ID)
			}
		}
	}
}

// handlePlayerLeave 處理玩家離開（外部調用）
func (h *Hub) handlePlayerLeave(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.handlePlayerLeaveInternal(client)
}

// handlePlayerLeaveInternal 處理玩家離開（內部調用，已持有鎖）
func (h *Hub) handlePlayerLeaveInternal(client *Client) {
	if client.RoomID == "" || client.PlayerName == "" {
		return
	}

	log.Printf("👋 處理玩家離開: %s 從房間 %s", client.PlayerName, client.RoomID)

	roomClients := h.rooms[client.RoomID]

	// 從房間服務中移除玩家
	if err := h.roomService.RemovePlayer(client.RoomID, client.ID); err != nil {
		log.Printf("❌ 移除玩家失敗: %v", err)
		return
	}

	// 獲取更新後的房間資訊
	room, err := h.roomService.GetRoom(client.RoomID)
	if err != nil {
		log.Printf("❌ 獲取房間資訊失敗: %v", err)

		// 房間可能已被清空，仍需通知其他客戶端
		leaveMsg := Message{
			Type: "PLAYER_LEFT",
			Data: map[string]interface{}{
				"playerId":     client.ID,
				"playerName":   client.PlayerName,
				"totalPlayers": 0,
				"players":      []*models.Player{},
				"currentHost":  "",
				"hostChanged":  false,
				"resetAnswers": true,
			},
		}

		if msgBytes, marshalErr := json.Marshal(leaveMsg); marshalErr == nil {
			h.broadcastToRoomExclude(client.RoomID, msgBytes, client)
		}

		// 通知遊戲結束
		if len(roomClients) > 0 {
			finishMsg := Message{
				Type: "GAME_FINISHED",
				Data: map[string]interface{}{
					"message": "所有玩家已離開，遊戲結束",
				},
			}
			if msgBytes, marshalErr := json.Marshal(finishMsg); marshalErr == nil {
				h.broadcastToRoomExclude(client.RoomID, msgBytes, nil)
			}
		}

		return
	}

	// 清除離開玩家的答案
	if room.Answers != nil {
		delete(room.Answers, client.ID)
	}

	remainingPlayers := room.GetPlayerCount()
	hostChanged := false
	resetAnswers := false
	shouldSkipCurrentQuestion := false
	var nextClient *Client

	if remainingPlayers > 0 {
		// 如果當前主角不存在或就是離開者，選擇新的主角
		currentHostMissing := room.CurrentHost == "" || room.Players[room.CurrentHost] == nil
		if currentHostMissing {
			if newHost := h.gameService.SelectNextHost(room, client.ID); newHost != "" {
				room.CurrentHost = newHost
				room.NextHostOverride = newHost
				hostChanged = true
			}
		}

		// 如果下一題預設主角是離開者，重新選擇
		if room.NextHostOverride == client.ID {
			if newOverride := h.gameService.SelectNextHost(room, client.ID); newOverride != "" {
				room.NextHostOverride = newOverride
				room.CurrentHost = newOverride
				hostChanged = true
			} else {
				room.NextHostOverride = ""
			}
		}

		if hostChanged {
			if len(room.Answers) > 0 {
				room.Answers = make(map[string]*models.Answer)
			}
			resetAnswers = true

			if room.Status == models.RoomStatusQuestionDisplay {
				shouldSkipCurrentQuestion = true
				for c := range roomClients {
					if c != client {
						nextClient = c
						break
					}
				}
			}
		}
	} else {
		room.NextHostOverride = ""
	}

	if err := h.roomService.UpdateRoom(room); err != nil {
		log.Printf("❌ 更新房間資料失敗: %v", err)
	}

	leaveData := map[string]interface{}{
		"playerId":     client.ID,
		"playerName":   client.PlayerName,
		"totalPlayers": remainingPlayers,
		"players":      room.GetPlayerList(),
		"currentHost":  room.CurrentHost,
		"hostChanged":  hostChanged,
		"resetAnswers": resetAnswers,
	}

	leaveMsg := Message{
		Type: "PLAYER_LEFT",
		Data: leaveData,
	}

	if msgBytes, err := json.Marshal(leaveMsg); err == nil {
		h.broadcastToRoomExclude(client.RoomID, msgBytes, client)
	}

	if shouldSkipCurrentQuestion && nextClient != nil {
		invalidMsg := Message{
			Type: "QUESTION_INVALID",
			Data: map[string]interface{}{
				"message": "主角離開房間，本題無效",
				"reason":  "host_left",
			},
		}

		if msgBytes, err := json.Marshal(invalidMsg); err == nil {
			h.broadcastToRoomExclude(client.RoomID, msgBytes, nil)
		}

		go func(handler *Client) {
			time.Sleep(2 * time.Second)
			handler.handleNextQuestion()
		}(nextClient)
	}

	log.Printf("✅ 玩家 %s 離開房間 %s 處理完成 (剩餘玩家: %d)", client.PlayerName, client.RoomID, remainingPlayers)
}

// GetRoomClients 獲取房間客戶端列表
func (h *Hub) GetRoomClients(roomID string) []*Client {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	var clients []*Client
	if roomClients, exists := h.rooms[roomID]; exists {
		for client := range roomClients {
			clients = append(clients, client)
		}
	}

	return clients
}

// GetRoomClientCount 獲取房間客戶端數量
func (h *Hub) GetRoomClientCount(roomID string) int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if roomClients, exists := h.rooms[roomID]; exists {
		return len(roomClients)
	}

	return 0
}

// GetTotalClients 獲取總客戶端數量
func (h *Hub) GetTotalClients() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return len(h.clients)
}

// GetTotalRooms 獲取總房間數量
func (h *Hub) GetTotalRooms() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return len(h.rooms)
}

// SendToClient 發送訊息給特定客戶端
func (h *Hub) SendToClient(clientID string, message []byte) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		if client.ID == clientID {
			select {
			case client.send <- message:
				return nil
			default:
				return fmt.Errorf("客戶端 %s 發送通道已滿", clientID)
			}
		}
	}

	return fmt.Errorf("找不到客戶端 %s", clientID)
}

// GetStats 獲取 Hub 統計資訊
func (h *Hub) GetStats() map[string]interface{} {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	roomStats := make(map[string]int)
	for roomID, clients := range h.rooms {
		roomStats[roomID] = len(clients)
	}

	return map[string]interface{}{
		"totalClients": len(h.clients),
		"totalRooms":   len(h.rooms),
		"roomStats":    roomStats,
	}
}
