FROM golang:1.20.7-alpine

WORKDIR /usr/src/omnivore-raindrop-sync

COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .

RUN go build -v -o omnivore-raindrop-sync

FROM alpine

WORKDIR /usr/src/omnivore-raindrop-sync

COPY --from=0 /usr/src/omnivore-raindrop-sync/omnivore-raindrop-sync /usr/src/omnivore-raindrop-sync/.env ./

CMD ["./omnivore-raindrop-sync"]
