FROM registry.cn-shanghai.aliyuncs.com/txm-graduation/golang:1.20-alpine as builder

WORKDIR /go/src/workspace

COPY . .

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0
ENV GOFLAGS="-mod=vendor"

RUN go build -trimpath -v -ldflags="-s -w" -o ./bin/app ./cmd

FROM registry.cn-shanghai.aliyuncs.com/txm-graduation/debian:bookworm-slim AS final

WORKDIR /workspace

COPY --from=builder /go/src/workspace/bin/app .

ENTRYPOINT ["./app"]