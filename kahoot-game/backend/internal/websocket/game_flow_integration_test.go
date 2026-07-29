package websocket

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"kahoot-game/internal/services"
	"kahoot-game/internal/stats"

	gws "github.com/gorilla/websocket"
)

// TestMain 讓整包 websocket 測試跑得快，而且不會把假對局送到正式統計服務。
func TestMain(m *testing.M) {
	stats.Disable()

	// 正式是 5 秒 / 3 秒，測試縮到毫秒級，整場 5 題才跑得完
	resultDisplayDelay = 200 * time.Millisecond
	skipQuestionDelay = 200 * time.Millisecond

	os.Exit(m.Run())
}

const (
	testTotalQuestions = 5
	testPlayerCount    = 5
	// 每題倒數留 5 秒，足夠讓「斷線 → 重連 → 作答」在同一題內完成
	testQuestionTimeLimit = 5
)

// ---------------------------------------------------------------------------
// 測試用的伺服器與連線工具
// ---------------------------------------------------------------------------

type envelope struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

type harness struct {
	hub   *Hub
	srv   *httptest.Server
	wsURL string
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	// redisClient / db 都傳 nil → 服務層走記憶體模式，不需要外部依賴
	gameService := services.NewGameService(nil, nil)
	roomService := services.NewRoomService(nil, gameService)
	hub := NewHub(roomService, gameService, "http://localhost:3333")
	go hub.Run()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWS(hub, w, r)
	}))
	t.Cleanup(srv.Close)

	return &harness{
		hub:   hub,
		srv:   srv,
		wsURL: "ws" + strings.TrimPrefix(srv.URL, "http"),
	}
}

// client 是一條測試用的 WebSocket 連線。
// 它可以跨越多次「斷線 → 重連」，events 通道會一直沿用，
// 這樣重連前後收到的訊息才能在同一條時間線上被檢查。
type client struct {
	name string
	url  string

	mu   sync.Mutex
	conn *gws.Conn

	events chan envelope
	// pending 只由持有這個 client 的那條 goroutine 存取，不需要鎖
	pending []envelope

	playerID  string
	roomID    string
	hostToken string
}

func newClient(name, url string) *client {
	return &client{
		name:   name,
		url:    url,
		events: make(chan envelope, 1024),
	}
}

func (c *client) connect() error {
	conn, _, err := gws.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("%s: 建立連線失敗: %w", c.name, err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	go c.readLoop(conn)
	return nil
}

// readLoop 只讀自己那條連線，連線關閉就結束，不碰 c.conn
func (c *client) readLoop(conn *gws.Conn) {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}
		select {
		case c.events <- env:
		default:
			// 緩衝區設得夠大，滿了代表測試邏輯有問題，讓後續 await 逾時失敗
		}
	}
}

func (c *client) currentConn() *gws.Conn {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn
}

func (c *client) send(msgType string, data map[string]interface{}) error {
	conn := c.currentConn()
	if conn == nil {
		return fmt.Errorf("%s: 尚未連線，無法送出 %s", c.name, msgType)
	}
	if err := conn.WriteJSON(map[string]interface{}{"type": msgType, "data": data}); err != nil {
		return fmt.Errorf("%s: 送出 %s 失敗: %w", c.name, msgType, err)
	}
	return nil
}

func (c *client) closeConn() {
	c.mu.Lock()
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
}

// next 取出下一個事件，優先吃 await 期間暫存下來的
func (c *client) next(timeout time.Duration) (envelope, error) {
	if len(c.pending) > 0 {
		env := c.pending[0]
		c.pending = c.pending[1:]
		return env, nil
	}
	select {
	case env := <-c.events:
		return env, nil
	case <-time.After(timeout):
		return envelope{}, fmt.Errorf("%s: 等待事件逾時 (%s)", c.name, timeout)
	}
}

// await 等待指定型別的事件；期間收到的其他事件會照原順序保留下來，
// 避免把 NEW_QUESTION / GAME_FINISHED 這類關鍵訊息吃掉。
func (c *client) await(msgType string, timeout time.Duration) (envelope, error) {
	deadline := time.Now().Add(timeout)
	var held []envelope

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			c.pending = append(held, c.pending...)
			return envelope{}, fmt.Errorf("%s: 等待 %s 逾時", c.name, msgType)
		}

		env, err := c.next(remaining)
		if err != nil {
			c.pending = append(held, c.pending...)
			return envelope{}, fmt.Errorf("%s: 等待 %s 時 %w", c.name, msgType, err)
		}

		if env.Type == msgType {
			c.pending = append(held, c.pending...)
			return env, nil
		}
		if env.Type == "ERROR" {
			// 錯誤訊息保留下來，最後統一檢查
			held = append(held, env)
			continue
		}
		held = append(held, env)
	}
}

