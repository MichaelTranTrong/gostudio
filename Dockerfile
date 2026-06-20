FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o gostudio .

FROM alpine:3.20
RUN apk add --no-cache ffmpeg ca-certificates tzdata
ENV TZ=Asia/Ho_Chi_Minh
WORKDIR /app
COPY --from=builder /app/gostudio .
COPY web/ ./web/
RUN mkdir -p uploads outputs
EXPOSE 2005
CMD ["./gostudio"]
