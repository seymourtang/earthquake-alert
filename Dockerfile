FROM golang:1.21-alpine as builder

WORKDIR /go/src/workspace

COPY . .

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0
ENV GOFLAGS="-mod=vendor"

RUN go build -trimpath -v -ldflags="-s -w" -o ./bin/app ./cmd

FROM debian:bookworm-slim AS final

WORKDIR /workspace

COPY --from=builder /go/src/workspace/bin/app .

ENTRYPOINT ["./app"]