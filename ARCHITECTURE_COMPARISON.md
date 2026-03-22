# 架構比較與改進計畫

## 一、目前架構的問題

```
目前的連線流程：

  使用者打開頁面
       │
       ▼
  WebSocket 連線建立 (onopen)
       │
       ▼
  檢查 localStorage 有沒有 ricky_game_session？
       │
    ┌──┴──┐
   有      沒有
    │       │
    ▼       ▼
  自動 REJOIN_ROOM     正常顯示首頁
  (無法拒絕，直接回舊房間)
    │
    ▼
  ❌ 問題：被綁死在舊房間
     - 想建新房間？不行，session 衝突
     - 想加入其他房間？不行，已經綁定
     - 遊戲已結束想重開？session 還指向舊房


目前的身份管理：

  Client 結構 (每個 WebSocket 連線)
  ┌─────────────────────┐
  │ ID: 每次連線新 UUID  │  ← 問題：主持人關頁面再開，ID 就變了
  │ PlayerName           │
  │ RoomID               │
  │ IsHost: bool         │  ← 問題：只在 CREATE_ROOM 時設 true
  └─────────────────────┘     REJOIN 有恢復，但前端流程不完整

  localStorage (前端)
  ┌──────────────────────────────┐
  │ roomId, playerId, playerName │  ← 問題：遊戲結束後沒清除
  │ isHost                       │  ← 只存在前端，伺服器無法驗證
  └──────────────────────────────┘


目前的房間生命週期：

  建立房間 → Redis (TTL: 24h)
       │
       ▼
  遊戲進行中
       │
       ├── 正常結束 → Status = finished → 但房間還在 Redis 裡 24h
       │
       ├── 主持人關頁面 → 標記 IsConnected=false → 遊戲卡在 playing
       │                                            沒人能結束它
       │
       └── 所有人都斷線 → Hub.rooms 移除 → 但 Redis 裡還有
                                            變成孤兒房間
```

---

## 二、Kahoot 的做法

```
Kahoot 設計理念：簡單粗暴，主持人 = 遊戲生命

  主持人 (大螢幕/投影)          玩家 (手機)
  ┌─────────────────┐          ┌─────────────┐
  │ 需要登入帳號      │          │ 不需登入      │
  │ 選題庫 → 產生 PIN │          │ 輸入 PIN + 暱稱│
  │ 控制開始/下一題    │          │ 只看到選項按鈕  │
  │ 顯示題目和排行榜   │          │ 不看題目       │
  └────────┬────────┘          └──────┬──────┘
           │                          │
           └──────── 伺服器 ───────────┘
                      │
                  主持人斷線？
                      │
                      ▼
                 遊戲立即結束
                 清理所有資源
                 不保留、不重連

  ❗ Kahoot 不需要處理孤兒房間，因為主持人就是房間的生命線
  ❗ 但我們的需求不同 — 大家都是玩家，主持人也在玩
```

---

## 三、新架構設計

### 設計原則

```
1. 重連是選擇，不是強制
   → 打開頁面如果有舊 session，「詢問」使用者要不要回去，而不是自動跳轉

2. 遊戲狀態由伺服器驅動，靠 hostToken 識別主持人
   → 不靠 Client.IsHost（會隨連線消失）
   → 主持人關頁面回來，帶 hostToken 就能恢復身份

3. 主角（當回合的焦點玩家）斷線 → 本回合強制結束，進入下一題
   → 跟畫畫遊戲一樣，畫畫的人掉了那題就沒了

4. 同一台電腦測試 → localStorage 用 sessionStorage 分身
   → 或者：不同瀏覽器 / 無痕模式
   → 後端不限制同 IP 多連線

5. 房間有明確的清理機制
```

### 前端流程改進

