package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"kahoot-game/internal/database"
	"kahoot-game/internal/models"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// RoomService 房間服務
type RoomService struct {
	redisClient *redis.Client
	gameService *GameService
	keys        *database.RedisKeys

	// 測試模式用的記憶體存儲
	memoryRooms map[string]*models.Room
	memoryMutex sync.RWMutex
}

// NewRoomService 創建房間服務
func NewRoomService(redisClient *redis.Client, gameService *GameService) *RoomService {
	return &RoomService{
		redisClient: redisClient,
		gameService: gameService,
		keys:        database.NewRedisKeys(),
		memoryRooms: make(map[string]*models.Room),
	}
}

// CreateRoom 創建房間
func (s *RoomService) CreateRoom(hostName string, totalQuestions, questionTimeLimit int) (*models.Room, error) {
	ctx := context.Background()

	// 生成唯一房間ID
	roomID := s.generateRoomID()

	// 生成「2種人」題目
	questions := GetRandomQuestions(totalQuestions)

	// 創建房間
	now := time.Now()
	room := &models.Room{
		ID:                roomID,
		HostName:          hostName,
		HostToken:         uuid.New().String(),
		HostConnected:     true,
		Status:            models.RoomStatusWaiting,
		Players:           make(map[string]*models.Player),
		CurrentQuestion:   0,
		TotalQuestions:    totalQuestions,
		QuestionTimeLimit: questionTimeLimit,
		Questions:         questions,
		CreatedAt:         now,
		LastActivity:      now,
	}

	// 存儲到 Redis（如果有 Redis 客戶端）或記憶體
	if s.redisClient != nil {
		roomData, err := json.Marshal(room)
		if err != nil {
			return nil, fmt.Errorf("序列化房間資料失敗: %w", err)
		}

		err = s.redisClient.Set(ctx, s.keys.RoomKey(roomID), roomData, database.RoomExpiration).Err()
		if err != nil {
			return nil, fmt.Errorf("存儲房間資料失敗: %w", err)
		}

		// 添加到活躍房間列表
		err = s.redisClient.SAdd(ctx, database.ActiveRoomsKey, roomID).Err()
		if err != nil {
			return nil, fmt.Errorf("添加到活躍房間列表失敗: %w", err)
		}
	} else {
		// 測試模式：存儲到記憶體
		s.memoryMutex.Lock()
		s.memoryRooms[roomID] = room
		s.memoryMutex.Unlock()
	}

	return room, nil
}

// GetRoom 獲取房間資訊
func (s *RoomService) GetRoom(roomID string) (*models.Room, error) {
	if s.redisClient != nil {
		ctx := context.Background()

		roomData, err := s.redisClient.Get(ctx, s.keys.RoomKey(roomID)).Result()
		if err == redis.Nil {
			return nil, fmt.Errorf("房間不存在")
		} else if err != nil {
			return nil, fmt.Errorf("獲取房間資料失敗: %w", err)
		}

		var room models.Room
		err = json.Unmarshal([]byte(roomData), &room)
		if err != nil {
			return nil, fmt.Errorf("反序列化房間資料失敗: %w", err)
		}

		return &room, nil
	} else {
		// 測試模式：從記憶體獲取
		s.memoryMutex.RLock()
		room, exists := s.memoryRooms[roomID]
		s.memoryMutex.RUnlock()

		if !exists {
			return nil, fmt.Errorf("房間不存在")
		}

		return room, nil
	}
}

// AddPlayer 添加玩家到房間
func (s *RoomService) AddPlayer(roomID, playerID, playerName string) (*models.Player, error) {
	ctx := context.Background()

	// 獲取房間
	room, err := s.GetRoom(roomID)
	if err != nil {
		return nil, err
	}

	// 檢查房間狀態
	if room.Status == models.RoomStatusFinished {
		return nil, fmt.Errorf("遊戲已結束，請建立新房間")
	}
	if room.Status != models.RoomStatusWaiting {
		return nil, fmt.Errorf("遊戲已開始，無法加入")
	}

	// 檢查房間人數限制
	if len(room.Players) >= 20 { // 最大20人
		return nil, fmt.Errorf("房間已滿")
	}

	// 待機階段：若有同名非主持人玩家（不論線上或離線），直接踢出讓新玩家全新加入
	for existingID, existing := range room.Players {
		if existing.Name == playerName {
			if existing.IsHost {
				// 主持人名稱不能被佔用
				return nil, fmt.Errorf("該名稱已被主持人使用")
			}
			log.Printf("⚠️ 踢出同名玩家 %s (ID: %s)，讓新玩家加入", playerName, existingID)
			delete(room.Players, existingID)
			break
		}
	}

	// 創建玩家
	player := &models.Player{
		ID:           playerID,
		Name:         playerName,
		RoomID:       roomID,
		Score:        0,
		IsHost:       false,
		IsConnected:  true,
		LastActivity: time.Now(),
	}

	// 添加玩家到房間
	room.AddPlayer(player)

	// 更新房間資料
	err = s.updateRoom(room)
	if err != nil {
		return nil, fmt.Errorf("更新房間資料失敗: %w", err)
	}

	// 存儲玩家資料（如果有 Redis 客戶端）
	if s.redisClient != nil {
		playerData, _ := json.Marshal(player)
		err = s.redisClient.Set(ctx, s.keys.PlayerKey(playerID), playerData, database.PlayerExpiration).Err()
		if err != nil {
			return nil, fmt.Errorf("存儲玩家資料失敗: %w", err)
		}
	}

	return player, nil
}