func str(data map[string]interface{}, key string) string {
	v, _ := data[key].(string)
	return v
}

func num(data map[string]interface{}, key string) float64 {
	v, _ := data[key].(float64)
	return v
}

// ---------------------------------------------------------------------------
// 主要整合測試：1 個房間 + 5 位玩家 + 5 題，期間隨機斷線重連
// ---------------------------------------------------------------------------

// disconnectMode 描述某位玩家在某一題要怎麼玩
type disconnectMode int

const (
	modeNormal disconnectMode = iota
	// 先關掉舊連線，等一下再重連（一般的閃退 / 換網路）
	modeDropThenRejoin
	// 先建立新連線並 REJOIN 成功，才關掉舊連線。
	// 這會重現「舊連線比新連線更晚被伺服器註銷」的競態，
	// 若 hub 沒有擋住，剛回來的玩家會被舊連線標記成離線並刪掉答案。
	modeGhostConnection
)

type questionPlan struct {
	mode        disconnectMode
	answer      string
	answerDelay time.Duration
	offlineFor  time.Duration
}

func TestFullGameFlow_FivePlayers_FiveQuestions_WithReconnects(t *testing.T) {
	seed := int64(20260729)
	if v := os.Getenv("GAME_TEST_SEED"); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			t.Fatalf("GAME_TEST_SEED 不是合法數字: %v", err)
		}
		seed = parsed
	}
	t.Logf("使用亂數種子 seed=%d（失敗時可用 GAME_TEST_SEED=%d 重現）", seed, seed)
	rng := rand.New(rand.NewSource(seed))

	h := newHarness(t)

	// --- 1. 房主開房 ---------------------------------------------------
	host := newClient("房主", h.wsURL)
	if err := host.connect(); err != nil {
		t.Fatalf("房主連線失敗: %v", err)
	}
	defer host.closeConn()

	if _, err := host.await("CONNECTED", 5*time.Second); err != nil {
		t.Fatalf("房主未收到 CONNECTED: %v", err)
	}
	if err := host.send("CREATE_ROOM", map[string]interface{}{
		"hostName":          "房主",
		"totalQuestions":    testTotalQuestions,
		"questionTimeLimit": testQuestionTimeLimit,
	}); err != nil {
		t.Fatalf("送出 CREATE_ROOM 失敗: %v", err)
	}

	created, err := host.await("ROOM_CREATED", 5*time.Second)
	if err != nil {
		t.Fatalf("未收到 ROOM_CREATED: %v", err)
	}
	roomID := str(created.Data, "roomId")
	if roomID == "" {
		t.Fatalf("ROOM_CREATED 沒有帶回 roomId: %+v", created.Data)
	}
	host.roomID = roomID
	host.playerID = str(created.Data, "clientId")
	host.hostToken = str(created.Data, "hostToken")
	t.Logf("房間建立成功: %s", roomID)

	// --- 2. 五位玩家依序加入 -------------------------------------------
	players := make([]*client, 0, testPlayerCount)
	for i := 1; i <= testPlayerCount; i++ {
		p := newClient(fmt.Sprintf("玩家%d", i), h.wsURL)
		if err := p.connect(); err != nil {
			t.Fatalf("%s 連線失敗: %v", p.name, err)
		}
		defer p.closeConn()

		connected, err := p.await("CONNECTED", 5*time.Second)
		if err != nil {
			t.Fatalf("%s 未收到 CONNECTED: %v", p.name, err)
		}
		p.playerID = str(connected.Data, "clientId")
		p.roomID = roomID

		if err := p.send("JOIN_ROOM", map[string]interface{}{
			"roomId":     roomID,
			"playerName": p.name,
		}); err != nil {
			t.Fatalf("%s 送出 JOIN_ROOM 失敗: %v", p.name, err)
		}

		joined, err := p.await("PLAYER_JOINED", 5*time.Second)
		if err != nil {
			t.Fatalf("%s 加入房間失敗: %v", p.name, err)
		}
		if got := str(joined.Data, "playerId"); got != p.playerID {
			t.Fatalf("%s 的 playerId 與 CONNECTED 不一致: %s vs %s", p.name, got, p.playerID)
		}
		players = append(players, p)
	}

	// --- 3. 先排好每位玩家每一題的行為（在主 goroutine 產生，避免 rand 競態） ---
	plans := make([][]questionPlan, len(players))
	for i := range players {
		plans[i] = make([]questionPlan, testTotalQuestions)
		for q := 0; q < testTotalQuestions; q++ {
			mode := modeNormal
			switch rng.Intn(4) {
			case 0:
				mode = modeDropThenRejoin
			case 1:
				mode = modeGhostConnection
			}
			answer := "A"
			if rng.Intn(2) == 1 {
				answer = "B"
			}
			plans[i][q] = questionPlan{
				mode:        mode,
				answer:      answer,
				answerDelay: time.Duration(50+rng.Intn(250)) * time.Millisecond,
				offlineFor:  time.Duration(80+rng.Intn(200)) * time.Millisecond,
			}
		}
	}

	// --- 4. 開始遊戲 ---------------------------------------------------
	if err := host.send("START_GAME", map[string]interface{}{"roomId": roomID}); err != nil {
		t.Fatalf("送出 START_GAME 失敗: %v", err)
	}

	type result struct {
		name          string
		questionsSeen int
		finished      bool
		err           error
	}
	results := make(chan result, len(players)+1)

	// 房主只旁觀，不作答（房主在計分邏輯裡不算作答者）
	go func() {
		res := result{name: host.name}
		deadline := time.Now().Add(90 * time.Second)
		for time.Now().Before(deadline) {
			env, err := host.next(time.Until(deadline))
			if err != nil {
				res.err = err
				break
			}
			switch env.Type {
			case "NEW_QUESTION":
				res.questionsSeen++
			case "GAME_FINISHED":
				res.finished = true
				host.pending = append([]envelope{env}, host.pending...)
				results <- res
				return
			}
		}
		if res.err == nil {
			res.err = fmt.Errorf("%s: 等不到 GAME_FINISHED", host.name)
		}
		results <- res
	}()

	for i, p := range players {
		go func(p *client, plan []questionPlan) {
			res := result{name: p.name}
			deadline := time.Now().Add(90 * time.Second)

			for time.Now().Before(deadline) {
				env, err := p.next(time.Until(deadline))
				if err != nil {
					res.err = err
					break
				}

				switch env.Type {
				case "GAME_FINISHED":
					res.finished = true
					results <- res
					return

				case "NEW_QUESTION":
					idx := res.questionsSeen
					res.questionsSeen++
					if idx >= len(plan) {
						// 伺服器發出的題目比設定還多
						res.err = fmt.Errorf("%s: 收到第 %d 題，超出預期的 %d 題", p.name, idx+1, len(plan))
						results <- res
						return
					}
					questionID := num(env.Data, "questionId")
					step := plan[idx]

					if err := p.playQuestion(step, questionID); err != nil {
						res.err = err
						results <- res
						return
					}
				}
			}
			if res.err == nil {
				res.err = fmt.Errorf("%s: 等不到 GAME_FINISHED", p.name)
			}
			results <- res
		}(p, plans[i])
	}

	// --- 5. 收集結果 ---------------------------------------------------
	finishedCount := 0
	for i := 0; i < len(players)+1; i++ {
		res := <-results
		if res.err != nil {
			t.Errorf("%s 失敗: %v", res.name, res.err)
			continue
		}
		if !res.finished {
			t.Errorf("%s 沒有收到 GAME_FINISHED", res.name)
			continue
		}
		if res.questionsSeen != testTotalQuestions {
			t.Errorf("%s 只收到 %d 題，預期 %d 題", res.name, res.questionsSeen, testTotalQuestions)
		}
		finishedCount++
	}
	if t.Failed() {
		t.FailNow()
	}
	if finishedCount != len(players)+1 {
		t.Fatalf("只有 %d/%d 個連線完成遊戲", finishedCount, len(players)+1)
	}

	// --- 6. 驗收最終分數 -----------------------------------------------
	finished, err := host.await("GAME_FINISHED", 5*time.Second)
	if err != nil {
		t.Fatalf("房主取不到 GAME_FINISHED: %v", err)
	}

	rawStats, ok := finished.Data["finalStats"].([]interface{})
	if !ok {
		t.Fatalf("GAME_FINISHED 沒有帶 finalStats: %+v", finished.Data)
	}

	// 房主自己也在 Players 裡，所以是 5 位玩家 + 1 位房主
	if len(rawStats) != testPlayerCount+1 {
		t.Fatalf("finalStats 有 %d 筆，預期 %d 筆", len(rawStats), testPlayerCount+1)
	}

	seenNames := map[string]bool{}
	seenRanks := map[int]bool{}
	totalScore := 0
	scoredPlayers := 0

	for _, raw := range rawStats {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			t.Fatalf("finalStats 內容格式錯誤: %+v", raw)
		}
		name := str(entry, "playerName")
		if name == "" {
			t.Errorf("finalStats 有一筆缺少 playerName: %+v", entry)
		}
		if seenNames[name] {
			t.Errorf("finalStats 出現重複的玩家: %s", name)
		}
		seenNames[name] = true

		rank := int(num(entry, "rank"))
		if rank < 1 || rank > len(rawStats) {
			t.Errorf("%s 的名次不合法: %d", name, rank)
		}
		if seenRanks[rank] {
			t.Errorf("名次 %d 重複出現", rank)
		}
		seenRanks[rank] = true

		score := int(num(entry, "totalScore"))
		if score < 0 {
			t.Errorf("%s 的分數是負數: %d", name, score)
		}
		totalScore += score
		if score > 0 {
			scoredPlayers++
		}

		t.Logf("第%d名 %s：%d 分，猜對 %d 題", rank, name, score, int(num(entry, "correctGuesses")))
	}

	for i := 1; i <= testPlayerCount; i++ {
		name := fmt.Sprintf("玩家%d", i)
		if !seenNames[name] {
			t.Errorf("最終結算少了 %s", name)
		}
	}

	// 「所有分數都出來」的核心驗收：整場必須真的有計分，
	// 而不是每題都因為斷線 / 主角沒作答而作廢。
	if totalScore <= 0 {
		t.Fatalf("整場總分是 0，代表沒有任何一題成功計分")
	}
	if scoredPlayers < 2 {
		t.Errorf("只有 %d 位玩家拿到分數，計分流程可能不正常", scoredPlayers)
	}

	// 房間狀態必須真的收在 finished
	room, err := h.hub.roomService.GetRoom(roomID)
	if err != nil {
		t.Fatalf("取不到房間 %s: %v", roomID, err)
	}
	if string(room.Status) != "finished" {
		t.Errorf("房間最終狀態是 %s，預期 finished", room.Status)
	}
	if len(room.GameHistory) == 0 {
		t.Errorf("房間沒有留下任何一題的歷史紀錄")
	}

	// 核心不變量：每一題的歷史紀錄裡，不能混進別題的答案。
	// 上一題殘留 / 遲到的作答若被算進下一題，這裡就會抓到。
	for i, hist := range room.GameHistory {
		if hist.QuestionNum != i+1 {
			t.Errorf("歷史紀錄第 %d 筆的題號是 %d，預期 %d", i+1, hist.QuestionNum, i+1)
		}
		for pid, ans := range hist.PlayerAnswers {
			if ans.QuestionID != hist.QuestionID {
				t.Errorf("第 %d 題的結算混進了別題的答案：玩家 %s 的答案屬於題目 %d，但本題是題目 %d",
					hist.QuestionNum, pid, ans.QuestionID, hist.QuestionID)
			}
		}
	}

	t.Logf("整場結束：%d 題、%d 位玩家、總分 %d、歷史紀錄 %d 題",
		testTotalQuestions, testPlayerCount, totalScore, len(room.GameHistory))
}

