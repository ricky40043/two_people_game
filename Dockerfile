# Stage 1: Build Vue frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend
COPY kahoot-game/frontend/package*.json ./
RUN npm ci --silent
COPY kahoot-game/frontend/ .
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.21-alpine AS backend-builder

WORKDIR /app
COPY kahoot-game/backend/go.mod kahoot-game/backend/go.sum ./
RUN go mod download
COPY kahoot-game/backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/main.go

# Stage 3: Final lightweight image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=backend-builder /app/main .
COPY --from=frontend-builder /frontend/dist ./static

EXPOSE 8080
CMD ["./main"]
