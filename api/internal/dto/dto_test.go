package dto

import (
	"strings"
	"testing"
)

func TestRegisterValidate(t *testing.T) {
	cases := []struct {
		name    string
		req     RegisterRequest
		wantErr bool
	}{
		{"valid", RegisterRequest{"Budi Santoso", "budi@example.com", "rahasia123"}, false},
		{"nama kosong", RegisterRequest{"", "budi@example.com", "rahasia123"}, true},
		{"email tanpa @", RegisterRequest{"Budi", "budi.example.com", "rahasia123"}, true},
		{"email tanpa domain", RegisterRequest{"Budi", "budi@example", "rahasia123"}, true},
		{"password pendek", RegisterRequest{"Budi", "budi@example.com", "abc"}, true},
		{"password > 72 byte", RegisterRequest{"Budi", "budi@example.com", strings.Repeat("a", 73)}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.req.Normalize()
			errs := c.req.Validate()
			if got := len(errs) > 0; got != c.wantErr {
				t.Errorf("error = %v (%v), mau error = %v", got, errs, c.wantErr)
			}
		})
	}
}

func TestRegisterNormalizeLowercasesEmail(t *testing.T) {
	req := RegisterRequest{FullName: "  Budi  ", Email: "  BUDI@Example.COM  "}
	req.Normalize()

	if req.Email != "budi@example.com" {
		t.Errorf("email = %q, mau %q", req.Email, "budi@example.com")
	}
	if req.FullName != "Budi" {
		t.Errorf("nama = %q, mau %q", req.FullName, "Budi")
	}
}

// Password tepat 72 byte harus lolos; 73 tidak. Batasnya ada karena bcrypt
// mengabaikan byte setelah ke-72, yang akan membuat dua password berbeda
// diterima sebagai sama.
func TestPasswordBoundary(t *testing.T) {
	if errs := validatePassword(strings.Repeat("a", 72)); len(errs) > 0 {
		t.Errorf("72 byte ditolak: %v", errs)
	}
	if errs := validatePassword(strings.Repeat("a", 73)); len(errs) == 0 {
		t.Error("73 byte diterima")
	}
}

func TestCreateRunValidate(t *testing.T) {
	cases := []struct {
		name    string
		request string
		wantErr bool
	}{
		{"valid", "Buat REST API untuk mencatat buku", false},
		{"kosong", "   ", true},
		{"terlalu pendek", "buat api", true},
		{"terlalu panjang", strings.Repeat("a", 5001), true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := CreateRunRequest{Request: c.request}
			req.Normalize()
			if got := len(req.Validate()) > 0; got != c.wantErr {
				t.Errorf("error = %v, mau %v", got, c.wantErr)
			}
		})
	}
}

// UpdateUserRequest memakai pointer supaya "tidak dikirim" bisa dibedakan dari
// "dikirim kosong". Tanpa itu, PATCH tanpa fullName akan mengosongkan nama.
func TestUpdateUserDistinguishesOmitted(t *testing.T) {
	empty := UpdateUserRequest{}
	if errs := empty.Validate(); len(errs) == 0 {
		t.Error("request tanpa field apa pun seharusnya ditolak")
	}

	name := "Budi Baru"
	only := UpdateUserRequest{FullName: &name}
	only.Normalize()
	if errs := only.Validate(); len(errs) > 0 {
		t.Errorf("update hanya nama ditolak: %v", errs)
	}
	if only.Role != nil {
		t.Error("role ikut terisi padahal tidak dikirim")
	}
}

func TestUpdateUserRejectsUnknownRole(t *testing.T) {
	role := "SUPERADMIN"
	req := UpdateUserRequest{Role: &role}
	req.Normalize()
	if errs := req.Validate(); len(errs) == 0 {
		t.Error("role tidak dikenal diterima")
	}
}
