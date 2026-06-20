package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"kahoot-game/internal/models"
	"kahoot-game/internal/services"
	"kahoot-game/internal/stats"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// 客戶端發送訊息的最大等待時間
	writeWait = 10 * time.Second

	// 客戶端發送 pong 訊息的最大等待時間
	pongWait = 60 * time.Second

	// 發送 ping 訊息的間隔時間，必須小於 pongWait
	pingPeriod = (pongWait * 9) / 10

	// 訊息的最大大小
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 在生產環境中應該檢查 origin
		return true
	},
}

// Client WebSocket 客戶端結構
type Client struct {
	// WebSocket 連線
	conn *websocket.Conn

	// 客戶端唯一 ID
	ID string

	// 發送訊息的通道
	send chan []byte

	// Hub 引用
	hub *Hub

	// 玩家資訊
	PlayerName string
	RoomID     string
	IsHost     bool
}

// Message WebSocket 訊息結構
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// NewClient 創建新的客戶端
func NewClient(conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		conn: conn,
		ID:   uuid.New().String(),
		send: make(chan []byte, 256),
		hub:  hub,
	}
}

// readPump 處理從客戶端讀取訊息
func (c *Client) readPump() {
	defer func() {
		log.Printf("🔄 readPump 結束，發送註銷請求: %s", c.ID)
		c.hub.unregister <- c
		c.conn.Close()
		log.Printf("❌ readPump 清理完成: %s", c.ID)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket 錯誤: %v", err)
			} else {
				log.Printf("WebSocket 正常關閉: %s, 錯誤: %v", c.ID, err)
			}
			break
		}

		// 解析訊息
		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("訊息解析錯誤: %v", err)
			c.sendError("INVALID_MESSAGE", "訊息格式錯誤")
			continue
		}

		// 處理訊息
		c.handleMessage(&msg)
	}
}

// writePump 處理向客戶端發送訊息
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

			// 處理隊列中的其他訊息（每個消息單獨發送）
			n := len(c.send)
			for i := 0; i < n; i++ {
				select {
				case additionalMessage := <-c.send:
					c.conn.SetWriteDeadline(time.Now().Add(writeWait))
					if err := c.conn.WriteMessage(websocket.TextMessage, additionalMessage); err != nil {
						return
					}
				default:
					// 如果沒有更多消息，跳出循環
					break
				}
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 處理客戶端訊息
func (c *Client) handleMessage(msg *Message) {
	log.Printf("📨 收到訊息 type=%s from=%s room=%s", msg.Type, c.ID, c.RoomID)
	switch msg.Type {
	case "CHECK_ROOM":
		c.handleCheckRoom(msg.Data)
	case "CREATE_ROOM":
		c.handleCreateRoom(msg.Data)
	case "JOIN_ROOM":
		c.handleJoinRoom(msg.Data)
	case "JOIN_AS_HOST":
		c.handleJoinAsHost(msg.Data)
	case "REJOIN_ROOM":
		c.handleRejoinRoom(msg.Data)
	case "START_GAME":
		c.handleStartGame(msg.Data)
	case "CONTINUE_GAME":
		c.handleContinueGame()
	case "FORCE_END_GAME":
		c.handleForceEndGame(msg.Data)
	case "SUBMIT_ANSWER":
		c.handleSubmitAnswer(msg.Data)
	case "LEAVE_ROOM":
		c.handleLeaveRoom(msg.Data)
	case "PING":
		c.handlePing()
	default:
		log.Printf("未知訊息類型: [%s] from=%s room=%s", msg.Type, c.ID, c.RoomID)
		c.sendError("UNKNOWN_MESSAGE_TYPE", fmt.Sprintf("未知的訊息類型: %q", msg.Type))
	}
}

// handleCheckRoom 處理查詢房間狀態（不加入房間）
func (c *Client) handleCheckRoom(data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		c.sendError("INVALID_DATA", "查詢房間資料格式錯誤")
		return
	}

	roomID, _ := dataMap["roomId"].(string)
	playerID, _ := dataMap["playerId"].(string)

	if roomID == "" {
		c.sendError("INVALID_DATA", "房間ID不能為空")
		return
	}

	// 查詢房間
	room, err := c.hub.roomService.GetRoom(roomID)
	if err != nil {
		// 房間不存在
		response := Message{
			Type: "ROOM_STATUS",
			Data: map[string]interface{}{
				"roomId": roomID,
				"exists": false,
			},
		}
		c.sendMessage(&response)
		return
	}

	// 檢查玩家是否在房間中
	playerExists := false
	isHost := false
	playerName := ""
	if playerID != "" {
		if player, exists := room.Players[playerID]; exists {
			playerExists = true
			isHost = player.IsHost
			playerName = player.Name
		}
	}

	response := Message{
		Type: "ROOM_STATUS",
		Data: map[string]interface{}{
			"roomId":       roomID,
			"exists":       true,
			"status":       room.Status,
			"playerCount":  room.GetPlayerCount(),
			"playerExists": playerExists,
			"playerName":   playerName,
			"isHost":       isHost,
			"hostName":     room.HostName,
		},
	}
	c.sendMessage(&response)

	log.Printf("🔍 CHECK_ROOM: 房間=%s, 狀態=%s, 玩家存在=%v, 是主持人=%v",
		roomID, room.Status, playerExists, isHost)
}

