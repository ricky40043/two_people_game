package services

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"kahoot-game/internal/models"

	"github.com/go-redis/redis/v8"
)

// GameService 遊戲服務
type GameService struct {
	db          *sql.DB
	redisClient *redis.Client
}

const MaxQuestionRerolls = 3

// RerollError 是換題流程可被前端辨識的業務錯誤。
type RerollError struct {
	Code    string
	Message string
}

func (e *RerollError) Error() string {
	return e.Message
}

// RerollResult 包含換題後需要廣播給房間的狀態。
type RerollResult struct {
	Question          models.Question
	QuestionVersion   int
	QuestionTimeLimit int
	TimeLeft          int
	RemainingRerolls  int
	RerolledBy        string
	ServerTime        time.Time
	EndsAt            time.Time
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

// ResetRoomToLobby 重置房間回到大廳狀態（再來一局）
func (s *GameService) ResetRoomToLobby(room *models.Room) {
	room.Status = models.RoomStatusWaiting
	room.CurrentQuestion = 0
	room.CurrentHost = ""
	room.NextHostOverride = ""
	room.Answers = make(map[string]*models.Answer)
	room.GameHistory = make([]models.QuestionHistory, 0)
	room.Questions = make([]models.Question, 0)
	room.QuestionVersion = 0
	room.QuestionStartedAt = nil
	room.QuestionEndsAt = nil
	room.TimeLeft = 0
	room.UsedQuestionIDs = nil
	room.DiscardedQuestionIDs = nil

	// 重置所有玩家的分數與統計
	for _, player := range room.Players {
		player.Score = 0
		player.CorrectAnswers = 0
		player.TimesAsHost = 0
		player.RerollUsed = 0
	}

	log.Printf("🔄 [房間重置] 房間 %s 已重置回到大廳狀態 (人數: %d)", room.ID, len(room.Players))
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
	room.QuestionVersion = 1
	room.QuestionStartedAt = nil
	room.QuestionEndsAt = nil
	room.TimeLeft = room.QuestionTimeLimit
	room.UsedQuestionIDs = make(map[int]bool)
	room.DiscardedQuestionIDs = make([]int, 0)

	// 重置所有玩家分數與主角擔任次數
	for _, player := range room.Players {
		player.Score = 0
		player.CorrectAnswers = 0
		player.TimesAsHost = 0
		player.RerollUsed = 0
	}

	// 設定第一題的主角
	room.CurrentHost = s.SelectNextHost(room, "")
	room.NextHostOverride = ""
	room.Status = models.RoomStatusQuestionDisplay

	return nil
}

// InitializeCurrentQuestion starts the server-side clock for the current question.
// It is intentionally separate from selecting a question so a reroll can keep the
// same CurrentQuestion number while replacing only the question and deadline.
func (s *GameService) InitializeCurrentQuestion(room *models.Room) error {
	if room.CurrentQuestion < 1 || room.CurrentQuestion > len(room.Questions) {
		return fmt.Errorf("目前沒有進行中的題目")
	}

	if room.QuestionVersion < 1 {
		room.QuestionVersion = 1
	}
	if room.UsedQuestionIDs == nil {
		room.UsedQuestionIDs = make(map[int]bool)
	}
	room.UsedQuestionIDs[room.Questions[room.CurrentQuestion-1].ID] = true

	now := time.Now()
	endsAt := now.Add(time.Duration(room.QuestionTimeLimit) * time.Second)
	room.QuestionStartedAt = &now
	room.QuestionEndsAt = &endsAt
	room.TimeLeft = room.QuestionTimeLimit
	room.Status = models.RoomStatusQuestionDisplay
	return nil
}

// RerollQuestion validates and atomically replaces the current question. The
// caller must protect the room with RoomService.WithRoomLock.
func (s *GameService) RerollQuestion(room *models.Room, playerID string, questionID, questionVersion int) (*RerollResult, error) {
	if room == nil {
		return nil, &RerollError{Code: "ROOM_NOT_FOUND", Message: "房間不存在"}
	}
	if room.Status != models.RoomStatusQuestionDisplay {
		return nil, &RerollError{Code: "GAME_NOT_IN_PROGRESS", Message: "目前不是可換題的答題階段"}
	}
	if room.CurrentQuestion < 1 || room.CurrentQuestion > len(room.Questions) {
		return nil, &RerollError{Code: "QUESTION_ALREADY_FINISHED", Message: "目前題目已結束"}
	}
	player, exists := room.Players[playerID]
	if !exists || !player.IsConnected {
		return nil, &RerollError{Code: "PLAYER_NOT_FOUND", Message: "玩家不存在或目前不在線"}
	}
	if room.CurrentHost != playerID {
		return nil, &RerollError{Code: "NOT_CURRENT_HOST", Message: "只有目前題目的主角可以換題"}
	}
	if player.RerollUsed >= MaxQuestionRerolls {
		return nil, &RerollError{Code: "REROLL_LIMIT_REACHED", Message: "本場遊戲的換題次數已用完"}
	}

	currentQuestion := room.Questions[room.CurrentQuestion-1]
	if currentQuestion.ID != questionID || room.QuestionVersion != questionVersion {
		return nil, &RerollError{Code: "STALE_QUESTION", Message: "題目已更新，請重新整理後再試"}
	}
	if room.QuestionEndsAt != nil && !time.Now().Before(*room.QuestionEndsAt) {
		return nil, &RerollError{Code: "QUESTION_ALREADY_FINISHED", Message: "答題時間已結束，無法換題"}
	}

	replacement, err := s.selectRerollQuestion(room)
	if err != nil {
		return nil, err
	}

	if room.UsedQuestionIDs == nil {
		room.UsedQuestionIDs = make(map[int]bool)
	}
	room.DiscardedQuestionIDs = append(room.DiscardedQuestionIDs, currentQuestion.ID)
	room.Questions[room.CurrentQuestion-1] = replacement
	room.UsedQuestionIDs[replacement.ID] = true
	room.Answers = make(map[string]*models.Answer)
	room.QuestionVersion++
	player.RerollUsed++

	now := time.Now()
	endsAt := now.Add(time.Duration(room.QuestionTimeLimit) * time.Second)
	room.QuestionStartedAt = &now
	room.QuestionEndsAt = &endsAt
	room.TimeLeft = room.QuestionTimeLimit

	return &RerollResult{
		Question:          replacement,
		QuestionVersion:   room.QuestionVersion,
		QuestionTimeLimit: room.QuestionTimeLimit,
		TimeLeft:          room.TimeLeft,
		RemainingRerolls:  MaxQuestionRerolls - player.RerollUsed,
		RerolledBy:        playerID,
		ServerTime:        now,
		EndsAt:            endsAt,
	}, nil
}

func (s *GameService) selectRerollQuestion(room *models.Room) (models.Question, error) {
	currentQuestionID := room.Questions[room.CurrentQuestion-1].ID
	discarded := make(map[int]bool, len(room.DiscardedQuestionIDs))
	for _, id := range room.DiscardedQuestionIDs {
		discarded[id] = true
	}
	used := room.UsedQuestionIDs
	scheduled := make(map[int]bool, len(room.Questions))
	for _, question := range room.Questions {
		scheduled[question.ID] = true
	}

	choose := func(allowUsed bool) (models.Question, bool) {
		allQuestions := ConvertToGameQuestions(GetTwoTypesQuestions())
		for _, question := range allQuestions {
			if question.ID == currentQuestionID || discarded[question.ID] || scheduled[question.ID] {
				continue
			}
			if !allowUsed && used[question.ID] {
				continue
			}
			return question, true
		}
		return models.Question{}, false
	}

	if question, ok := choose(false); ok {
		return question, nil
	}
	if question, ok := choose(true); ok {
		return question, nil
	}
	return models.Question{}, &RerollError{Code: "NO_AVAILABLE_QUESTION", Message: "題庫中沒有可替換的題目"}
}

// SelectNextHost 選擇下一個主角（嚴格公平輪播，保證每個人都輪過一次後才進入下一輪）
func (s *GameService) SelectNextHost(room *models.Room, currentHost string) string {
	// 1. 只從非房間主持人中選擇
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

	// 2. 對 players 按 ID 固定排序，消弭 Map 隨機疊代不穩定問題
	sort.Slice(players, func(i, j int) bool {
		return players[i].ID < players[j].ID
	})

	// 3. 找出目前當主角次數最少者 minTimes
	minTimes := players[0].TimesAsHost
	for _, p := range players {
		if p.TimesAsHost < minTimes {
			minTimes = p.TimesAsHost
		}
	}

	// 4. 收集所有 TimesAsHost == minTimes 的候選人
	candidates := make([]*models.Player, 0)
	for _, p := range players {
		if p.TimesAsHost == minTimes {
			candidates = append(candidates, p)
		}
	}

	// 5. 在候選人中，優先選擇非 currentHost 的玩家（絕對避開連續選中同一人）
	var selected *models.Player
	for _, p := range candidates {
		if p.ID != currentHost {
			selected = p
			break
		}
	}

	// 如果所有候選人都跟 currentHost 一樣（例如全場只有 1 位玩家），則選擇 candidates[0]
	if selected == nil {
		selected = candidates[0]
	}

	// 6. 更新選中者的當主角次數
	selected.TimesAsHost++
	log.Printf("👑 [主角輪替] 選擇主角: %s (ID: %s), 第 %d 次當主角", selected.Name, selected.ID, selected.TimesAsHost)

	return selected.ID
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
				// 其他玩家：猜對主角答案得分，使用指數曲線衰減拉大時間急距
				// 0秒極速答對可拿 150 分，前幾秒顯著急降，15~30秒平緩微降至 50 分保底
				responseTime := answer.ResponseTime
				if responseTime < 0 {
					responseTime = 0
				}

				minScore := 50.0  // 保底得分
				maxBonus := 100.0 // 速度最高加成
				decayRate := 0.18 // 衰減速率

				calculatedScore := minScore + maxBonus*math.Exp(-decayRate*responseTime)
				scoreGained = int(math.Round(calculatedScore))

				log.Printf("   ├─ 猜對主角! 響應時間: %.2f秒, 曲線分數: %d (對數曲線: 150 -> 50)", responseTime, scoreGained)
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
		room.QuestionVersion++
		room.QuestionStartedAt = nil
		room.QuestionEndsAt = nil
		room.TimeLeft = 0
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
