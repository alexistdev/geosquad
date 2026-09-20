package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexistdev/geosquad/api/internal/dto"
	"github.com/alexistdev/geosquad/api/internal/repository"
	"github.com/alexistdev/geosquad/api/internal/response"
	"github.com/alexistdev/geosquad/api/internal/service"
)

type RunHandler struct {
	svc *service.RunService
	hub *service.Hub
}

func NewRunHandler(svc *service.RunService, hub *service.Hub) *RunHandler {
	return &RunHandler{svc: svc, hub: hub}
}

func (h *RunHandler) Create(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(w, r)
	if !ok {
		return
	}

	var req dto.CreateRunRequest
	if !decode(w, r, &req) {
		return
	}
	req.Normalize()
	if errs := req.Validate(); len(errs) > 0 {
		response.BadRequest(w, errs...)
		return
	}

	run, err := h.svc.Create(r.Context(), req, a)
	if err != nil {
		response.FromError(w, err)
		return
	}
	// 202, bukan 201: engine baru saja mulai, hasilnya belum ada. Client
	// harus memantau lewat GET run atau stream SSE.
	w.Header().Set("Location", "/api/v1/runs/"+run.ID)
	response.Created(w, "Run diterima dan sedang diproses.", dto.NewRunResponse(run))
}

func (h *RunHandler) List(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(w, r)
	if !ok {
		return
	}

	p := parsePagination(r)
	filter := repository.RunFilter{
		UserID: strings.TrimSpace(r.URL.Query().Get("userId")),
		Status: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status"))),
		Limit:  p.PerPage,
		Offset: p.Offset(),
	}

	runs, total, err := h.svc.List(r.Context(), filter, a)
	if err != nil {
		response.FromError(w, err)
		return
	}
	response.OK(w, "", response.Paged{Items: dto.NewRunResponses(runs), Meta: meta(p, total)})
}

func (h *RunHandler) Get(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(w, r)
	if !ok {
		return
	}
	run, err := h.svc.Get(r.Context(), r.PathValue("id"), a)
	if err != nil {
		response.FromError(w, err)
		return
	}
	response.OK(w, "", dto.NewRunResponse(run))
}

func (h *RunHandler) Logs(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(w, r)
	if !ok {
		return
	}

	afterSeq, _ := strconv.Atoi(r.URL.Query().Get("afterSeq"))
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 || limit > 2000 {
		limit = 500
	}

	logs, err := h.svc.Logs(r.Context(), r.PathValue("id"), afterSeq, limit, a)
	if err != nil {
		response.FromError(w, err)
		return
	}
	response.OK(w, "", dto.NewRunLogResponses(logs))
}

func (h *RunHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(w, r)
	if !ok {
		return
	}
	if err := h.svc.Cancel(r.Context(), r.PathValue("id"), a); err != nil {
		response.FromError(w, err)
		return
	}
	response.NoContent(w, "Run sedang dihentikan.")
}

func (h *RunHandler) Delete(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), r.PathValue("id"), a); err != nil {
		response.FromError(w, err)
		return
	}
	response.NoContent(w, "Run berhasil dihapus.")
}

// Stream mengalirkan log run secara realtime lewat Server-Sent Events.
//
// SSE dipilih daripada WebSocket karena alirannya satu arah: server bicara,
// client mendengar. SSE juga berjalan di atas HTTP biasa, jadi ikut cookie
// sesi dan proxy yang sudah ada tanpa perlakuan khusus.
func (h *RunHandler) Stream(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(w, r)
	if !ok {
		return
	}

	runID := r.PathValue("id")
	// Hak akses diperiksa sebelum satu byte pun dikirim. Setelah header SSE
	// terkirim, status code tidak bisa diubah lagi.
	run, err := h.svc.Get(r.Context(), runID, a)
	if err != nil {
		response.FromError(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		response.Internal(w, fmt.Errorf("response writer tidak mendukung streaming"))
		return
	}

	// Berlangganan dulu, baru kirim riwayat. Urutan ini mencegah baris yang
	// muncul di sela keduanya hilang.
	events, unsubscribe := h.hub.Subscribe(runID)
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Memberi tahu nginx supaya tidak membufer response ini.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	afterSeq, _ := strconv.Atoi(r.URL.Query().Get("afterSeq"))
	history, err := h.svc.Logs(r.Context(), runID, afterSeq, 2000, a)
	if err == nil {
		for _, l := range history {
			writeEvent(w, service.Event{
				Kind:   service.EventLog,
				Seq:    l.Seq,
				Stream: string(l.Stream),
				Line:   l.Line,
			})
			afterSeq = l.Seq
		}
		flusher.Flush()
	}

	// Run yang sudah selesai tidak perlu ditunggui. Kirim penutup lalu keluar.
	if run.Status.Terminal() {
		writeEvent(w, service.Event{Kind: service.EventDone, Status: run.Status})
		flusher.Flush()
		return
	}

	// Ping berkala menjaga koneksi tetap hidup melewati proxy yang memutus
	// koneksi diam.
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-r.Context().Done():
			// Client menutup tab atau koneksi putus.
			return

		case ev, open := <-events:
			if !open {
				return
			}
			// Baris yang sudah terkirim lewat riwayat dilewati, supaya tidak
			// tampil dobel di layar.
			if ev.Kind == service.EventLog && ev.Seq <= afterSeq {
				continue
			}
			if ev.Kind == service.EventLog {
				afterSeq = ev.Seq
			}
			writeEvent(w, ev)
			flusher.Flush()
			if ev.Kind == service.EventDone {
				return
			}

		case <-ping.C:
			// Komentar SSE: diabaikan client, cukup untuk menahan timeout.
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func writeEvent(w http.ResponseWriter, ev service.Event) {
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Kind, payload)
}
