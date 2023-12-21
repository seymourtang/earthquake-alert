package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type Response struct {
	Code    int     `json:"code"`
	Message string  `json:"message"`
	Data    []Event `json:"data"`
}

type Event struct {
	EventId   int     `json:"eventId"`
	Updates   int     `json:"updates"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Depth     float64 `json:"depth"`
	Epicenter string  `json:"epicenter"`
	StartAt   int64   `json:"startAt"`
	UpdateAt  int64   `json:"updateAt"`
	Magnitude float64 `json:"magnitude"`
	InsideNet int     `json:"insideNet"`
	Sations   int     `json:"sations"`
}

func query[T any](ctx context.Context, lastTs int64) (*T, error) {
	url := fmt.Sprintf("https://mobile-new.chinaeew.cn/v1/earlywarnings?start_at=%d&updates=4", lastTs)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var resp T
	if err = json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func loop(ctx context.Context, notification chan<- Event) {
	ticker := time.NewTicker(*duration)
	defer func() {
		ticker.Stop()
	}()

	var (
		lastTs int64 = 0
	)

	for {
		select {
		case <-ticker.C:
			resp, err := query[Response](ctx, lastTs)
			if err != nil {
				slog.Error("query data", "err", err)
			} else {
				if resp != nil && len(resp.Data) > 0 {
					slog.Info("found the events", "num", len(resp.Data), "events", resp.Data)
					lastTs = resp.Data[0].StartAt
					tt := time.UnixMilli(lastTs)
					if time.Since(tt) <= 30*time.Minute {
						notification <- resp.Data[0]
					} else {
						slog.Info("the latest event is out of date", "startAt", tt.String(), "event", resp.Data[0])
					}
				}
			}
		case <-ctx.Done():
			slog.Info("loop exiting")
			return
		}
		ticker.Reset(*duration)
	}
}

func notification(ctx context.Context, ch <-chan Event) {
	fn := func(event Event) error {
		tz, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			return err
		}
		url := fmt.Sprintf("https://api.day.app/%s/%s/%s", *key,
			fmt.Sprintf("%s 有%.1f级地震发生了", time.UnixMilli(event.StartAt).In(tz).Format(time.DateTime), event.Magnitude),
			fmt.Sprintf("地点:%s,东经:%f°,北纬:%f°,地震深度:%.1f公里", event.Epicenter, event.Longitude, event.Latitude, event.Depth))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		response, err := client.Do(req)
		if err != nil {
			return err
		}
		defer func() {
			_ = response.Body.Close()
		}()

		data, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		slog.Info("notification successfully", "result", string(data))
		return nil

	}
	defer func() {
		slog.Info("notification exiting...")
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-ch:
			if err := fn(event); err != nil {
				slog.Error("send notification failed", "err", err)
			}
		}
	}
}

var (
	key      = flag.String("key", "", "the key of bar app")
	duration = flag.Duration("duration", 3*time.Second, "the interval of query data")
)

var client = http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
	Timeout: 10 * time.Second,
}

func main() {
	flag.Parse()
	if *key == "" {
		panic("key should have a value")
	}

	ctx, cancelFunc := context.WithCancel(context.TODO())
	ch := make(chan Event)

	go notification(ctx, ch)
	go loop(ctx, ch)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	cancelFunc()

	slog.Info("exiting...")
}
