FROM golang:1.26.5-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/server ./cmd/server
RUN go build -o /out/migrate ./cmd/migrate

FROM alpine:3.22

WORKDIR /app

COPY --from=build /out/server /app/server
COPY --from=build /out/migrate /app/migrate
COPY migrations /app/migrations

EXPOSE 8080

CMD ["/app/server"]
