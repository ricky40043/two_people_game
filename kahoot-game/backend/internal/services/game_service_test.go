package services

import (
	"fmt"
	"testing"
	"time"

	"kahoot-game/internal/models"
)

func TestSelectNextHost_StrictRoundRobin(t *testing.T) {
	gameService := NewGameService(nil, nil)

	room := &models.Room{
		ID:      "TEST01",
		Players: make(map[string]*models.Player),
	}

	// 4 位玩家
	players := []*models.Player{
		{ID: "p1", Name: "Player 1", IsHost: false, TimesAsHost: 0},
		{ID: "p2", Name: "Player 2", IsHost: false, TimesAsHost: 0},
		{ID: "p3", Name: "Player 3", IsHost: false, TimesAsHost: 0},
		{ID: "p4", Name: "Player 4", IsHost: false, TimesAsHost: 0},
	}

	for _, p := range players {
		room.Players[p.ID] = p
	}

	currentHost := ""
	hostSequence := make([]string, 0)

	// 進行 8 題測試
	for q := 1; q <= 8; q++ {
		nextHost := gameService.SelectNextHost(room, currentHost)
		if nextHost == "" {
			t.Fatalf("第 %d 題未能選出主角", q)
		}
		if currentHost != "" && nextHost == currentHost && len(room.Players) > 1 {
			t.Errorf("第 %d 題連續兩題由同一人當主角: %s", q, nextHost)
		}

		hostSequence = append(hostSequence, nextHost)
		currentHost = nextHost
	}

	fmt.Printf("前 8 題主角選擇順序: %v\n", hostSequence)

	// 驗證前 4 題中 4 位玩家每人恰當一次
	round1Hosts := make(map[string]int)
	for i := 0; i < 4; i++ {
		round1Hosts[hostSequence[i]]++
	}
	if len(round1Hosts) != 4 {
		t.Errorf("第一輪 4 題未做到每人擔任一次主角: %v", round1Hosts)
	}

	// 驗證後 4 題中 4 位玩家每人恰當一次
	round2Hosts := make(map[string]int)
	for i := 4; i < 8; i++ {
		round2Hosts[hostSequence[i]]++
	}
	if len(round2Hosts) != 4 {
		t.Errorf("第二輪 4 題未做到每人擔任一次主角: %v", round2Hosts)
	}
}

