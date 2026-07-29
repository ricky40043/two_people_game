# Stage 1: Build Vue frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend
COPY frontend/package*.json ./
RUN npm ci --silent
COPY frontend/ .
# 生產環境不設定 VITE_API_URL / VITE_WS_URL
# → api.ts 用 window.location.origin/api（同源）
# → socket.ts 用 wss://window.location.host/ws（同源）
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.21-alpine AS backend-builder

WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .

# 上版關卡：測試沒過就不給 build。
# 其中 internal/websocket 是整場遊戲的整合測試（1 房 + 5 玩家 + 5 題，
# 過程含隨機斷線重連），必須跑到最終分數全部結算出來才算通過，約 3~5 秒。
# 全部測試都在記憶體模式下跑，不需要 Redis / PostgreSQL / 對外網路。
RUN CGO_ENABLED=0 go test -timeout 5m ./...

RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/main.go

# Stage 3: Final lightweight image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Go binary
COPY --from=backend-builder /app/main .

# Vue dist → served by Go at /static (and NoRoute → index.html)
COPY --from=frontend-builder /frontend/dist ./static

EXPOSE 8080
CMD ["./main"]