// handleCreateRoom 處理創建房間
func (c *Client) handleCreateRoom(data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		c.sendError("INVALID_DATA", "創建房間資料格式錯誤")
		return
	}

	hostName, _ := dataMap["hostName"].(string)
	totalQuestions := int(dataMap["totalQuestions"].(float64))
	questionTimeLimit := int(dataMap["questionTimeLimit"].(float64))

	if hostName == "" {
		c.sendError("INVALID_HOST_NAME", "主持人名稱不能為空")
		return
	}

	// 呼叫房間服務創建房間
	room, err := c.hub.roomService.CreateRoom(hostName, totalQuestions, questionTimeLimit)
	if err != nil {
		log.Printf("創建房間錯誤: %v", err)
		c.sendError("CREATE_ROOM_FAILED", "創建房間失敗")
		return
	}

	// 設定客戶端資訊
	c.PlayerName = hostName
	c.RoomID = room.ID
	c.IsHost = true

	// 設定房間的主持人 ID
	room.HostID = c.ID

	// 將主持人添加為玩家
	hostPlayer, err := c.hub.roomService.AddPlayer(room.ID, c.ID, hostName)
	if err != nil {
		log.Printf("❌ 添加主持人失敗: %v", err)
	} else {
		hostPlayer.IsHost = true
		// 重新獲取房間確保數據更新
		room, _ = c.hub.roomService.GetRoom(room.ID)
		if room != nil {
			c.hub.roomService.UpdateRoom(room)
		}
		log.Printf("✅ 主持人已添加為玩家: %s (ID: %s)", hostName, c.ID)
	}

	// 將客戶端加入房間
	c.hub.AddClientToRoom(c, room.ID)

	stats.Track("room_create", map[string]interface{}{"room": room.ID, "meta": map[string]interface{}{"totalQuestions": totalQuestions}})

	// 生成房間 URL（根據配置調整）
	roomUrl := c.hub.BuildJoinURL(room.ID)

	// 發送房間創建成功訊息
	response := Message{
		Type: "ROOM_CREATED",
		Data: map[string]interface{}{
			"roomId":            room.ID,
			"hostName":          hostName,
			"clientId":          c.ID, // 返回 clientId 作為 playerId
			"hostToken":         room.HostToken, // 主持人身份驗證 token
			"totalQuestions":    totalQuestions,
			"questionTimeLimit": questionTimeLimit,
			"roomUrl":           roomUrl,
			"joinCode":          room.ID, // 用於 QR Code 生成
		},
	}

	c.sendMessage(&response)
	log.Printf("🏠 房間 %s 創建成功，主持人: %s", room.ID, hostName)
}

// handleJoinRoom 處理加入房間
func (c *Client) handleJoinRoom(data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		c.sendError("INVALID_DATA", "加入房間資料格式錯誤")
		return
	}

	roomID, _ := dataMap["roomId"].(string)
	playerName, _ := dataMap["playerName"].(string)

	if roomID == "" || playerName == "" {
		c.sendError("INVALID_ROOM_DATA", "房間ID和玩家名稱不能為空")
		return
	}

	// 檢查客戶端是否已經在其他房間
	if c.RoomID != "" && c.RoomID != roomID {
		log.Printf("⚠️ 客戶端 %s 嘗試加入新房間 %s，但仍在舊房間 %s 中", c.ID, roomID, c.RoomID)
		c.hub.removeClientFromRoom(c, c.RoomID)
	}

	// 呼叫房間服務加入房間
	player, err := c.hub.roomService.AddPlayer(roomID, c.ID, playerName)
	if err != nil {
		log.Printf("加入房間錯誤: %v", err)
		c.sendError("JOIN_ROOM_FAILED", err.Error())
		return
	}

	// 設定客戶端資訊
	c.PlayerName = playerName
	c.RoomID = roomID
	c.IsHost = false

	// 將客戶端加入房間
	c.hub.AddClientToRoom(c, roomID)

	// 獲取房間資訊
	room, _ := c.hub.roomService.GetRoom(roomID)

	// 發送加入成功訊息給該玩家
	joinResponse := Message{
		Type: "PLAYER_JOINED",
		Data: map[string]interface{}{
			"playerId":     player.ID,
			"playerName":   player.Name,
			"roomId":       roomID,
			"totalPlayers": room.GetPlayerCount(),
			"players":      room.GetPlayerList(),
		},
	}
	c.sendMessage(&joinResponse)

	// 廣播給房間內其他玩家
	broadcastMsg := Message{
		Type: "PLAYER_JOINED",
		Data: map[string]interface{}{
			"playerId":     player.ID,
			"playerName":   player.Name,
			"totalPlayers": room.GetPlayerCount(),
			"players":      room.GetPlayerList(),
		},
	}

	if msgBytes, err := json.Marshal(broadcastMsg); err == nil {
		c.hub.BroadcastToRoom(roomID, msgBytes)
	}

	log.Printf("👤 玩家 %s 加入房間 %s", playerName, roomID)
}

// handleJoinAsHost 處理主持人加入房間（房間已通過 HTTP API 創建）
func (c *Client) handleJoinAsHost(data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		c.sendError("INVALID_DATA", "加入房間資料格式錯誤")
		return
	}

	roomID, _ := dataMap["roomId"].(string)
	hostName, _ := dataMap["hostName"].(string)

	if roomID == "" {
		c.sendError("INVALID_ROOM_DATA", "房間ID不能為空")
		return
	}

	// 檢查客戶端是否已經在其他房間
	if c.RoomID != "" && c.RoomID != roomID {
		log.Printf("⚠️ 客戶端 %s 嘗試加入新房間 %s，但仍在舊房間 %s 中", c.ID, roomID, c.RoomID)
		// 從舊房間移除
		c.hub.removeClientFromRoom(c, c.RoomID)
	}

	// 驗證房間是否存在
	room, err := c.hub.roomService.GetRoom(roomID)
	if err != nil {
		log.Printf("房間不存在: %v", err)
		c.sendError("ROOM_NOT_FOUND", "房間不存在")
		return
	}

	// 如果沒有提供 hostName，從房間數據獲取
	if hostName == "" {
		hostName = room.HostName
	}

	// 設定客戶端資訊
	c.PlayerName = hostName
	c.RoomID = roomID
	c.IsHost = true

	// 將主持人加入 room.Players（讓重連/離線偵測可以運作）
	if _, exists := room.Players[c.ID]; !exists {
		hostPlayer := &models.Player{
			ID:          c.ID,
			Name:        hostName,
			RoomID:      roomID,
			Score:       0,
			IsHost:      true,
			IsConnected: true,
			LastActivity: time.Now(),
		}
		room.AddPlayer(hostPlayer)
		room.HostID = c.ID
		if err := c.hub.roomService.UpdateRoom(room); err != nil {
			log.Printf("⚠️ 更新房間主持人玩家失敗: %v", err)
		}
	} else {
		// 主持人已存在（可能是重連），更新為在線
		room.Players[c.ID].IsConnected = true
		c.hub.roomService.UpdateRoom(room)
	}

	// 將客戶端加入房間
	c.hub.AddClientToRoom(c, roomID)

	log.Printf("✅ 主持人 %s 加入房間 %s (playerID=%s)", hostName, roomID, c.ID)

	// 發送加入成功訊息
	roomUrl := c.hub.BuildJoinURL(roomID)

	joinResponse := Message{
		Type: "HOST_JOINED",
		Data: map[string]interface{}{
			"clientId":     c.ID,
			"hostName":     hostName,
			"roomId":       roomID,
			"roomUrl":      roomUrl,
			"hostToken":    room.HostToken,
			"totalPlayers": room.GetPlayerCount(),
			"players":      room.GetPlayerList(),
		},
	}
	c.sendMessage(&joinResponse)

	log.Printf("🎯 主持人 %s 通過 WebSocket 加入房間 %s", hostName, roomID)
}