// RemovePlayer 從房間移除玩家
func (s *RoomService) RemovePlayer(roomID, playerID string) error {
	ctx := context.Background()

	// 獲取房間
	room, err := s.GetRoom(roomID)
	if err != nil {
		return err
	}

	// 移除玩家
	room.RemovePlayer(playerID)

	// 如果房間沒有玩家了，刪除房間
	if len(room.Players) == 0 {
		return s.DeleteRoom(roomID)
	}

	// 更新房間資料
	err = s.updateRoom(room)
	if err != nil {
		return fmt.Errorf("更新房間資料失敗: %w", err)
	}

	// 刪除玩家資料（如果有 Redis 客戶端）
	if s.redisClient != nil {
		err = s.redisClient.Del(ctx, s.keys.PlayerKey(playerID)).Err()
		if err != nil {
			return fmt.Errorf("刪除玩家資料失敗: %w", err)
		}
	}

	return nil
}

// DeleteRoom 刪除房間
func (s *RoomService) DeleteRoom(roomID string) error {
	if s.redisClient != nil {
		ctx := context.Background()

		// 從 Redis 刪除房間資料
		err := s.redisClient.Del(ctx, s.keys.RoomKey(roomID)).Err()
		if err != nil {
			return fmt.Errorf("刪除房間資料失敗: %w", err)
		}

		// 從活躍房間列表移除
		err = s.redisClient.SRem(ctx, database.ActiveRoomsKey, roomID).Err()
		if err != nil {
			return fmt.Errorf("從活躍房間列表移除失敗: %w", err)
		}
	} else {
		// 測試模式：從記憶體刪除房間
		s.memoryMutex.Lock()
		delete(s.memoryRooms, roomID)
		s.memoryMutex.Unlock()
	}

	return nil
}

// updateRoom 更新房間資料
func (s *RoomService) updateRoom(room *models.Room) error {
	if s.redisClient != nil {
		ctx := context.Background()

		roomData, err := json.Marshal(room)
		if err != nil {
			return fmt.Errorf("序列化房間資料失敗: %w", err)
		}

		err = s.redisClient.Set(ctx, s.keys.RoomKey(room.ID), roomData, database.RoomExpiration).Err()
		if err != nil {
			return fmt.Errorf("更新房間資料失敗: %w", err)
		}
	} else {
		// 測試模式：更新記憶體中的房間
		s.memoryMutex.Lock()
		s.memoryRooms[room.ID] = room
		s.memoryMutex.Unlock()
	}

	return nil
}

// generateRoomID 生成房間ID
func (s *RoomService) generateRoomID() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 6

	rand.Seed(time.Now().UnixNano())

	for {
		roomID := make([]byte, length)
		for i := range roomID {
			roomID[i] = charset[rand.Intn(len(charset))]
		}

		// 如果沒有 Redis 客戶端（測試模式），檢查記憶體中是否存在
		if s.redisClient == nil {
			s.memoryMutex.RLock()
			_, exists := s.memoryRooms[string(roomID)]
			s.memoryMutex.RUnlock()

			if !exists {
				return string(roomID)
			}
		} else {
			// 檢查ID是否已存在
			ctx := context.Background()
			exists, _ := s.redisClient.Exists(ctx, s.keys.RoomKey(string(roomID))).Result()
			if exists == 0 {
				return string(roomID)
			}
		}
	}
}

// UpdateRoom 更新房間（公開方法）
func (s *RoomService) UpdateRoom(room *models.Room) error {
	return s.updateRoom(room)
}

// GetAllRooms 獲取所有房間（用於清掃）
func (s *RoomService) GetAllRooms() ([]*models.Room, error) {
	if s.redisClient != nil {
		ctx := context.Background()
		roomIDs, err := s.redisClient.SMembers(ctx, database.ActiveRoomsKey).Result()
		if err != nil {
			return nil, fmt.Errorf("獲取活躍房間列表失敗: %w", err)
		}

		rooms := make([]*models.Room, 0, len(roomIDs))
		for _, roomID := range roomIDs {
			room, err := s.GetRoom(roomID)
			if err != nil {
				// 房間可能已過期，從活躍列表移除
				s.redisClient.SRem(ctx, database.ActiveRoomsKey, roomID)
				continue
			}
			rooms = append(rooms, room)
		}
		return rooms, nil
	}

	// 記憶體模式
	s.memoryMutex.RLock()
	defer s.memoryMutex.RUnlock()

	rooms := make([]*models.Room, 0, len(s.memoryRooms))
	for _, room := range s.memoryRooms {
		rooms = append(rooms, room)
	}
	return rooms, nil
}
