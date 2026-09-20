package service

import (
	"sync"

	"github.com/alexistdev/geosquad/api/internal/model"
)

// Hub menyiarkan log run ke klien SSE yang sedang menonton.
//
// Siaran ini sengaja hanya dalam proses, tidak lewat Redis pub/sub. Runner
// juga hanya hidup di satu proses: run yang dijalankan instance A tidak bisa
// dipantau lewat instance B, karena proses Python-nya pun ada di A. Menambah
// Redis di sini akan memberi kesan skala horizontal yang sebenarnya belum ada.
// Kalau nanti API di-scale, runner dan hub harus dipindah bersama-sama.
type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan Event]struct{}
}

type EventKind string

const (
	EventLog  EventKind = "log"
	EventDone EventKind = "done"
)

type Event struct {
	Kind   EventKind       `json:"kind"`
	Seq    int             `json:"seq,omitempty"`
	Stream string          `json:"stream,omitempty"`
	Line   string          `json:"line,omitempty"`
	Status model.RunStatus `json:"status,omitempty"`
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[string]map[chan Event]struct{})}
}

// Subscribe mendaftarkan penonton baru. Fungsi yang dikembalikan wajib
// dipanggil saat selesai, kalau tidak channel-nya bocor.
func (h *Hub) Subscribe(runID string) (<-chan Event, func()) {
	// Buffer memberi ruang saat klien sedang lambat. Kalau buffer penuh,
	// pesan dibuang, bukan memblokir runner.
	ch := make(chan Event, 256)

	h.mu.Lock()
	if h.subscribers[runID] == nil {
		h.subscribers[runID] = make(map[chan Event]struct{})
	}
	h.subscribers[runID][ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			if subs, ok := h.subscribers[runID]; ok {
				delete(subs, ch)
				if len(subs) == 0 {
					delete(h.subscribers, runID)
				}
			}
			h.mu.Unlock()
			close(ch)
		})
	}
	return ch, unsubscribe
}

func (h *Hub) Publish(runID string, entry model.RunLog) {
	h.broadcast(runID, Event{
		Kind:   EventLog,
		Seq:    entry.Seq,
		Stream: string(entry.Stream),
		Line:   entry.Line,
	})
}

func (h *Hub) Close(runID string, status model.RunStatus) {
	h.broadcast(runID, Event{Kind: EventDone, Status: status})
}

func (h *Hub) broadcast(runID string, ev Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.subscribers[runID] {
		select {
		case ch <- ev:
		default:
			// Klien tidak mengejar. Barisnya dilewati; ia tetap bisa
			// mengambil yang terlewat lewat endpoint log biasa dengan
			// parameter afterSeq.
		}
	}
}