// handleRejoinRoom 處理玩家重新加入房間
func (c *Client) handleRejoinRoom(data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		c.sendError("INVALID_DATA", "重連資料格式錯誤")
		return
	}

	roomID, _ := dataMap["roomId"].(string)
	playerID, _ := dataMap["playerId"].(string)
	hostToken, _ := dataMap["hostToken"].(string)

	log.Printf("🔄 收到重連請求: roomID=%s, playerID=%s, hasHostToken=%v", roomID, playerID, hostToken != "")

	if roomID == "" || playerID == "" {
		c.sendError("INVALID_DATA", "房間ID和玩家ID不能為空")
		return
	}

	// 檢查客戶端是否已經在其他房間
	if c.RoomID != "" && c.RoomID != roomID {
		log.Printf("⚠️ 客戶端 %s 嘗試重連到新房間 %s，但仍在舊房間 %s 中", c.ID, roomID, c.RoomID)
		c.hub.removeClientFromRoom(c, c.RoomID)
	}

	// 獲取房間
	room, err := c.hub.roomService.GetRoom(roomID)
	if err != nil {
		log.Printf("❌ 房間不存在: %s", roomID)
		c.sendError("ROOM_NOT_FOUND", "房間不存在或已關閉")
		return
	}

	log.Printf("🔍 房間玩家列表: %v", room.Players)

	// 檢查玩家是否在房間中
	player, exists := room.Players[playerID]
	if !exists {
		log.Printf("❌ 玩家不存在: playerID=%s, 現有玩家=%v", playerID, room.Players)
		c.sendError("PLAYER_NOT_FOUND", "找不到您的玩家記錄，請重新加入")
		return
	}

	// 判斷是否為主持人（通過 hostToken 驗證或原有的 IsHost 標記）
	isHost := player.IsHost
	if hostToken != "" && room.HostToken != "" && hostToken == room.HostToken {
		isHost = true
		player.IsHost = true
		log.Printf("✅ hostToken 驗證通過，恢復主持人身份: %s", player.Name)
	}

	// 更新客戶端狀態
	c.ID = playerID
	c.PlayerName = player.Name
	c.RoomID = roomID
	c.IsHost = isHost

	// 更新玩家為在線
	player.IsConnected = true
	room.LastActivity = time.Now()
	c.hub.roomService.UpdateRoom(room)

	// 將客戶端加入房間
	c.hub.AddClientToRoom(c, roomID)

	log.Printf("🔄 玩家 %s (%s) 成功重新加入房間 %s (isHost=%v)", player.Name, playerID, roomID, isHost)

	// 判斷是否需要等待下一題
	isGameInProgress := room.Status == "question_display" ||
		room.Status == "show_result" ||
		room.Status == "answering"
	waitingForNextQuestion := isGameInProgress

	// 建立完整的重連資料
	rejoinData := map[string]interface{}{
		"playerId":               playerID,
		"playerName":             player.Name,
		"roomId":                 roomID,
		"roomStatus":             room.Status,
		"gameState":              room.Status,
		"score":                  player.Score,
		"players":                room.GetPlayerList(),
		"currentQuestionIndex":   room.CurrentQuestion - 1, // 前端使用 0-based index
		"totalQuestions":         room.TotalQuestions,
		"waitingForNextQuestion": waitingForNextQuestion,
		"isHost":                 isHost,
		"canEndGame":             isHost,
		"currentHost":            room.CurrentHost,
		"scores":                 room.GetSortedPlayersByScore(),
		"hostId":                 room.HostID,
		"hostName":               room.HostName,
		"timeLeft":               room.TimeLeft,
		"message":                "重新連線成功！",
	}

	// 遊戲進行中時，發送當前題目的完整資訊（room.CurrentQuestion 是 1-based）
	if isGameInProgress && room.CurrentQuestion >= 1 && room.CurrentQuestion <= len(room.Questions) {
		q := room.Questions[room.CurrentQuestion-1]
		rejoinData["currentQuestionData"] = map[string]interface{}{
			"id":           q.ID,
			"questionText": q.QuestionText,
			"optionA":      q.OptionA,
			"optionB":      q.OptionB,
			"category":     q.Category,
		}
	}

	// 發送重連成功訊息
	response := Message{
		Type: "REJOIN_SUCCESS",
		Data: rejoinData,
	}
	c.sendMessage(&response)

	// 廣播玩家回來了
	broadcastMsg := Message{
		Type: "PLAYER_REJOINED",
		Data: map[string]interface{}{
			"playerId":   playerID,
			"playerName": player.Name,
			"players":    room.GetPlayerList(),
		},
	}
	if msgBytes, err := json.Marshal(broadcastMsg); err == nil {
		c.hub.BroadcastToRoom(roomID, msgBytes)
	}
}