```
新的連線流程：

  使用者打開頁面
       │
       ▼
  WebSocket 連線建立 (onopen)
       │
       ▼
  檢查 localStorage 有沒有 ricky_game_session？
       │
    ┌──┴──┐
   有      沒有
    │       │
    ▼       ▼
  先發 CHECK_ROOM     正常顯示首頁
  (只查詢，不加入)
    │
    ▼
  伺服器回覆 ROOM_STATUS
    │
    ├── 房間存在 + 遊戲進行中 → 顯示彈窗：
    │     ┌───────────────────────────────────┐
    │     │  你之前在房間 ABC123               │
    │     │  遊戲正在進行中                     │
    │     │                                    │
    │     │  [返回遊戲]     [離開，回到首頁]     │
    │     └───────────────────────────────────┘
    │     點「返回遊戲」→ REJOIN_ROOM → 恢復狀態
    │     點「離開」→ LEAVE_ROOM → 清 localStorage → 首頁
    │
    ├── 房間存在 + 等待中 → 顯示彈窗：
    │     ┌───────────────────────────────────┐
    │     │  你之前在房間 ABC123               │
    │     │  遊戲尚未開始                       │
    │     │                                    │
    │     │  [返回房間]     [離開，回到首頁]     │
    │     └───────────────────────────────────┘
    │
    ├── 房間存在 + 已結束 → 自動清除 localStorage → 首頁
    │
    └── 房間不存在 → 自動清除 localStorage → 首頁


  建新房間/加入房間時：

  使用者點「建立房間」或「加入房間」
       │
       ▼
  localStorage 有舊 session？
       │
    ┌──┴──┐
   有      沒有
    │       │
    ▼       ▼
  自動發 LEAVE_ROOM (舊房間)    直接操作
  清除 localStorage
       │
       ▼
  發送 CREATE_ROOM 或 JOIN_ROOM
       │
       ▼
  更新 localStorage 為新房間
```

### 後端改進

```
新的 Room 模型：

  Room {
    ID                string
    HostName          string
    HostPlayerID      string        ← 建房者的 playerId
    HostToken         string        ← 新增：UUID token，只有主持人知道
    Status            string        ← waiting / playing / finished / abandoned
    Players           map[string]*Player
    LastActivity      time.Time     ← 新增：最後有人操作的時間
    CreatedAt         time.Time
    ...（其他欄位不變）
  }


新增訊息類型：

  Client → Server：
  ┌──────────────────┬──────────────────────────────────────────┐
  │ CHECK_ROOM       │ { roomId, playerId }                     │
  │                  │ 只查詢房間狀態，不加入                     │
  └──────────────────┴──────────────────────────────────────────┘

  Server → Client：
  ┌──────────────────┬──────────────────────────────────────────┐
  │ ROOM_STATUS      │ { roomId, status, playerCount,           │
  │                  │   playerExists, isHost, gameState }      │
  │                  │ 回覆房間狀態，讓前端決定要不要重連         │
  └──────────────────┴──────────────────────────────────────────┘


改進 REJOIN_ROOM：

  收到 REJOIN_ROOM { roomId, playerId, hostToken? }
       │
       ▼
  房間存在？
    │
    ├── 不存在 → 回覆 ERROR (ROOM_NOT_FOUND)
    │            前端清除 localStorage
    │
    └── 存在 →
         │
         ▼
       玩家在房間裡？
         │
         ├── 不在 → 回覆 ERROR (PLAYER_NOT_FOUND)
         │          前端清除 localStorage
         │
         └── 在 →
              │
              ▼
            有 hostToken 且匹配？
              │
              ├── 是 → 設定 Client.IsHost = true
              │        恢復完整主持人權限
              │
              └── 否 → IsHost = false（普通玩家重連）
              │
              ▼
            回覆 REJOIN_SUCCESS { ..., isHost, canEndGame }


改進「主角斷線」處理：

  遊戲進行中，當回合的主角 (CurrentHost) 斷線
       │
       ▼
  本回合立即無效
  廣播 QUESTION_INVALID { message: "主角已離開，本題跳過" }
       │
       ▼
  等待 3 秒
       │
       ▼
  選擇新主角 → 進入下一題
  (不是選新主角重出這題，是直接跳過)

  ✅ 目前的 handlePlayerLeaveInternal 已經有這段邏輯
     只需確認它穩定運作


新增「房間清掃」背景程式：

  goroutine: roomCleaner (每 60 秒執行一次)
       │
       ▼
  掃描所有房間
       │
       ├── Status == finished 且超過 10 分鐘 → 刪除
       │
       ├── Status == playing 且所有人 IsConnected == false
       │   且 LastActivity 超過 5 分鐘
       │   → 標記 Status = abandoned → 刪除
       │
       ├── Status == waiting 且 LastActivity 超過 30 分鐘
       │   → 刪除（沒人開始的房間）
       │
       └── Redis TTL 24h 兜底（不用改）
```

### 同一台電腦測試

