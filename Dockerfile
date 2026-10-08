FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/api-react ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Jakarta
COPY --from=build /out/api-react /usr/local/bin/api-react
EXPOSE 8080
USER nobody
ENTRYPOINT ["/usr/local/bin/api-react"]