// handleStartGame 處理開始遊戲
func (c *Client) handleStartGame(data interface{}) {
	if !c.IsHost {
		c.sendError("PERMISSION_DENIED", "只有主持人可以開始遊戲")
		return
	}

	// 獲取房間信息
	room, err := c.hub.roomService.GetRoom(c.RoomID)
	if err != nil {
		log.Printf("獲取房間錯誤: %v", err)
		c.sendError("ROOM_NOT_FOUND", "房間不存在")
		return
	}

	log.Printf("🔍 開始遊戲前檢查: 房間狀態=%s, 玩家數量=%d", room.Status, room.GetPlayerCount())

	// 檢查玩家數量
	if room.GetPlayerCount() < 2 {
		c.sendError("INSUFFICIENT_PLAYERS", "至少需要2個玩家才能開始遊戲")
		return
	}

	// 如果房間已經結束，重置房間狀態以允許重新開始
	if room.Status == models.RoomStatusFinished {
		log.Printf("🔄 房間已結束，重置狀態以重新開始遊戲")
		room.Status = models.RoomStatusWaiting
		room.CurrentQuestion = 0
		room.Answers = make(map[string]*models.Answer)

		// 重置所有玩家分數
		for _, player := range room.Players {
			player.Score = 0
		}

		// 更新房間狀態
		err = c.hub.roomService.UpdateRoom(room)
		if err != nil {
			log.Printf("重置房間狀態錯誤: %v", err)
		}
	}

	// 開始「2種人」遊戲
	err = c.hub.gameService.StartTwoTypesGame(room)
	if err != nil {
		log.Printf("開始遊戲錯誤: %v", err)
		c.sendError("START_GAME_FAILED", err.Error())
		return
	}

	// 更新房間狀態
	err = c.hub.roomService.UpdateRoom(room)
	if err != nil {
		log.Printf("更新房間狀態錯誤: %v", err)
	}

	// 廣播遊戲開始訊息
	gameStartMsg := Message{
		Type: "GAME_STARTED",
		Data: map[string]interface{}{
			"roomId":         room.ID,
			"firstHost":      room.CurrentHost,
			"totalQuestions": room.TotalQuestions,
		},
	}

	log.Printf("🎮 準備廣播 GAME_STARTED: 房間ID=%s, 客戶端房間ID=%s", room.ID, c.RoomID)

	if msgBytes, err := json.Marshal(gameStartMsg); err == nil {
		log.Printf("🎮 廣播 GAME_STARTED 到房間 %s", c.RoomID)
		c.hub.BroadcastToRoom(c.RoomID, msgBytes)
	} else {
		log.Printf("❌ 廣播 GAME_STARTED 失敗: %v", err)
	}

	stats.Track("game_start", map[string]interface{}{"room": room.ID, "players": len(room.Players), "meta": map[string]interface{}{"totalQuestions": room.TotalQuestions}})

	// 發送第一題
	c.sendFirstQuestion()

	log.Printf("🎮 房間 %s 開始遊戲，第一個主角: %s", c.RoomID, room.CurrentHost)
}

// sendFirstQuestion 發送第一題
func (c *Client) sendFirstQuestion() {
	room, err := c.hub.roomService.GetRoom(c.RoomID)
	if err != nil {
		log.Printf("獲取房間錯誤: %v", err)
		return
	}

	log.Printf("🔍 檢查房間題目: 房間ID=%s, 題目數量=%d, 總題目設定=%d", c.RoomID, len(room.Questions), room.TotalQuestions)

	if len(room.Questions) == 0 {
		log.Printf("❌ 房間 %s 沒有題目，嘗試重新載入題目", c.RoomID)

		// 嘗試重新載入題目
		room.Questions = services.GetRandomQuestions(room.TotalQuestions)
		log.Printf("🔄 重新載入後題目數量: %d", len(room.Questions))

		if len(room.Questions) == 0 {
			log.Printf("❌ 重新載入題目失敗，無法開始遊戲")
			c.sendError("NO_QUESTIONS", "無法載入遊戲題目")
			return
		}

		// 更新房間
		err = c.hub.roomService.UpdateRoom(room)
		if err != nil {
			log.Printf("更新房間錯誤: %v", err)
		}
	}

	currentQuestion := room.Questions[room.CurrentQuestion-1]

	// 發送新題目訊息
	newQuestionMsg := Message{
		Type: "NEW_QUESTION",
		Data: map[string]interface{}{
			"questionId":      currentQuestion.ID,
			"questionText":    currentQuestion.QuestionText,
			"optionA":         currentQuestion.OptionA,
			"optionB":         currentQuestion.OptionB,
			"questionIndex":   room.CurrentQuestion - 1, // 前端使用 0-based index
			"currentQuestion": room.CurrentQuestion,
			"totalQuestions":  room.TotalQuestions,
			"hostPlayer":      room.CurrentHost,
			"timeLimit":       room.QuestionTimeLimit,
			"question":        currentQuestion.QuestionText, // 前端可能使用這個字段
			"roomId":          room.ID,                      // 幫助客戶端驗證訊息所屬房間
		},
	}

	if msgBytes, err := json.Marshal(newQuestionMsg); err == nil {
		c.hub.BroadcastToRoom(c.RoomID, msgBytes)
	}

	log.Printf("📝 房間 %s 發送第 %d 題，主角: %s", c.RoomID, room.CurrentQuestion, room.CurrentHost)

	// 只有主持人啟動計時器，避免重複
	if c.IsHost {
		go c.startQuestionTimer(room.QuestionTimeLimit)
	}
}

