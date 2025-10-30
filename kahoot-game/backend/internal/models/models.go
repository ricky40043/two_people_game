package models

import (
	"time"
)

// Player 玩家結構
type Player struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	RoomID       string    `json:"roomId"`
	Score        int       `json:"score"`
	IsHost       bool      `json:"isHost"`       // 是否為房間主持人
	IsConnected  bool      `json:"isConnected"`
	LastActivity time.Time `json:"lastActivity"`
	SocketConn   interface{} `json:"-"` // WebSocket 連線，不序列化
}

// Room 房間結構
type Room struct {
	ID                string            `json:"id"`
	HostID            string            `json:"hostId"`
	HostName          string            `json:"hostName"`
	Status            RoomStatus        `json:"status"`
	Players           map[string]*Player `json:"players"`
	CurrentQuestion   int               `json:"currentQuestion"`
	TotalQuestions    int               `json:"totalQuestions"`
	QuestionTimeLimit int               `json:"questionTimeLimit"`
	CurrentHost       string            `json:"currentHost"`       // 當前題目的主角玩家
	NextHostOverride  string            `json:"nextHostOverride,omitempty"`
	TimeLeft          int               `json:"timeLeft"`
	Questions         []Question        `json:"questions"`
	Answers           map[string]*Answer `json:"answers"`           // 當前題目的玩家答案
	GameHistory       []QuestionHistory `json:"gameHistory"`       // 所有題目的答題記錄
	CreatedAt         time.Time         `json:"createdAt"`
	StartedAt         *time.Time        `json:"startedAt,omitempty"`
	FinishedAt        *time.Time        `json:"finishedAt,omitempty"`
}

// RoomStatus 房間狀態枚舉
type RoomStatus string

const (
	RoomStatusWaiting         RoomStatus = "waiting"          // 等待玩家加入
	RoomStatusStarting        RoomStatus = "starting"         // 遊戲即將開始
	RoomStatusQuestionDisplay RoomStatus = "question_display" // 顯示題目
	RoomStatusAnswering       RoomStatus = "answering"        // 答題時間
	RoomStatusShowResult      RoomStatus = "show_result"      // 顯示答案結果
	RoomStatusFinished        RoomStatus = "finished"         // 遊戲結束
)

// Question 題目結構 - 適用於「2種人」遊戲
type Question struct {
	ID            int      `json:"id" db:"id"`
	QuestionText  string   `json:"questionText" db:"question_text"`
	OptionA       string   `json:"optionA" db:"option_a"`
	OptionB       string   `json:"optionB" db:"option_b"`
	Category      string   `json:"category" db:"category"`
	TimesUsed     int      `json:"timesUsed" db:"times_used"`
	IsActive      bool     `json:"isActive" db:"is_active"`
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
}

// GetOptions 獲取題目選項陣列
func (q *Question) GetOptions() []string {
	return []string{q.OptionA, q.OptionB}
}

// Answer 答案結構 - 適用於「2種人」遊戲
type Answer struct {
	PlayerID     string  `json:"playerId"`
	QuestionID   int     `json:"questionId"`
	Answer       string  `json:"answer"`        // A 或 B
	IsCorrect    bool    `json:"isCorrect"`     // 猜測是否與主角一致
	ResponseTime float64 `json:"responseTime"`  // 答題時間（秒）
	ScoreGained  int     `json:"scoreGained"`
	WasHost      bool    `json:"wasHost"`       // 該題是否為主角
	HostAnswer   string  `json:"hostAnswer"`    // 主角的答案（只有主角有值）
	SubmittedAt  time.Time `json:"submittedAt"`
}

// Game 遊戲記錄結構
type Game struct {
	ID               int        `json:"id" db:"id"`
	RoomID           string     `json:"roomId" db:"room_id"`
	HostName         string     `json:"hostName" db:"host_name"`
	TotalPlayers     int        `json:"totalPlayers" db:"total_players"`
	TotalQuestions   int        `json:"totalQuestions" db:"total_questions"`
	QuestionTimeLimit int       `json:"questionTimeLimit" db:"question_time_limit"`
	WinnerName       *string    `json:"winnerName" db:"winner_name"`
	WinnerScore      *int       `json:"winnerScore" db:"winner_score"`
	DurationSeconds  *int       `json:"durationSeconds" db:"duration_seconds"`
	Status           string     `json:"status" db:"status"`
	CreatedAt        time.Time  `json:"createdAt" db:"created_at"`
	StartedAt        *time.Time `json:"startedAt" db:"started_at"`
	FinishedAt       *time.Time `json:"finishedAt" db:"finished_at"`
}

// PlayerStats 玩家統計結構
type PlayerStats struct {
	ID                  int     `json:"id" db:"id"`
	GameID              int     `json:"gameId" db:"game_id"`
	PlayerName          string  `json:"playerName" db:"player_name"`
	FinalScore          int     `json:"finalScore" db:"final_score"`
	FinalRank           int     `json:"finalRank" db:"final_rank"`
	CorrectAnswers      int     `json:"correctAnswers" db:"correct_answers"`
	TotalAnswers        int     `json:"totalAnswers" db:"total_answers"`
	AccuracyPercentage  float64 `json:"accuracyPercentage" db:"accuracy_percentage"`
	AvgResponseTime     float64 `json:"avgResponseTime" db:"avg_response_time"`
	TimesAsHost         int     `json:"timesAsHost" db:"times_as_host"`
	FastestAnswerTime   *float64 `json:"fastestAnswerTime" db:"fastest_answer_time"`
	SlowestAnswerTime   *float64 `json:"slowestAnswerTime" db:"slowest_answer_time"`
	CreatedAt           time.Time `json:"createdAt" db:"created_at"`
}