func newRerollTestRoom() *models.Room {
	questions := GetRandomQuestions(3)
	room := &models.Room{
		ID:                "REROLL1",
		Status:            models.RoomStatusQuestionDisplay,
		Players:           make(map[string]*models.Player),
		CurrentQuestion:   1,
		TotalQuestions:    3,
		QuestionTimeLimit: 30,
		CurrentHost:       "p1",
		QuestionVersion:   1,
		Questions:         questions,
		Answers:           map[string]*models.Answer{"p1": {PlayerID: "p1", QuestionID: questions[0].ID}},
		UsedQuestionIDs:   map[int]bool{questions[0].ID: true},
		QuestionEndsAt:    ptrTime(time.Now().Add(time.Minute)),
	}
	room.Players["p1"] = &models.Player{ID: "p1", Name: "Player 1", IsConnected: true, Score: 120}
	room.Players["p2"] = &models.Player{ID: "p2", Name: "Player 2", IsConnected: true, Score: 80}
	return room
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func TestRerollQuestion_PreservesRoundAndResetsAnswers(t *testing.T) {
	service := NewGameService(nil, nil)
	room := newRerollTestRoom()
	oldQuestionID := room.Questions[0].ID

	result, err := service.RerollQuestion(room, "p1", oldQuestionID, 1)
	if err != nil {
		t.Fatalf("第一次換題應成功: %v", err)
	}
	if result.RemainingRerolls != 2 || room.Players["p1"].RerollUsed != 1 {
		t.Fatalf("換題次數錯誤: result=%d player=%d", result.RemainingRerolls, room.Players["p1"].RerollUsed)
	}
	if room.CurrentQuestion != 1 || room.CurrentHost != "p1" {
		t.Fatalf("換題不應改變回合或主角: question=%d host=%s", room.CurrentQuestion, room.CurrentHost)
	}
	if room.QuestionVersion != 2 || room.Questions[0].ID == oldQuestionID {
		t.Fatalf("題目版本或題目未更新: version=%d question=%d", room.QuestionVersion, room.Questions[0].ID)
	}
	if len(room.Answers) != 0 || room.Players["p1"].Score != 120 || room.Players["p2"].Score != 80 {
		t.Fatalf("換題不應保留本題答案或清除分數: answers=%d scores=%d/%d", len(room.Answers), room.Players["p1"].Score, room.Players["p2"].Score)
	}
	if room.TimeLeft != room.QuestionTimeLimit || room.QuestionEndsAt == nil || !room.UsedQuestionIDs[room.Questions[0].ID] {
		t.Fatalf("換題後計時或已用題目狀態錯誤")
	}
}

func TestRerollQuestion_EnforcesPerPlayerLimitAndHost(t *testing.T) {
	service := NewGameService(nil, nil)
	room := newRerollTestRoom()
	questionID := room.Questions[0].ID

	for attempt := 0; attempt < MaxQuestionRerolls; attempt++ {
		room.Status = models.RoomStatusQuestionDisplay
		room.QuestionEndsAt = ptrTime(time.Now().Add(time.Minute))
		if _, err := service.RerollQuestion(room, "p1", room.Questions[0].ID, room.QuestionVersion); err != nil {
			t.Fatalf("第 %d 次換題應成功: %v", attempt+1, err)
		}
	}
	if room.Players["p1"].RerollUsed != MaxQuestionRerolls {
		t.Fatalf("三次換題後使用次數錯誤: %d", room.Players["p1"].RerollUsed)
	}
	if _, err := service.RerollQuestion(room, "p1", room.Questions[0].ID, room.QuestionVersion); !hasRerollCode(err, "REROLL_LIMIT_REACHED") {
		t.Fatalf("第四次換題應回傳 REROLL_LIMIT_REACHED，得到 %v", err)
	}

	room = newRerollTestRoom()
	if _, err := service.RerollQuestion(room, "p2", questionID, room.QuestionVersion); !hasRerollCode(err, "NOT_CURRENT_HOST") {
		t.Fatalf("非主角換題應被拒絕，得到 %v", err)
	}
}

func TestRerollQuestion_RejectsStaleAndFinishedQuestions(t *testing.T) {
	service := NewGameService(nil, nil)
	room := newRerollTestRoom()

	if _, err := service.RerollQuestion(room, "p1", room.Questions[0].ID, room.QuestionVersion-1); !hasRerollCode(err, "STALE_QUESTION") {
		t.Fatalf("舊版本應被拒絕，得到 %v", err)
	}
	room.Status = models.RoomStatusShowResult
	if _, err := service.RerollQuestion(room, "p1", room.Questions[0].ID, room.QuestionVersion); !hasRerollCode(err, "GAME_NOT_IN_PROGRESS") {
		t.Fatalf("結算階段應被拒絕，得到 %v", err)
	}
}

func TestRerollQuestion_ReturnsNoAvailableQuestion(t *testing.T) {
	service := NewGameService(nil, nil)
	room := newRerollTestRoom()
	allQuestions := ConvertToGameQuestions(GetTwoTypesQuestions())
	room.UsedQuestionIDs = make(map[int]bool, len(allQuestions))
	room.DiscardedQuestionIDs = make([]int, 0, len(allQuestions))
	for _, question := range allQuestions {
		room.UsedQuestionIDs[question.ID] = true
		if question.ID != room.Questions[0].ID {
			room.DiscardedQuestionIDs = append(room.DiscardedQuestionIDs, question.ID)
		}
	}
	if _, err := service.RerollQuestion(room, "p1", room.Questions[0].ID, room.QuestionVersion); !hasRerollCode(err, "NO_AVAILABLE_QUESTION") {
		t.Fatalf("題庫無可用題目應回傳 NO_AVAILABLE_QUESTION，得到 %v", err)
	}
}

func hasRerollCode(err error, code string) bool {
	rerollErr, ok := err.(*RerollError)
	return ok && rerollErr.Code == code
}