// startQuestionTimer 開始答題倒數計時
func (c *Client) startQuestionTimer(timeLimit int) {
	room, err := c.hub.roomService.GetRoom(c.RoomID)
	if err != nil {
		return
	}

	log.Printf("⏰ 計時器啟動: 觸發者=%s, 是否主持人=%t, 房間=%s, 題目=%d", c.PlayerName, c.IsHost, c.RoomID, room.CurrentQuestion)

	// 設置計時器標識，防止重複啟動
	timerKey := fmt.Sprintf("timer_%s_%d", c.RoomID, room.CurrentQuestion)

	for i := timeLimit; i >= 0; i-- {
		// 檢查房間狀態，如果已經不在答題狀態就停止計時
		room, err = c.hub.roomService.GetRoom(c.RoomID)
		if err != nil || room.Status != models.RoomStatusQuestionDisplay {
			log.Printf("⏹️ 計時器停止: 房間狀態改變或錯誤")
			return
		}

		// 檢查是否有新題目開始（避免舊計時器繼續）
		currentTimerKey := fmt.Sprintf("timer_%s_%d", c.RoomID, room.CurrentQuestion)
		if currentTimerKey != timerKey {
			log.Printf("⏹️ 計時器停止: 新題目已開始")
			return
		}

		// 更新房間剩餘時間（供重連玩家使用）
		room.TimeLeft = i

		// 廣播倒數時間
		timerMsg := Message{
			Type: "TIMER_UPDATE",
			Data: map[string]interface{}{
				"timeLeft":      i,
				"questionIndex": room.CurrentQuestion,
			},
		}

		if msgBytes, err := json.Marshal(timerMsg); err == nil {
			c.hub.BroadcastToRoom(c.RoomID, msgBytes)
		}

		log.Printf("⏱️ 房間 %s 第 %d 題倒數: %d 秒", c.RoomID, room.CurrentQuestion, i)

		// 如果時間到了，處理答題結束
		if i == 0 {
			c.handleQuestionTimeout()
			return
		}

		time.Sleep(1 * time.Second)
	}
}

// handleQuestionTimeout 處理答題時間結束
func (c *Client) handleQuestionTimeout() {
	room, err := c.hub.roomService.GetRoom(c.RoomID)
	if err != nil {
		return
	}

	// 廣播時間結束
	timeoutMsg := Message{
		Type: "QUESTION_TIMEOUT",
		Data: map[string]interface{}{
			"message": "答題時間結束",
		},
	}

	if msgBytes, err := json.Marshal(timeoutMsg); err == nil {
		c.hub.BroadcastToRoom(c.RoomID, msgBytes)
	}

	log.Printf("⏰ 房間 %s 第 %d 題答題時間結束", c.RoomID, room.CurrentQuestion)

	// 檢查倒數結束時的答題情況
	totalPlayers := room.GetPlayerCount()
	answeredPlayers := 0
	hostAnswered := false

	if room.Answers != nil {
		answeredPlayers = len(room.Answers)
		// 檢查主角是否已答題
		if _, exists := room.Answers[room.CurrentHost]; exists {
			hostAnswered = true
		}
	}

	log.Printf("⏰ 時間結束統計: 總玩家=%d, 已答題=%d, 主角已答題=%t", totalPlayers, answeredPlayers, hostAnswered)

	if answeredPlayers > 0 && hostAnswered {
		// 主角已答題，可以進行正常計分
		log.Printf("📊 主角已答題，開始計算分數")
		c.calculateAndShowResults(room)
	} else if answeredPlayers > 0 && !hostAnswered {
		// 有人答題但主角沒答題，這題無效
		log.Printf("⚠️ 主角未答題，本題無效，3秒後進入下一題")

		// 廣播主角未答題訊息
		invalidMsg := Message{
			Type: "QUESTION_INVALID",
			Data: map[string]interface{}{
				"message": "主角未在時間內答題，本題無效",
				"reason":  "host_no_answer",
			},
		}

		if msgBytes, err := json.Marshal(invalidMsg); err == nil {
			c.hub.BroadcastToRoom(c.RoomID, msgBytes)
		}

		go func() {
			time.Sleep(3 * time.Second)
			// 清除答案記錄
			room.Answers = make(map[string]*models.Answer)
			c.hub.roomService.UpdateRoom(room)
			c.handleNextQuestion()
		}()
	} else {
		// 沒人答題，直接進入下一題
		log.Printf("📊 沒有玩家答題，3秒後進入下一題")

		// 廣播沒人答題訊息
		noAnswerMsg := Message{
			Type: "QUESTION_SKIPPED",
			Data: map[string]interface{}{
				"message": "時間到，沒有玩家答題",
				"reason":  "no_answers",
			},
		}

		if msgBytes, err := json.Marshal(noAnswerMsg); err == nil {
			c.hub.BroadcastToRoom(c.RoomID, msgBytes)
		}

		go func() {
			time.Sleep(3 * time.Second)
			c.handleNextQuestion()
		}()
	}
}

// handleNextQuestion 處理下一題邏輯
func (c *Client) handleNextQuestion() {
	room, err := c.hub.roomService.GetRoom(c.RoomID)
	if err != nil {
		log.Printf("獲取房間錯誤: %v", err)
		return
	}

	log.Printf("🔄 準備進入下一題: 當前題目=%d, 總題目=%d, 題庫大小=%d", room.CurrentQuestion, room.TotalQuestions, len(room.Questions))

	// 進入下一題
	c.hub.gameService.NextTwoTypesQuestion(room)

	log.Printf("🔄 進入下一題後: 當前題目=%d, 總題目=%d, 房間狀態=%s", room.CurrentQuestion, room.TotalQuestions, room.Status)

	// 更新房間狀態
	err = c.hub.roomService.UpdateRoom(room)
	if err != nil {
		log.Printf("更新房間狀態錯誤: %v", err)
	}

	// 檢查遊戲是否結束
	if room.Status == models.RoomStatusFinished {
		// 遊戲結束，發送最終結果 (包含詳細統計)
		finalStats := c.hub.gameService.GetFinalRanking(room)

		log.Printf("🏁 發送遊戲結束訊息，最終統計:")
		for i, stat := range finalStats {
			log.Printf("   %d. %s (ID: %s): %d分, 答對%d題", i+1, stat.PlayerName, stat.PlayerID, stat.TotalScore, stat.CorrectGuesses)
		}

		gameEndMsg := Message{
			Type: "GAME_FINISHED",
			Data: map[string]interface{}{
				"finalStats":     finalStats,
				"message":        "遊戲結束！",
				"totalQuestions": room.TotalQuestions,
			},
		}

		if msgBytes, err := json.Marshal(gameEndMsg); err == nil {
			c.hub.BroadcastToRoom(c.RoomID, msgBytes)
		}

		stats.Track("game_end", map[string]interface{}{"room": c.RoomID, "players": len(room.Players), "meta": map[string]interface{}{"totalQuestions": room.TotalQuestions}})
		log.Printf("🏁 房間 %s 遊戲結束，發送詳細統計給所有玩家", c.RoomID)
	} else {
		// 檢查是否還有題目可以發送
		if room.CurrentQuestion <= len(room.Questions) {
			log.Printf("📝 發送第 %d 題", room.CurrentQuestion)
			c.sendNextQuestion()
		} else {
			log.Printf("❌ 沒有更多題目了，強制結束遊戲")
			// 強制結束遊戲
			room.Status = models.RoomStatusFinished
			c.hub.roomService.UpdateRoom(room)

			finalResults := c.hub.gameService.GetFinalRanking(room)
			gameEndMsg := Message{
				Type: "GAME_FINISHED",
				Data: map[string]interface{}{
					"finalRanking": finalResults,
					"message":      "遊戲結束！",
				},
			}

			if msgBytes, err := json.Marshal(gameEndMsg); err == nil {
				c.hub.BroadcastToRoom(c.RoomID, msgBytes)
			}
		}
	}
}

