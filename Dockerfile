FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /wallet ./cmd/main.go

FROM scratch
COPY --from=build /wallet /wallet
EXPOSE 8080
ENTRYPOINT ["/wallet"]