```
問題：同一瀏覽器的 localStorage 是共用的
     → 主持人和玩家會互相覆蓋 session

解法：

  方法 1（推薦，最簡單）：
  ┌──────────────────────────────────────────┐
  │ 主持人用 Chrome，玩家用 Chrome 無痕模式    │
  │ 或主持人用 Chrome，玩家用 Safari           │
  │ → localStorage 完全獨立，不會衝突          │
  └──────────────────────────────────────────┘

  方法 2（代碼改進，未來可做）：
  ┌──────────────────────────────────────────┐
  │ URL 帶 ?session=xxx 參數                  │
  │ 每個分頁有獨立的 session key              │
  │ 例如：                                    │
  │   ricky_game_session_default              │
  │   ricky_game_session_tab2                 │
  │ → 但這會增加複雜度，先不做                 │
  └──────────────────────────────────────────┘

  方法 3（最快測試）：
  ┌──────────────────────────────────────────┐
  │ 用手機掃 QR Code 當玩家                   │
  │ 電腦當主持人                              │
  │ → 完全不同裝置，不會衝突                   │
  └──────────────────────────────────────────┘
```

---

## 四、實作步驟（確認後按順序執行）

### Step 1：後端 — Room 模型加欄位
- `models.go`: Room 加 `HostToken`, `HostPlayerID`, `LastActivity`
- `room_service.go`: CreateRoom 時生成 `HostToken` (UUID)
- 每次房間操作更新 `LastActivity`

### Step 2：後端 — 新增 CHECK_ROOM 訊息處理
- `client.go`: 新增 `handleCheckRoom()`
- 收到 `{ roomId, playerId }` → 查詢房間狀態
- 回覆 `ROOM_STATUS { roomId, status, playerExists, isHost, ... }`
- 不加入房間，只是查詢

### Step 3：後端 — 改進 REJOIN_ROOM 支援 hostToken
- `handleRejoinRoom()`: 接受可選的 `hostToken` 參數
- 如果 hostToken 匹配 → 恢復 IsHost = true
- 回覆增加 `isHost`, `canEndGame` 欄位

### Step 4：後端 — 改進 FORCE_END_GAME 支援 hostToken
- `handleForceEndGame()`: 除了檢查 `c.IsHost`，也接受 `hostToken` 驗證
- 只要 hostToken 正確就允許結束

### Step 5：後端 — 新增房間清掃 goroutine
- `hub.go` 或獨立的 `cleaner.go`
- 每 60 秒掃描一次
- 清理 finished (>10min), abandoned (全離線>5min), idle waiting (>30min)

### Step 6：前端 — 連線時改為「詢問」而非「自動重連」
- `socket.ts`: onopen 改流程
  - 有 session → 發 `CHECK_ROOM`
  - 收到 `ROOM_STATUS` → 根據狀態顯示彈窗或清除 session
- 新增「返回遊戲/離開」的確認彈窗元件

### Step 7：前端 — 支援主動離開房間和切換房間
- CREATE_ROOM / JOIN_ROOM 前，先自動 LEAVE 舊房間
- LEAVE 後清除 localStorage
- 遊戲結束 (GAME_FINISHED) 時清除 localStorage

### Step 8：前端 — localStorage 存 hostToken
- `handleRoomCreated`: 存入 `hostToken`
- `rejoinRoom`: 帶上 `hostToken`
- `forceEndGame`: 帶上 `hostToken`

### Step 9：整合測試
- 場景 A：玩家手機滑掉 → 回來 → 看到彈窗 → 點返回 → 恢復遊戲
- 場景 B：主持人關頁面 → 回來 → 帶 hostToken 重連 → 可結束遊戲
- 場景 C：主角斷線 → 本題跳過 → 自動下一題
- 場景 D：遊戲結束 → session 清除 → 可以建新房
- 場景 E：所有人斷線 → 5 分鐘後房間自動清理
- 場景 F：同電腦測試 → Chrome + 無痕模式

---

## 五、問題與解法對照表

| 場景 | 目前行為 | 新架構行為 |
|------|---------|-----------|
| 手機滑掉/關掉 | 自動嘗試重連，成功就回去 | WebSocket 重連 → CHECK_ROOM → 彈窗詢問「返回遊戲？」|
| 主持人關頁面再回來 | REJOIN 但可能無法結束遊戲 | 帶 hostToken REJOIN → 恢復主持人身份 → 可結束遊戲 |
| 主角斷線 | 本題無效 → 選新主角 → 下一題 | 一樣（已實作），確認穩定運作 |
| 想換房間 | 被 localStorage 綁死 | 建房/加入前自動 LEAVE 舊房；彈窗也有「離開」選項 |
| 遊戲結束想重開 | session 衝突 | 遊戲結束時清 session；ROOM_STATUS 回報 finished → 清 session |
| 同電腦測試 | localStorage 衝突 | 用無痕模式/不同瀏覽器（不改代碼） |
| 孤兒房間 | 卡在 Redis 24h | 背景清掃 goroutine 定期清理 |
