package services

import (
	"fmt"
	"testing"

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
