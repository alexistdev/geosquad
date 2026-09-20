# GeoSquad API

Backend Go yang mengorkestrasi run [engine](../engine/): menerima request, menjalankan
`engine/main.py` sebagai proses terpisah, melacak statusnya, dan mengalirkan
log-nya realtime ke klien.

Bagian dari [GeoSquad](../README.md). Dikonsumsi oleh `web/` (Angular).

---

## Prasyarat

| Kebutuhan | Keterangan |
|---|---|
| Go 1.26 | lihat `go.mod` |
| MySQL 8 | skema dibuat otomatis oleh migrasi saat start |
| Redis 7 | menyimpan sesi login |
| Engine siap | `engine/.venv` sudah ada, lihat [tutorial langkah 1](../README.md#1-engine) |

---

## Menjalankan

```bash
cd api
cp .env.example .env
make secret          # salin hasilnya ke JWT_SECRET di .env
make up              # nyalakan MySQL + Redis lewat docker compose
make run             # migrasi jalan otomatis, lalu API melayani di :8084
```

Sudah punya MySQL dan Redis sendiri? Lewati `make up`, cukup isi `DB_*` dan
`REDIS_*` di `.env` lalu buat database kosongnya:

```sql
CREATE DATABASE geosquad CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

Perintah lain: `make help`.

---

## Autentikasi

Polanya sama dengan geolicense: **JWT dibuat saat login tapi tidak pernah
dikirim ke browser.** Yang dikirim adalah `sessionId` acak, dan JWT-nya
disimpan di Redis dengan `sessionId` sebagai kunci.

```
POST /auth/login
      │
      ├─ verifikasi bcrypt
      ├─ terbitkan JWT (HS256)
      ├─ Redis SET geosquad:session:<sessionId> = <JWT>, TTL 24 jam
      └─ balas sessionId + pasang cookie SID (HttpOnly)

request berikutnya
      │
      ├─ baca cookie SID  (atau header X-Session-Id, atau Bearer <JWT>)
      ├─ Redis GET → dapat JWT
      ├─ verifikasi tanda tangan & masa berlaku
      └─ baca ulang peran dari database
```

Dua hal yang membuat ini lebih kuat dari JWT polos:

- **Sesi bisa dicabut seketika.** Hapus satu key Redis, sesi mati. JWT murni
  tetap sah sampai kedaluwarsa. Dipakai saat logout, ganti password, suspend,
  dan hapus user.
- **Peran dibaca ulang dari database tiap request**, tidak dipercaya dari klaim
  token. User yang baru diturunkan dari admin langsung kehilangan haknya.

Cookie `SID` bersifat `HttpOnly` (JavaScript tidak bisa membacanya, jadi XSS
tidak otomatis berarti sesi tercuri) dan `SameSite=Lax`. Di production
`SESSION_COOKIE_SECURE=true` wajib, dan aplikasi menolak start kalau tidak.

---

## Endpoint

Semua balasan berbentuk `{status, messages, payload}`, sama seperti geobill dan
geolicense, supaya interceptor Angular yang sama bisa dipakai ulang.

### Publik

| Method | Path | Keterangan |
|---|---|---|
| `GET` | `/health` | cek MySQL + Redis. Balas 503 kalau ada yang jatuh |
| `POST` | `/api/v1/auth/register` | registrasi mandiri, selalu menghasilkan peran USER |
| `POST` | `/api/v1/auth/login` | balas `sessionId` + pasang cookie `SID` |

### Butuh login

| Method | Path | Keterangan |
|---|---|---|
| `GET` | `/api/v1/auth/me` | profil sendiri |
| `POST` | `/api/v1/auth/logout` | akhiri sesi ini |
| `POST` | `/api/v1/auth/logout-all` | akhiri semua sesi di semua perangkat |
| `POST` | `/api/v1/auth/change-password` | ganti password, semua sesi ikut dicabut |
| `POST` | `/api/v1/runs` | buat run baru, engine langsung dijalankan |
| `GET` | `/api/v1/runs` | daftar run. `?status=` `?page=` `?perPage=` |
| `GET` | `/api/v1/runs/{id}` | detail satu run |
| `GET` | `/api/v1/runs/{id}/logs` | log tersimpan. `?afterSeq=` `?limit=` |
| `GET` | `/api/v1/runs/{id}/stream` | log realtime lewat SSE |
| `POST` | `/api/v1/runs/{id}/cancel` | hentikan run yang sedang jalan |
| `DELETE` | `/api/v1/runs/{id}` | hapus run yang sudah selesai (soft delete) |

User biasa hanya melihat run miliknya sendiri. Filter `userId` yang dikirim user
biasa diabaikan, bukan dipercaya.

### Admin

| Method | Path | Keterangan |
|---|---|---|
| `GET` | `/api/v1/admin/users` | daftar user. `?search=` `?role=` |
| `POST` | `/api/v1/admin/users` | buat user, termasuk menentukan peran |
| `GET` | `/api/v1/admin/users/{id}` | detail user |
| `PATCH` | `/api/v1/admin/users/{id}` | ubah nama, peran, atau status suspend |
| `DELETE` | `/api/v1/admin/users/{id}` | hapus user (soft delete) |

Admin aktif terakhir tidak bisa diturunkan, dinonaktifkan, atau dihapus, dan
admin tidak bisa menonaktifkan atau menghapus akunnya sendiri. Tanpa pagar ini,
satu klik bisa mengunci semua orang keluar dari panel admin.

---

## Memantau run

Buat run, lalu dengarkan lognya:

```bash
curl -c cookies.txt -X POST http://localhost:8084/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@gmail.com","password":"password"}'

curl -b cookies.txt -X POST http://localhost:8084/api/v1/runs \
  -H 'Content-Type: application/json' \
  -d '{"request":"Buat REST API untuk mencatat buku dengan endpoint tambah, list, dan hapus"}'

curl -b cookies.txt -N http://localhost:8084/api/v1/runs/<id>/stream
```

Bentuk event SSE:

```
event: log
data: {"kind":"log","seq":42,"stream":"STDOUT","line":"[git] branch: squad/20260919-203145"}

event: done
data: {"kind":"done","status":"PASSED"}
```

Sambungan putus di tengah? Sambung lagi dengan `?afterSeq=<seq terakhir>`;
riwayat yang terlewat dikirim ulang sebelum aliran live disusul.

### Status run

```
PENDING ──> RUNNING ──┬──> PASSED      engine exit 0, SQA lulus
                      ├──> FAILED      engine exit bukan 0, SQA masih menolak
                      ├──> CANCELLED   dihentikan user
                      └──> ERROR       gagal di luar logika engine
```

`PENDING` berarti run sudah diterima tapi masih antre slot
(`ENGINE_MAX_CONCURRENT`). Run yang tercatat `RUNNING` saat API restart
otomatis dijadikan `ERROR`, karena proses Python-nya ikut mati bersama API.

---

## Konfigurasi

Semua lewat `.env`. Daftar lengkap ada di `.env.example`; yang sering diubah:

| Variabel | Default | Fungsi |
|---|---|---|
| `APP_PORT` | `8084` | port API (geobill 8083, geolicense 8082) |
| `JWT_SECRET` | — | **wajib**, minimal 32 karakter |
| `JWT_EXPIRATION` | `24h` | umur JWT |
| `SESSION_TTL` | `24h` | umur sessionId di Redis |
| `SESSION_COOKIE_SECURE` | `false` | **wajib `true` di production** |
| `ENGINE_DIR` | `../engine` | lokasi `main.py` |
| `ENGINE_PYTHON` | `../engine/.venv/bin/python` | interpreter venv runner engine |
| `ENGINE_MAX_CONCURRENT` | `2` | berapa run boleh jalan bersamaan |
| `ENGINE_TIMEOUT` | `60m` | batas waktu satu run |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:4200` | dipisah koma |

`CORS_ALLOWED_ORIGINS` tidak menerima `*`. API ini memakai cookie sesi, dan
browser menolak kombinasi `Allow-Origin: *` dengan `Allow-Credentials: true`.

---

## Migrasi

Ditulis sebagai file SQL bernomor di `migrations/`, dijalankan
[goose](https://github.com/pressly/goose). File-nya **ditanam ke dalam binary**
lewat `embed.FS`, jadi deploy cukup mengirim satu binary — tidak ada folder
migrations yang bisa tertinggal beda versi di server.

```bash
make migrate         # jalankan yang belum pernah jalan, lalu keluar
make migrate-status  # lihat versi skema saat ini
make migrate-down    # mundur satu versi
```

Saat `make run`, migrasi jalan otomatis sebelum server melayani request.

Menambah migrasi: buat `migrations/0000N_nama.sql` dengan penanda goose.

```sql
-- +goose Up
ALTER TABLE runs ADD COLUMN catatan TEXT NULL;

-- +goose Down
ALTER TABLE runs DROP COLUMN catatan;
```

### Skema

```
users ──< runs ──< run_logs
```

Semua tabel domain punya kolom audit (`created_by`, `modified_by`,
`created_date`, `modified_date`) dan `is_deleted`, mengikuti `BaseEntity` di
project lain. **Hapus selalu soft delete** — setiap query menyaring
`is_deleted = 0`.

Unique index email sengaja mencakup `is_deleted`. Tanpa itu, email bekas user
terhapus akan terkunci selamanya dan orang yang sama tidak bisa mendaftar lagi.

### Akun bawaan

Migrasi `00004` menanam satu admin supaya API bisa langsung dipakai, dan
migrasi `00006` mengubah emailnya:

```
admin@gmail.com / password
```

**Ganti password ini sebelum dipakai di luar mesin sendiri.**

---

## Struktur

```
api/
├── cmd/api/main.go        # rakit dependensi, jalankan server
├── migrations/            # file .sql + embed.go
└── internal/
    ├── config/            # baca & validasi environment
    ├── db/                # koneksi MySQL, runner migrasi
    ├── cache/             # koneksi Redis
    ├── model/             # bentuk data domain
    ├── dto/               # bentuk request/response + validasi
    ├── response/          # {status, messages, payload} + peta error → HTTP
    ├── auth/              # JWT, session store, bcrypt
    ├── middleware/        # auth, peran, CORS, logging, recover
    ├── repository/        # semua SQL ada di sini, tidak di tempat lain
    ├── service/           # aturan bisnis + hub SSE
    ├── handler/           # HTTP masuk/keluar + router
    └── runner/            # jalankan engine sebagai subprocess
```

Arahnya satu jalur: `handler → service → repository`. Service tidak mengimpor
`net/http`; ia melempar error domain, dan handler yang menerjemahkannya jadi
status HTTP. Semua SQL terkurung di `repository`.

---

## Catatan desain

### Request user tidak pernah lewat shell

Engine dijalankan sebagai `exec.Command(python, "main.py", request)` — teks dari
user masuk sebagai satu argumen utuh, bukan dirangkai jadi perintah shell.
Tanpa shell, tidak ada karakter yang punya arti khusus dan tidak ada yang perlu
di-escape.

### Log ditulis batch

Engine yang verbose bisa mengeluarkan ratusan baris per detik. Satu `INSERT` per
baris akan membuat database jadi penghambat proses yang seharusnya cuma diamati,
jadi log dikumpulkan dulu (50 baris atau 500 ms, mana yang lebih dulu). Siaran
ke penonton SSE tidak menunggu itu — mereka dapat barisnya seketika.

### Hub SSE hanya dalam proses

Siaran log tidak lewat Redis pub/sub, karena runner-nya pun hanya hidup di satu
proses: run yang dijalankan instance A tidak bisa dipantau lewat instance B —
proses Python-nya ada di A. Menambah Redis di sini akan memberi kesan skala
horizontal yang sebenarnya belum ada. Kalau nanti API di-scale, runner dan hub
harus dipindah bersama-sama.

### Run milik orang lain dibalas 404, bukan 403

Membalas 403 sama saja mengonfirmasi bahwa run dengan id itu memang ada, cuma
milik orang lain. Pesan login yang gagal juga tidak membedakan "email tidak
terdaftar" dari "password salah", dengan alasan yang sama.

---

## Pengembangan

```bash
make check   # gofmt + vet + test -race
make test    # test saja
make build   # binary statis ke bin/api (CGO_ENABLED=0)
```

`make build` menghasilkan satu binary tanpa ketergantungan C, karena driver
MySQL-nya Go murni. Migrasi ikut di dalamnya.
