package services

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	"kahoot-game/internal/models"

	"github.com/go-redis/redis/v8"
)

// GameService 遊戲服務
type GameService struct {
	db          *sql.DB
	redisClient *redis.Client
}

// NewGameService 創建遊戲服務
func NewGameService(db *sql.DB, redisClient *redis.Client) *GameService {
	return &GameService{
		db:          db,
		redisClient: redisClient,
	}
}

// CreateGame 創建遊戲記錄
func (s *GameService) CreateGame(roomID, hostName string, totalQuestions, questionTimeLimit int) (*models.Game, error) {
	query := `
		INSERT INTO games (room_id, host_name, total_questions, question_time_limit, status)
		VALUES ($1, $2, $3, $4, 'waiting')
		RETURNING id, created_at
	`

	var game models.Game
	err := s.db.QueryRow(query, roomID, hostName, totalQuestions, questionTimeLimit).
		Scan(&game.ID, &game.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("創建遊戲記錄失敗: %w", err)
	}

	game.RoomID = roomID
	game.HostName = hostName
	game.TotalQuestions = totalQuestions
	game.QuestionTimeLimit = questionTimeLimit
	game.Status = "waiting"

	return &game, nil
}

// GetActiveGames 獲取活躍遊戲列表
func (s *GameService) GetActiveGames() ([]models.Game, error) {
	query := `
		SELECT id, room_id, host_name, total_players, total_questions, 
			   question_time_limit, status, created_at
		FROM games 
		WHERE status IN ('waiting', 'playing')
		ORDER BY created_at DESC
		LIMIT 50
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查詢活躍遊戲失敗: %w", err)
	}
	defer rows.Close()

	var games []models.Game
	for rows.Next() {
		var game models.Game
		err := rows.Scan(
			&game.ID, &game.RoomID, &game.HostName, &game.TotalPlayers,
			&game.TotalQuestions, &game.QuestionTimeLimit,
			&game.Status, &game.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("掃描遊戲資料失敗: %w", err)
		}
		games = append(games, game)
	}

	return games, nil
}

// GetGameStats 獲取遊戲統計
func (s *GameService) GetGameStats(gameID int) (*models.GameStatistics, error) {
	query := `
		SELECT g.room_id, g.host_name, g.total_players, g.total_questions,
			   g.duration_seconds, g.winner_name, g.winner_score, g.created_at,
			   COALESCE(AVG(ps.accuracy_percentage), 0) as avg_accuracy,
			   COALESCE(AVG(ps.avg_response_time), 0) as avg_response_time
		FROM games g
		LEFT JOIN player_stats ps ON g.id = ps.game_id
		WHERE g.id = $1
		GROUP BY g.id, g.room_id, g.host_name, g.total_players, g.total_questions,
				 g.duration_seconds, g.winner_name, g.winner_score, g.created_at
	`

	var stats models.GameStatistics
	var winnerName sql.NullString
	var winnerScore sql.NullInt32
	var durationSeconds sql.NullInt32

	err := s.db.QueryRow(query, gameID).Scan(
		&stats.RoomID, &stats.HostName, &stats.TotalPlayers, &stats.TotalQuestions,
		&durationSeconds, &winnerName, &winnerScore, &stats.CreatedAt,
		&stats.AvgAccuracy, &stats.AvgResponseTime,
	)
	if err != nil {
		return nil, fmt.Errorf("查詢遊戲統計失敗: %w", err)
	}

	if winnerName.Valid {
		stats.WinnerName = winnerName.String
	}
	if winnerScore.Valid {
		stats.WinnerScore = int(winnerScore.Int32)
	}
	if durationSeconds.Valid {
		stats.DurationSeconds = int(durationSeconds.Int32)
	}

	return &stats, nil
}

// StartTwoTypesGame 開始「2種人」遊戲
func (s *GameService) StartTwoTypesGame(room *models.Room) error {
	if len(room.Players) < 2 {
		return fmt.Errorf("至少需要2個玩家才能開始遊戲")
	}

	// 每次開始遊戲都重新載入題目，確保遊戲能正常進行
	room.Questions = GetRandomQuestions(room.TotalQuestions)
	if len(room.Questions) == 0 {
		return fmt.Errorf("無法載入遊戲題目")
	}

	// 重置遊戲狀態
	room.CurrentQuestion = 1
	room.Answers = make(map[string]*models.Answer)

	// 重置所有玩家分數
	for _, player := range room.Players {
		player.Score = 0
	}

	// 設定第一題的主角
	room.CurrentHost = s.SelectNextHost(room, "")
	room.NextHostOverride = ""
	room.Status = models.RoomStatusQuestionDisplay

	return nil
}

// SelectNextHost 選擇下一個主角（輪流，排除房間主持人）
func (s *GameService) SelectNextHost(room *models.Room, currentHost string) string {
	// 只從非主持人玩家中選擇
	allPlayers := room.GetPlayerList()
	players := make([]*models.Player, 0, len(allPlayers))
	for _, p := range allPlayers {
		if !p.IsHost {
			players = append(players, p)
		}
	}

	if len(players) == 0 {
		return ""
	}

	// 如果是第一題，隨機選擇
	if currentHost == "" {
		rand.Seed(time.Now().UnixNano())
		return players[rand.Intn(len(players))].ID
	}

	// 找到當前主角的位置，選擇下一個
	for i, player := range players {
		if player.ID == currentHost {
			nextIndex := (i + 1) % len(players)
			return players[nextIndex].ID
		}
	}

	// 如果找不到當前主角，隨機選擇
	return players[rand.Intn(len(players))].ID
}

// SubmitTwoTypesAnswer 提交「2種人」答案
func (s *GameService) SubmitTwoTypesAnswer(room *models.Room, playerID, answer string, timeUsed float64) (*models.Answer, error) {
	currentQuestion := room.Questions[room.CurrentQuestion-1]

	// 檢查答案是否有效
	if answer != "A" && answer != "B" {
		return nil, fmt.Errorf("無效的答案選項")
	}

	// 檢查玩家是否存在
	_, exists := room.GetPlayer(playerID)
	if !exists {
		return nil, fmt.Errorf("玩家不存在")
	}

	// 創建答案記錄
	answerRecord := &models.Answer{
		PlayerID:     playerID,
		QuestionID:   currentQuestion.ID,
		Answer:       answer,
		ResponseTime: timeUsed,
		WasHost:      playerID == room.CurrentHost,
		SubmittedAt:  time.Now(),
	}

	// 如果是主角，記錄主角答案
	if playerID == room.CurrentHost {
		answerRecord.HostAnswer = answer
		answerRecord.IsCorrect = true // 主角答案永遠是"正確"的
	}

	return answerRecord, nil
}

// CalculateTwoTypesScores 計算「2種人」遊戲分數
func (s *GameService) CalculateTwoTypesScores(room *models.Room, answers map[string]*models.Answer) []models.ScoreInfo {
	log.Printf("🔢 === 開始計算第 %d 題分數 ===", room.CurrentQuestion)
	log.Printf("🎯 當前主角: %s", room.CurrentHost)
	log.Printf("📊 收到答案數量: %d", len(answers))

	// 找到主角的答案
	var hostAnswer string
	for playerID, answer := range answers {
		if playerID == room.CurrentHost {
			hostAnswer = answer.Answer
			log.Printf("👑 主角答案: %s (玩家ID: %s)", hostAnswer, playerID)
			break
		}
	}

	if hostAnswer == "" {
		log.Printf("⚠️ 警告: 沒有找到主角答案!")
	}

	scores := make([]models.ScoreInfo, 0, len(room.Players))

	for playerID, player := range room.Players {
		answer, hasAnswered := answers[playerID]
		scoreGained := 0

		log.Printf("👤 計算玩家分數: %s (ID: %s)", player.Name, playerID)
		log.Printf("   ├─ 是否答題: %t", hasAnswered)
		if hasAnswered {
			log.Printf("   ├─ 玩家答案: %s", answer.Answer)
			log.Printf("   ├─ 答題時間: %.2f秒", answer.ResponseTime)
		}

		if hasAnswered {
			if playerID == room.CurrentHost {
				// 主角得分邏輯：有答題就得基礎分
				scoreGained = 50
				log.Printf("   ├─ 主角基礎分: %d", scoreGained)
			} else if answer.Answer == hostAnswer {
				// 其他玩家：猜對主角答案得分，越快越高分
				baseScore := 100

				// 計算速度獎勵：最多 50 分
				timeBonus := int((float64(room.QuestionTimeLimit) - answer.ResponseTime) * 1.5)
				if timeBonus < 0 {
					timeBonus = 0
				}
				if timeBonus > 50 {
					timeBonus = 50 // 限制速度獎勵最高 50 分
				}

				scoreGained = baseScore + timeBonus
				log.Printf("   ├─ 猜對主角! 基礎分: %d, 速度獎勵: %d, 總得分: %d", baseScore, timeBonus, scoreGained)
			} else {
				log.Printf("   ├─ 猜錯主角 (答案: %s, 主角答案: %s), 得分: 0", answer.Answer, hostAnswer)
			}
			// 如果猜錯主角答案，得0分
		} else {
			log.Printf("   ├─ 未答題，得分: 0")
		}

		// 更新玩家總分
		oldScore := player.Score
		player.Score += scoreGained
		log.Printf("   └─ 分數更新: %d + %d = %d", oldScore, scoreGained, player.Score)

		scores = append(scores, models.ScoreInfo{
			PlayerID:       playerID,
			PlayerName:     player.Name,
			Score:          player.Score,
			ScoreGained:    scoreGained,
			CorrectAnswers: player.CorrectAnswers,
		})

		// 更新答案記錄
		if hasAnswered {
			answer.ScoreGained = scoreGained
			if playerID != room.CurrentHost {
				answer.IsCorrect = (answer.Answer == hostAnswer)
				// 更新玩家累計答對次數
				if answer.IsCorrect {
					player.CorrectAnswers++
					log.Printf("   └─ 玩家 %s 答對，累計答對: %d", player.Name, player.CorrectAnswers)
				}
			}
		}
	}

	log.Printf("📊 排序前的分數:")
	for i, score := range scores {
		log.Printf("   %d. %s: %d分 (本題+%d)", i+1, score.PlayerName, score.Score, score.ScoreGained)
	}

	// 按總分排序
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].Score > scores[i].Score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// 設置排名
	for i := range scores {
		scores[i].Rank = i + 1
	}

	log.Printf("🏆 排序後的排名:")
	for _, score := range scores {
		log.Printf("   第%d名: %s - %d分 (本題+%d)", score.Rank, score.PlayerName, score.Score, score.ScoreGained)
	}
	log.Printf("🔢 === 第 %d 題分數計算完成 ===", room.CurrentQuestion)

	return scores
}

// NextTwoTypesQuestion 進入下一題
func (s *GameService) NextTwoTypesQuestion(room *models.Room) {
	// 選擇下一個主角
	if room.NextHostOverride != "" {
		room.CurrentHost = room.NextHostOverride
		room.NextHostOverride = ""
	} else {
		room.CurrentHost = s.SelectNextHost(room, room.CurrentHost)
	}

	// 增加題目編號
	room.CurrentQuestion++

	// 檢查是否遊戲結束
	if room.CurrentQuestion > room.TotalQuestions {
		room.Status = models.RoomStatusFinished
	} else {
		room.Status = models.RoomStatusQuestionDisplay
	}
}

// GetFinalRanking 獲取最終排名
func (s *GameService) GetFinalRanking(room *models.Room) []models.PlayerGameStats {
	log.Printf("🏁 === 遊戲結束 - 最終結算 ===")
	log.Printf("🎮 房間ID: %s", room.ID)
	log.Printf("📝 總題數: %d", room.TotalQuestions)
	log.Printf("👥 參與玩家數: %d", len(room.Players))
	log.Printf("📚 遊戲歷史記錄數: %d", len(room.GameHistory))

	// 計算每個玩家的詳細統計
	playerStats := s.calculatePlayerGameStats(room)

	log.Printf("🏆 最終排名與統計:")
	for i, stats := range playerStats {
		log.Printf("   第%d名: %s - %d分", i+1, stats.PlayerName, stats.TotalScore)
		log.Printf("      ├─ 當主角: %d次", stats.AsHost)
		log.Printf("      ├─ 當猜測者: %d次", stats.AsGuesser)
		log.Printf("      ├─ 猜對次數: %d次", stats.CorrectGuesses)
		log.Printf("      └─ 猜測正確率: %.1f%%", stats.GuessAccuracy)
	}

	log.Printf("🏁 === 最終結算完成 ===")
	return playerStats
}

// calculatePlayerGameStats 計算玩家遊戲統計
func (s *GameService) calculatePlayerGameStats(room *models.Room) []models.PlayerGameStats {
	statsMap := make(map[string]*models.PlayerGameStats)

	// 初始化每個玩家的統計
	for playerID, player := range room.Players {
		statsMap[playerID] = &models.PlayerGameStats{
			PlayerID:       playerID,
			PlayerName:     player.Name,
			TotalScore:     player.Score,
			TotalQuestions: room.TotalQuestions,
			AsHost:         0,
			AsGuesser:      0,
			CorrectGuesses: 0,
			GuessAccuracy:  0.0,
		}
	}

	// 分析每題的歷史記錄
	for _, history := range room.GameHistory {
		log.Printf("📊 分析第%d題統計:", history.QuestionNum)
		log.Printf("   主角: %s, 主角答案: %s", history.HostPlayerID, history.HostAnswer)

		for playerID, answer := range history.PlayerAnswers {
			if stats, exists := statsMap[playerID]; exists {
				if answer.WasHost {
					// 當主角
					stats.AsHost++
					log.Printf("   玩家 %s: 當主角", stats.PlayerName)
				} else {
					// 當猜測者
					stats.AsGuesser++
					if answer.IsCorrect {
						stats.CorrectGuesses++
						log.Printf("   玩家 %s: 猜對 (答案:%s)", stats.PlayerName, answer.Answer)
					} else {
						log.Printf("   玩家 %s: 猜錯 (答案:%s, 主角答案:%s)", stats.PlayerName, answer.Answer, history.HostAnswer)
					}
				}
			}
		}
	}

	// 計算猜測正確率並轉換為數組
	result := make([]models.PlayerGameStats, 0, len(statsMap))
	for _, stats := range statsMap {
		// 計算猜測正確率 (只計算猜測部分，不包括當主角)
		if stats.AsGuesser > 0 {
			stats.GuessAccuracy = float64(stats.CorrectGuesses) / float64(stats.AsGuesser) * 100
		} else {
			stats.GuessAccuracy = 0.0
		}

		log.Printf("🔢 玩家 %s 最終統計:", stats.PlayerName)
		log.Printf("   ├─ 總分: %d", stats.TotalScore)
		log.Printf("   ├─ 當主角: %d/%d", stats.AsHost, stats.TotalQuestions)
		log.Printf("   ├─ 當猜測者: %d/%d", stats.AsGuesser, stats.TotalQuestions)
		log.Printf("   ├─ 猜對次數: %d/%d", stats.CorrectGuesses, stats.AsGuesser)
		log.Printf("   └─ 猜測正確率: %.1f%%", stats.GuessAccuracy)

		result = append(result, *stats)
	}

	// 按總分排序
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].TotalScore > result[i].TotalScore {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	// 設置排名
	for i := range result {
		result[i].Rank = i + 1
	}

	return result
}
