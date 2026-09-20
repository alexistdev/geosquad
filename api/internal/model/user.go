// Package model memuat bentuk data domain seperti tersimpan di database.
package model

import "time"

type Role string

const (
	RoleAdmin Role = "ADMIN"
	RoleUser  Role = "USER"
)

func (r Role) Valid() bool {
	return r == RoleAdmin || r == RoleUser
}

func (r Role) IsAdmin() bool { return r == RoleAdmin }

// Audit adalah kolom yang dimiliki semua tabel domain, mengikuti BaseEntity
// yang dipakai project lain. Diisi repository, tidak pernah oleh handler.
type Audit struct {
	CreatedBy    string    `json:"createdBy"`
	ModifiedBy   string    `json:"modifiedBy"`
	CreatedDate  time.Time `json:"createdDate"`
	ModifiedDate time.Time `json:"modifiedDate"`
	IsDeleted    bool      `json:"-"`
}

type User struct {
	ID       string `json:"id"`
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	// Password tidak pernah ikut serialisasi JSON. Tag "-" adalah pertahanan
	// terakhir kalau ada yang tidak sengaja mengembalikan model ini langsung.
	Password    string `json:"-"`
	Role        Role   `json:"role"`
	IsSuspended bool   `json:"isSuspended"`
	Audit
}