// sendNextQuestion 發送下一題
func (c *Client) sendNextQuestion() {
	room, err := c.hub.roomService.GetRoom(c.RoomID)
	if err != nil {
		log.Printf("獲取房間錯誤: %v", err)
		return
	}

	if room.CurrentQuestion > len(room.Questions) {
		log.Printf("沒有更多題目了，當前題目: %d, 總題目: %d", room.CurrentQuestion, len(room.Questions))
		return
	}

	log.Printf("🔍 發送題目檢查: 當前題目編號=%d, 題目總數=%d", room.CurrentQuestion, len(room.Questions))

	if room.CurrentQuestion < 1 || room.CurrentQuestion > len(room.Questions) {
		log.Printf("❌ 題目編號超出範圍: %d (總共 %d 題)", room.CurrentQuestion, len(room.Questions))
		return
	}

	currentQuestion := room.Questions[room.CurrentQuestion-1]
	log.Printf("📝 準備發送第 %d 題: %s", room.CurrentQuestion, currentQuestion.QuestionText)

	// 確保房間狀態正確
	room.Status = models.RoomStatusQuestionDisplay

	// 發送新題目訊息
	newQuestionMsg := Message{
		Type: "NEW_QUESTION",
		Data: map[string]interface{}{
			"questionId":      currentQuestion.ID,
			"questionText":    currentQuestion.QuestionText,
			"optionA":         currentQuestion.OptionA,
			"optionB":         currentQuestion.OptionB,
			"questionIndex":   room.CurrentQuestion - 1, // 前端使用 0-based index
			"currentQuestion": room.CurrentQuestion,
			"totalQuestions":  room.TotalQuestions,
			"hostPlayer":      room.CurrentHost,
			"timeLimit":       room.QuestionTimeLimit,
			"question":        currentQuestion.QuestionText, // 前端可能使用這個字段
			"roomId":          room.ID,                      // 幫助客戶端驗證訊息所屬房間
		},
	}

	if msgBytes, err := json.Marshal(newQuestionMsg); err == nil {
		c.hub.BroadcastToRoom(c.RoomID, msgBytes)
	}

	log.Printf("📝 房間 %s 發送第 %d 題，主角: %s", c.RoomID, room.CurrentQuestion, room.CurrentHost)

	// 啟動計時器（移除主持人限制，因為任何客戶端都可能觸發下一題）
	log.Printf("⏰ 啟動第 %d 題計時器 (觸發者: %s)", room.CurrentQuestion, c.PlayerName)
	go c.startQuestionTimer(room.QuestionTimeLimit)
}

// handleSubmitAnswer 處理提交答案
func (c *Client) handleSubmitAnswer(data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		c.sendError("INVALID_DATA", "提交答案資料格式錯誤")
		return
	}

	answer, ok := dataMap["answer"].(string)
	if !ok {
		c.sendError("INVALID_DATA", "答案格式錯誤")
		return
	}

	timeUsed, _ := dataMap["timeUsed"].(float64)

	// 獲取房間信息
	room, err := c.hub.roomService.GetRoom(c.RoomID)
	if err != nil {
		log.Printf("獲取房間錯誤: %v", err)
		c.sendError("ROOM_NOT_FOUND", "房間不存在")
		return
	}

	// 檢查遊戲狀態
	if room.Status != models.RoomStatusQuestionDisplay {
		c.sendError("INVALID_STATE", "當前不在答題階段")
		return
	}

	// 提交「2種人」答案
	answerRecord, err := c.hub.gameService.SubmitTwoTypesAnswer(room, c.ID, answer, timeUsed)
	if err != nil {
		log.Printf("提交答案錯誤: %v", err)
		c.sendError("SUBMIT_FAILED", err.Error())
		return
	}

	// 存儲答案到房間
	if room.Answers == nil {
		room.Answers = make(map[string]*models.Answer)
	}
	room.Answers[c.ID] = answerRecord

	// 記錄答案提交詳情
	isHost := c.ID == room.CurrentHost
	log.Printf("📝 答案記錄: 玩家ID=%s, 玩家名=%s, 答案=%s, 是否主角=%t, 當前主角=%s", c.ID, c.PlayerName, answer, isHost, room.CurrentHost)
	log.Printf("📊 當前答案總數: %d/%d", len(room.Answers), room.GetPlayerCount())

	// 記錄所有已答題的玩家
	for playerID, ans := range room.Answers {
		player, exists := room.GetPlayer(playerID)
		playerName := "Unknown"
		if exists {
			playerName = player.Name
		}
		log.Printf("  - 玩家 %s (%s): %s", playerName, playerID, ans.Answer)
	}

	// 更新房間狀態
	err = c.hub.roomService.UpdateRoom(room)
	if err != nil {
		log.Printf("更新房間狀態錯誤: %v", err)
	}

	// 發送答案確認給提交者
	confirmMsg := Message{
		Type: "ANSWER_SUBMITTED",
		Data: map[string]interface{}{
			"success":    true,
			"answer":     answer,
			"timeUsed":   timeUsed,
			"playerId":   c.ID,
			"playerName": c.PlayerName,
		},
	}

	if msgBytes, err := json.Marshal(confirmMsg); err == nil {
		c.send <- msgBytes
	}

	// 廣播給房間內所有人，告知有人已作答
	// 給主持人發送包含答案的訊息，給其他玩家發送不含答案的訊息
	roomClients := c.hub.rooms[c.RoomID]

	log.Printf("📡 廣播答題狀態: 房間 %s 有 %d 個客戶端連接", c.RoomID, len(roomClients))
	for client := range roomClients {
		log.Printf("  - Client %s: IsHost=%v, PlayerName=%s", client.ID, client.IsHost, client.PlayerName)
	}

	// 先發送答案給主持人（包含答案）
	hostFound := false
	for client := range roomClients {
		if client.IsHost {
			hostFound = true
			log.Printf("📤 發送 PLAYER_ANSWERED 給主持人: %s", client.ID)
			msgData := map[string]interface{}{
				"playerId":   c.ID,
				"playerName": c.PlayerName,
				"isHost":     c.ID == room.CurrentHost,
				"answer":     answer,
			}
			broadcastMsg := Message{
				Type: "PLAYER_ANSWERED",
				Data: msgData,
			}
			if broadcastBytes, err := json.Marshal(broadcastMsg); err == nil {
				client.send <- broadcastBytes
			}
			break
		}
	}

	if !hostFound {
		log.Printf("⚠️ 房間 %s 中找不到主持人客戶端", c.RoomID)
	}

	// 發送不含答案的訊息給其他玩家
	for client := range roomClients {
		if client == c || client.IsHost {
			continue
		}
		log.Printf("📤 發送 PLAYER_ANSWERED 給玩家: %s", client.ID)
		msgData := map[string]interface{}{
			"playerId":   c.ID,
			"playerName": c.PlayerName,
			"isHost":     c.ID == room.CurrentHost,
		}
		broadcastMsg := Message{
			Type: "PLAYER_ANSWERED",
			Data: msgData,
		}
		if broadcastBytes, err := json.Marshal(broadcastMsg); err == nil {
			client.send <- broadcastBytes
		}
	}

	// 檢查是否所有玩家都已答題
	if c.checkAllPlayersAnswered(room) {
		// 所有人都答完了，計算分數並顯示結果
		c.calculateAndShowResults(room)
	}

	log.Printf("🎯 玩家 %s 提交答案: %s (耗時: %.2f秒), 已廣播給其他玩家", c.PlayerName, answer, timeUsed)
}