// playQuestion 依照排好的劇本處理一題：視情況斷線重連，然後送出答案
func (c *client) playQuestion(step questionPlan, questionID float64) error {
	switch step.mode {
	case modeDropThenRejoin:
		c.closeConn()
		time.Sleep(step.offlineFor)
		if err := c.rejoin(); err != nil {
			return err
		}

	case modeGhostConnection:
		old := c.currentConn()
		// 注意順序：先建立新連線並完成 REJOIN，才關掉舊的，
		// 這樣舊連線的註銷一定發生在重連之後
		if err := c.rejoin(); err != nil {
			return err
		}
		if old != nil {
			_ = old.Close()
		}
		// 給伺服器一點時間處理舊連線的註銷
		time.Sleep(step.offlineFor)
	}

	time.Sleep(step.answerDelay)

	return c.send("SUBMIT_ANSWER", map[string]interface{}{
		"roomId":     c.roomID,
		"questionId": questionID,
		"answer":     step.answer,
		"timeUsed":   1.0,
	})
}

// rejoin 建立一條新連線並重新加入房間（沿用原本的 playerID）
func (c *client) rejoin() error {
	if err := c.connect(); err != nil {
		return err
	}
	if _, err := c.await("CONNECTED", 5*time.Second); err != nil {
		return err
	}
	if err := c.send("REJOIN_ROOM", map[string]interface{}{
		"roomId":    c.roomID,
		"playerId":  c.playerID,
		"hostToken": c.hostToken,
	}); err != nil {
		return err
	}
	if _, err := c.await("REJOIN_SUCCESS", 10*time.Second); err != nil {
		return fmt.Errorf("%s 重連失敗: %w", c.name, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 針對「舊連線晚註銷」競態的專門回歸測試
// ---------------------------------------------------------------------------

// TestStaleConnectionDoesNotEvictReconnectedPlayer 重現閃退後立刻重連的情境：
// 舊的 WebSocket 稍後才被伺服器發現斷開，這時不能把剛回來的玩家標記成離線、
// 也不能刪掉他重連後送出的答案。
func TestStaleConnectionDoesNotEvictReconnectedPlayer(t *testing.T) {
	h := newHarness(t)

	host := newClient("房主", h.wsURL)
	if err := host.connect(); err != nil {
		t.Fatalf("房主連線失敗: %v", err)
	}
	defer host.closeConn()
	if _, err := host.await("CONNECTED", 5*time.Second); err != nil {
		t.Fatalf("房主未收到 CONNECTED: %v", err)
	}
	if err := host.send("CREATE_ROOM", map[string]interface{}{
		"hostName":          "房主",
		"totalQuestions":    testTotalQuestions,
		"questionTimeLimit": 30,
	}); err != nil {
		t.Fatalf("送出 CREATE_ROOM 失敗: %v", err)
	}
	created, err := host.await("ROOM_CREATED", 5*time.Second)
	if err != nil {
		t.Fatalf("未收到 ROOM_CREATED: %v", err)
	}
	roomID := str(created.Data, "roomId")

	// 兩位玩家就夠開場
	joined := make([]*client, 0, 2)
	for i := 1; i <= 2; i++ {
		p := newClient(fmt.Sprintf("玩家%d", i), h.wsURL)
		if err := p.connect(); err != nil {
			t.Fatalf("%s 連線失敗: %v", p.name, err)
		}
		defer p.closeConn()
		connected, err := p.await("CONNECTED", 5*time.Second)
		if err != nil {
			t.Fatalf("%s 未收到 CONNECTED: %v", p.name, err)
		}
		p.playerID = str(connected.Data, "clientId")
		p.roomID = roomID
		if err := p.send("JOIN_ROOM", map[string]interface{}{
			"roomId":     roomID,
			"playerName": p.name,
		}); err != nil {
			t.Fatalf("%s 送出 JOIN_ROOM 失敗: %v", p.name, err)
		}
		if _, err := p.await("PLAYER_JOINED", 5*time.Second); err != nil {
			t.Fatalf("%s 加入房間失敗: %v", p.name, err)
		}
		joined = append(joined, p)
	}

	if err := host.send("START_GAME", map[string]interface{}{"roomId": roomID}); err != nil {
		t.Fatalf("送出 START_GAME 失敗: %v", err)
	}

	victim := joined[0]
	q, err := victim.await("NEW_QUESTION", 10*time.Second)
	if err != nil {
		t.Fatalf("%s 沒收到第一題: %v", victim.name, err)
	}
	questionID := num(q.Data, "questionId")

	// 關鍵順序：先重連成功，才關掉舊連線
	oldConn := victim.currentConn()
	if err := victim.rejoin(); err != nil {
		t.Fatalf("重連失敗: %v", err)
	}
	if err := victim.send("SUBMIT_ANSWER", map[string]interface{}{
		"roomId":     roomID,
		"questionId": questionID,
		"answer":     "A",
		"timeUsed":   1.0,
	}); err != nil {
		t.Fatalf("重連後送出答案失敗: %v", err)
	}
	if _, err := victim.await("ANSWER_SUBMITTED", 5*time.Second); err != nil {
		t.Fatalf("重連後的答案沒有被接受: %v", err)
	}

	// 現在才讓舊連線斷開，觸發那筆遲到的註銷
	if oldConn != nil {
		_ = oldConn.Close()
	}
	time.Sleep(500 * time.Millisecond)

	room, err := h.hub.roomService.GetRoom(roomID)
	if err != nil {
		t.Fatalf("取不到房間: %v", err)
	}

	player, exists := room.Players[victim.playerID]
	if !exists {
		t.Fatalf("%s 在舊連線註銷後從房間裡消失了", victim.name)
	}
	if !player.IsConnected {
		t.Errorf("%s 已經重連成功，卻被舊連線的註銷標記成離線", victim.name)
	}
	if _, answered := room.Answers[victim.playerID]; !answered {
		t.Errorf("%s 重連後送出的答案被舊連線的註銷流程刪掉了", victim.name)
	}
}