// ScoreInfo 計分資訊
type ScoreInfo struct {
	PlayerID    string `json:"playerId"`
	PlayerName  string `json:"playerName"`
	Score       int    `json:"score"`
	Rank        int    `json:"rank"`
	ScoreGained int    `json:"scoreGained"`
}

// QuestionHistory 題目歷史記錄
type QuestionHistory struct {
	QuestionID   int                    `json:"questionId"`
	QuestionNum  int                    `json:"questionNum"`
	HostPlayerID string                 `json:"hostPlayerId"`
	HostAnswer   string                 `json:"hostAnswer"`
	PlayerAnswers map[string]*Answer    `json:"playerAnswers"`
}

// PlayerGameStats 玩家遊戲統計
type PlayerGameStats struct {
	PlayerID        string  `json:"playerId"`
	PlayerName      string  `json:"playerName"`
	TotalScore      int     `json:"totalScore"`
	Rank            int     `json:"rank"`
	TotalQuestions  int     `json:"totalQuestions"`
	AsHost          int     `json:"asHost"`          // 當主角次數
	AsGuesser       int     `json:"asGuesser"`       // 當猜測者次數
	CorrectGuesses  int     `json:"correctGuesses"`  // 猜對次數 (只計算猜測部分)
	GuessAccuracy   float64 `json:"guessAccuracy"`   // 猜測正確率 (只計算猜測部分)
}

// QuestionResult 題目結果
type QuestionResult struct {
	QuestionID    int                    `json:"questionId"`
	CorrectAnswer string                 `json:"correctAnswer"`
	Explanation   string                 `json:"explanation"`
	PlayerAnswers map[string]Answer      `json:"playerAnswers"`
	Scores        []ScoreInfo            `json:"scores"`
	NextHost      string                 `json:"nextHost"`
}

// GameStatistics 遊戲統計
type GameStatistics struct {
	RoomID           string    `json:"roomId"`
	HostName         string    `json:"hostName"`
	TotalPlayers     int       `json:"totalPlayers"`
	TotalQuestions   int       `json:"totalQuestions"`
	DurationSeconds  int       `json:"durationSeconds"`
	WinnerName       string    `json:"winnerName"`
	WinnerScore      int       `json:"winnerScore"`
	AvgAccuracy      float64   `json:"avgAccuracy"`
	AvgResponseTime  float64   `json:"avgResponseTime"`
	CreatedAt        time.Time `json:"createdAt"`
}

// CreateRoomRequest 創建房間請求
type CreateRoomRequest struct {
	HostName          string `json:"hostName" binding:"required,min=1,max=50"`
	TotalQuestions    int    `json:"totalQuestions" binding:"min=1,max=50"`
	QuestionTimeLimit int    `json:"questionTimeLimit" binding:"min=10,max=120"`
}

// JoinRoomRequest 加入房間請求
type JoinRoomRequest struct {
	RoomID     string `json:"roomId" binding:"required,len=6"`
	PlayerName string `json:"playerName" binding:"required,min=1,max=50"`
}

// SubmitAnswerRequest 提交答案請求
type SubmitAnswerRequest struct {
	RoomID       string  `json:"roomId" binding:"required"`
	QuestionID   int     `json:"questionId" binding:"required"`
	Answer       string  `json:"answer" binding:"required,oneof=A B"`
	TimeUsed     float64 `json:"timeUsed" binding:"min=0"`
}

// CreateQuestionRequest 創建題目請求 - 適用於「2種人」遊戲
type CreateQuestionRequest struct {
	QuestionText  string `json:"questionText" binding:"required,min=5,max=500"`
	OptionA       string `json:"optionA" binding:"required,min=1,max=200"`
	OptionB       string `json:"optionB" binding:"required,min=1,max=200"`
	Category      string `json:"category" binding:"max=50"`
}

// RoomInfo 房間資訊（用於列表顯示）
type RoomInfo struct {
	ID               string     `json:"id"`
	HostName         string     `json:"hostName"`
	Status           RoomStatus `json:"status"`
	PlayerCount      int        `json:"playerCount"`
	MaxPlayers       int        `json:"maxPlayers"`
	CurrentQuestion  int        `json:"currentQuestion"`
	TotalQuestions   int        `json:"totalQuestions"`
	CreatedAt        time.Time  `json:"createdAt"`
}

// GetPlayerCount 獲取房間玩家數量
func (r *Room) GetPlayerCount() int {
	return len(r.Players)
}

// AddPlayer 添加玩家到房間
func (r *Room) AddPlayer(player *Player) {
	if r.Players == nil {
		r.Players = make(map[string]*Player)
	}
	r.Players[player.ID] = player
}

// RemovePlayer 從房間移除玩家
func (r *Room) RemovePlayer(playerID string) {
	delete(r.Players, playerID)
}

// GetPlayer 獲取房間中的玩家
func (r *Room) GetPlayer(playerID string) (*Player, bool) {
	player, exists := r.Players[playerID]
	return player, exists
}

// GetPlayerList 獲取房間玩家列表（陣列形式）
func (r *Room) GetPlayerList() []*Player {
	players := make([]*Player, 0, len(r.Players))
	for _, player := range r.Players {
		players = append(players, player)
	}
	return players
}

// GetSortedPlayersByScore 按分數排序獲取玩家列表
func (r *Room) GetSortedPlayersByScore() []ScoreInfo {
	scores := make([]ScoreInfo, 0, len(r.Players))
	
	for _, player := range r.Players {
		scores = append(scores, ScoreInfo{
			PlayerID:   player.ID,
			PlayerName: player.Name,
			Score:      player.Score,
		})
	}
	
	// 按分數降序排序
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
	
	return scores
}