// checkAllPlayersAnswered 檢查是否所有非主持人玩家都已答題
func (c *Client) checkAllPlayersAnswered(room *models.Room) bool {
	totalNonHostPlayers := 0
	for _, p := range room.Players {
		if !p.IsHost {
			totalNonHostPlayers++
		}
	}
	answeredPlayers := len(room.Answers)

	log.Printf("📊 答題進度: %d/%d 非主持人玩家已答題", answeredPlayers, totalNonHostPlayers)
	return answeredPlayers >= totalNonHostPlayers
}

// calculateAndShowResults 計算並顯示結果
func (c *Client) calculateAndShowResults(room *models.Room) {
	// 計算分數
	scores := c.hub.gameService.CalculateTwoTypesScores(room, room.Answers)

	// 更新房間狀態為 show_result（讓 goroutine 可以檢查此狀態）
	room.Status = models.RoomStatusShowResult
	err := c.hub.roomService.UpdateRoom(room)
	if err != nil {
		log.Printf("更新房間狀態錯誤: %v", err)
	}

	// 計算答題統計
	answeredCount := 0
	correctCount := 0
	hostAnswer := c.getHostAnswer(room)
	playerCount := room.GetPlayerCount() // 玩家數量（不含主持人）

	if room.Answers != nil {
		answeredCount = len(room.Answers)
		// 計算猜對主角答案的玩家數量
		for playerID, answer := range room.Answers {
			if playerID != room.CurrentHost && answer.Answer == hostAnswer {
				correctCount++
			}
		}
	}

	log.Printf("📊 答題統計: 玩家數=%d, 作答人數=%d, 答對人數=%d", playerCount, answeredCount, correctCount)

	// 廣播分數結果
	scoresMsg := Message{
		Type: "SCORES_UPDATE",
		Data: map[string]interface{}{
			"scores":          scores,
			"currentQuestion": room.CurrentQuestion,
			"hostAnswer":      hostAnswer,
			"answeredCount":   answeredCount,
			"correctCount":    correctCount,
			"totalPlayers":    playerCount,
		},
	}

	if msgBytes, err := json.Marshal(scoresMsg); err == nil {
		c.hub.BroadcastToRoom(c.RoomID, msgBytes)
	}

	// 記錄題目歷史
	c.recordQuestionHistory(room)

	// 延遲5秒後自動進入下一題，讓玩家有時間查看分數
	go func() {
		time.Sleep(5 * time.Second)

		// 重新獲取房間狀態（避免併發問題）
		currentRoom, err := c.hub.roomService.GetRoom(c.RoomID)
		if err != nil {
			log.Printf("獲取房間錯誤: %v", err)
			return
		}

		// 若主持人已按「繼續」手動提前進入下一題，跳過自動觸發
		if currentRoom.Status != models.RoomStatusShowResult {
			log.Printf("⏭️ 主持人已手動繼續，跳過自動下一題")
			return
		}

		// 清除答案記錄，準備下一題
		currentRoom.Answers = make(map[string]*models.Answer)
		c.hub.roomService.UpdateRoom(currentRoom)

		log.Printf("🔄 開始處理下一題邏輯...")
		c.handleNextQuestion()
	}()

	log.Printf("📊 房間 %s 第 %d 題計分完成，5秒後自動下一題", c.RoomID, room.CurrentQuestion)
}

