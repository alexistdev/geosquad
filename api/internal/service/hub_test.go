package service

import (
	"testing"
	"time"

	"github.com/alexistdev/geosquad/api/internal/model"
)

func TestHubDeliversToSubscriber(t *testing.T) {
	hub := NewHub()
	events, unsubscribe := hub.Subscribe("run-1")
	defer unsubscribe()

	hub.Publish("run-1", model.RunLog{Seq: 1, Stream: model.StreamStdout, Line: "halo"})

	select {
	case ev := <-events:
		if ev.Line != "halo" || ev.Seq != 1 {
			t.Errorf("event = %+v, mau baris 'halo' seq 1", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("event tidak sampai")
	}
}

func TestHubIsolatesRuns(t *testing.T) {
	hub := NewHub()
	events, unsubscribe := hub.Subscribe("run-1")
	defer unsubscribe()

	hub.Publish("run-2", model.RunLog{Seq: 1, Line: "punya run lain"})

	select {
	case ev := <-events:
		t.Fatalf("menerima event run lain: %+v", ev)
	case <-time.After(100 * time.Millisecond):
		// benar: tidak ada yang datang
	}
}

// Penonton yang lambat tidak boleh memblokir runner. Begitu buffer penuh,
// pesan dibuang dan Publish tetap kembali seketika.
func TestHubDoesNotBlockOnSlowSubscriber(t *testing.T) {
	hub := NewHub()
	_, unsubscribe := hub.Subscribe("run-1")
	defer unsubscribe()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 5000; i++ {
			hub.Publish("run-1", model.RunLog{Seq: i, Line: "banjir"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Publish terblokir oleh penonton yang tidak membaca")
	}
}

func TestHubUnsubscribeIsIdempotent(t *testing.T) {
	hub := NewHub()
	_, unsubscribe := hub.Subscribe("run-1")

	unsubscribe()
	// Panggilan kedua tidak boleh panic karena close channel dua kali.
	unsubscribe()

	// Publish setelah semua penonton pergi juga harus aman.
	hub.Publish("run-1", model.RunLog{Seq: 1, Line: "tidak ada yang dengar"})
}

func TestHubCloseSendsDone(t *testing.T) {
	hub := NewHub()
	events, unsubscribe := hub.Subscribe("run-1")
	defer unsubscribe()

	hub.Close("run-1", model.RunPassed)

	select {
	case ev := <-events:
		if ev.Kind != EventDone || ev.Status != model.RunPassed {
			t.Errorf("event = %+v, mau done/PASSED", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("event penutup tidak sampai")
	}
}
