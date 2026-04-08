package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"
)

type Event struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

type Observer interface {
	Notify(context.Context, Event) error
}

type Broadcaster struct {
	mu        sync.RWMutex
	observers []Observer
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		observers: make([]Observer, 0),
	}
}

func (b *Broadcaster) Subscribe(o Observer) {
	if o == nil {
		return
	}
	b.mu.Lock()
	b.observers = append(b.observers, o)
	b.mu.Unlock()
}

func (b *Broadcaster) Publish(ctx context.Context, event Event) {
	b.mu.RLock()
	observers := make([]Observer, len(b.observers))
	copy(observers, b.observers)
	b.mu.RUnlock()

	for _, o := range observers {
		_ = o.Notify(ctx, event)
	}
}

type FileObserver struct {
	path string
	mu   sync.Mutex
}

func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

func (o *FileObserver) Notify(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	f, err := os.OpenFile(o.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append(data, '\n'))
	return err
}

type HTTPObserver struct {
	url    string
	client *http.Client
}

func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (o *HTTPObserver) Notify(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