// recordQuestionHistory 記錄題目歷史
func (c *Client) recordQuestionHistory(room *models.Room) {
	if room.Answers == nil || len(room.Answers) == 0 {
		log.Printf("⚠️ 沒有答案記錄，跳過歷史記錄")
		return
	}

	// 找到主角答案
	hostAnswer := ""
	for playerID, answer := range room.Answers {
		if playerID == room.CurrentHost {
			hostAnswer = answer.Answer
			break
		}
	}

	// 創建題目歷史記錄
	history := models.QuestionHistory{
		QuestionID:    room.Questions[room.CurrentQuestion-1].ID,
		QuestionNum:   room.CurrentQuestion,
		HostPlayerID:  room.CurrentHost,
		HostAnswer:    hostAnswer,
		PlayerAnswers: make(map[string]*models.Answer),
	}

	// 複製所有玩家答案
	for playerID, answer := range room.Answers {
		history.PlayerAnswers[playerID] = &models.Answer{
			PlayerID:     answer.PlayerID,
			QuestionID:   answer.QuestionID,
			Answer:       answer.Answer,
			IsCorrect:    answer.IsCorrect,
			ResponseTime: answer.ResponseTime,
			ScoreGained:  answer.ScoreGained,
			WasHost:      answer.WasHost,
			HostAnswer:   hostAnswer,
			SubmittedAt:  answer.SubmittedAt,
		}
	}

	// 添加到房間歷史
	if room.GameHistory == nil {
		room.GameHistory = make([]models.QuestionHistory, 0)
	}
	room.GameHistory = append(room.GameHistory, history)

	log.Printf("📝 記錄第 %d 題歷史: 主角=%s, 答案=%s, 玩家答案數=%d",
		room.CurrentQuestion, room.CurrentHost, hostAnswer, len(history.PlayerAnswers))
}

// getHostAnswer 獲取主角答案
func (c *Client) getHostAnswer(room *models.Room) string {
	for playerID, answer := range room.Answers {
		if playerID == room.CurrentHost {
			return answer.Answer
		}
	}
	return ""
}

// handleForceEndGame 處理強制結束遊戲 (例如主持人手動結束)
// handleContinueGame 主持人手動繼續（結果畫面 → 下一題，或跳過剩餘倒數）
func (c *Client) handleContinueGame() {
	if !c.IsHost {
		c.sendError("PERMISSION_DENIED", "只有主持人可以繼續遊戲")
		return
	}

	room, err := c.hub.roomService.GetRoom(c.RoomID)
	if err != nil {
		log.Printf("獲取房間錯誤: %v", err)
		return
	}

	log.Printf("▶️ 主持人 %s 手動繼續，當前狀態: %s", c.PlayerName, room.Status)

	switch room.Status {
	case models.RoomStatusShowResult:
		// 結果畫面：立即進入下一題（不等 5 秒）
		room.Answers = make(map[string]*models.Answer)
		room.Status = models.RoomStatusQuestionDisplay // 提前改狀態，讓 goroutine 跳過
		c.hub.roomService.UpdateRoom(room)
		c.handleNextQuestion()
	case models.RoomStatusQuestionDisplay:
		// 題目還在倒數：強制結算
		c.calculateAndShowResults(room)
	default:
		log.Printf("⚠️ CONTINUE_GAME 在非預期狀態下被呼叫: %s", room.Status)
	}
}

func (c *Client) handleForceEndGame(data interface{}) {
	// 支援 hostToken 驗證，即使 c.IsHost 為 false 也能通過
	authorized := c.IsHost
	if !authorized {
		if dataMap, ok := data.(map[string]interface{}); ok {
			if hostToken, _ := dataMap["hostToken"].(string); hostToken != "" {
				if room, err := c.hub.roomService.GetRoom(c.RoomID); err == nil {
					if room.HostToken != "" && hostToken == room.HostToken {
						authorized = true
						c.IsHost = true
						log.Printf("✅ FORCE_END_GAME: hostToken 驗證通過: %s", c.PlayerName)
					}
				}
			}
		}
	}
	if !authorized {
		c.sendError("PERMISSION_DENIED", "只有主持人可以結束遊戲")
		return
	}

	room, err := c.hub.roomService.GetRoom(c.RoomID)
	if err != nil {
		log.Printf("獲取房間錯誤: %v", err)
		return
	}

	log.Printf("🛑 主持人 %s 強制結束房間 %s 的遊戲", c.PlayerName, c.RoomID)
	stats.Track("game_end", map[string]interface{}{"room": c.RoomID, "players": len(room.Players), "meta": map[string]interface{}{"forced": true}})

	// 強制結束遊戲
	room.Status = models.RoomStatusFinished
	_ = c.hub.roomService.UpdateRoom(room)

	// 發送最終結果給所有人
	finalStats := c.hub.gameService.GetFinalRanking(room)
	
	gameEndMsg := Message{
		Type: "GAME_FINISHED",
		Data: map[string]interface{}{
			"finalStats":     finalStats,
			"message":        "遊戲已由主持人提早結束",
			"totalQuestions": len(room.Questions),
		},
	}

	if msgBytes, err := json.Marshal(gameEndMsg); err == nil {
		c.hub.BroadcastToRoom(c.RoomID, msgBytes)
	}
}

// handleLeaveRoom 處理離開房間（軟離開：標記離線，不關閉 WebSocket）
// 不呼叫 hub.unregister，避免 close(client.send) 導致後續訊息丟失
func (c *Client) handleLeaveRoom(data interface{}) {
	if c.RoomID == "" {
		return
	}
	c.hub.softLeaveRoom(c)
}

// handlePing 處理 ping 訊息
func (c *Client) handlePing() {
	pongMsg := Message{
		Type: "PONG",
		Data: map[string]interface{}{
			"timestamp": time.Now().Unix(),
		},
	}
	c.sendMessage(&pongMsg)
}

// sendMessage 發送訊息
func (c *Client) sendMessage(msg *Message) {
	if msgBytes, err := json.Marshal(msg); err == nil {
		select {
		case c.send <- msgBytes:
		default:
			log.Printf("客戶端 %s 發送通道已滿", c.ID)
		}
	}
}

// sendError 發送錯誤訊息
func (c *Client) sendError(code, message string) {
	errorMsg := Message{
		Type: "ERROR",
		Data: map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	c.sendMessage(&errorMsg)
}

// ServeWS 處理 WebSocket 連線升級
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) *Client {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket 升級失敗: %v", err)
		return nil
	}

	client := NewClient(conn, hub)
	client.hub.register <- client

	// 在新的 goroutine 中處理讀寫
	go client.writePump()
	go client.readPump()

	return client
}
