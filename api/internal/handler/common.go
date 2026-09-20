// Package handler menerjemahkan HTTP ke pemanggilan service dan sebaliknya.
// Tidak ada aturan bisnis di sini.
package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/alexistdev/geosquad/api/internal/middleware"
	"github.com/alexistdev/geosquad/api/internal/response"
	"github.com/alexistdev/geosquad/api/internal/service"
)

// maxBodyBytes membatasi ukuran body. Tanpa batas, satu request bisa
// menghabiskan memori server.
const maxBodyBytes = 1 << 20 // 1 MB

// decode membaca JSON body dengan galat yang bisa dimengerti user.
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	// Field asing ditolak, bukan diabaikan. Salah ketik nama field lebih baik
	// terasa sebagai error daripada diam-diam tidak berpengaruh.
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		switch {
		case err == io.EOF:
			response.BadRequest(w, "Body request kosong.")
		default:
			response.BadRequest(w, "Body request bukan JSON yang valid: "+err.Error())
		}
		return false
	}
	return true
}

// actor mengambil identitas pemanggil. Handler yang memakainya selalu berada
// di belakang middleware Authenticate, jadi ketiadaannya adalah salah pasang
// route, bukan kesalahan user.
func actor(w http.ResponseWriter, r *http.Request) (*service.Actor, bool) {
	a, ok := middleware.ActorFrom(r.Context())
	if !ok {
		response.Unauthorized(w, "Belum terautentikasi.")
		return nil, false
	}
	return a, true
}

type pagination struct {
	Page    int
	PerPage int
}

func (p pagination) Offset() int { return (p.Page - 1) * p.PerPage }

// parsePagination membaca page dan perPage, dengan batas atas supaya satu
// request tidak bisa meminta seluruh tabel sekaligus.
func parsePagination(r *http.Request) pagination {
	const (
		defaultPerPage = 20
		maxPerPage     = 100
	)

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	perPage, err := strconv.Atoi(r.URL.Query().Get("perPage"))
	if err != nil || perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	return pagination{Page: page, PerPage: perPage}
}

func meta(p pagination, total int64) response.Meta {
	totalPages := 0
	if p.PerPage > 0 {
		totalPages = int((total + int64(p.PerPage) - 1) / int64(p.PerPage))
	}
	return response.Meta{
		Page:       p.Page,
		PerPage:    p.PerPage,
		Total:      total,
		TotalPages: totalPages,
	}
}
