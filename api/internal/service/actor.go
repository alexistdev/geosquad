package service

import "github.com/alexistdev/geosquad/api/internal/model"

// Actor adalah identitas pemanggil yang sudah terautentikasi. Dibawa dari
// middleware ke service lewat context, supaya service bisa memutuskan hak
// akses tanpa menyentuh request HTTP.
type Actor struct {
	UserID    string
	Email     string
	Role      model.Role
	SessionID string
}

func (a *Actor) IsAdmin() bool { return a != nil && a.Role.IsAdmin() }

// CanAccess menjawab apakah actor boleh menyentuh data milik ownerID.
// Admin boleh apa saja; user biasa hanya miliknya sendiri.
func (a *Actor) CanAccess(ownerID string) bool {
	if a == nil {
		return false
	}
	return a.IsAdmin() || a.UserID == ownerID
}
