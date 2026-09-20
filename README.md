# GeoSquad

Squad AI yang membangun aplikasi dari satu kalimat request.

```
web (Angular :4200)  ──>  api (Go :8084)  ──>  engine (Python)  ──>  LLM
   tulis request           auth, MySQL,         Tech Lead, Dev,
   tonton log SSE          Redis, orkestrasi    SQA, DevOps
```

Halaman ini **tutorial menjalankan aplikasinya dari nol**, berurutan:
[engine](#1-engine) → [api](#2-api) → [web](#3-web). Ikuti dari atas ke bawah;
tiap bagian bergantung pada bagian sebelumnya.

Dokumentasi acuan tiap komponen ada di README masing-masing:
[engine/](engine/README.md) · [api/](api/README.md) · [web/](web/README.md).

---

## Prasyarat

Pasang dulu semuanya sebelum mulai:

| Kebutuhan | Dipakai oleh | Keterangan |
|---|---|---|
| Python 3.12 | engine | lihat `engine/.python-version`. Versi lain belum diuji |
| Go 1.26 | api | lihat `api/go.mod` |
| Node 20+ dan npm | web | |
| Docker | api | untuk menyalakan MySQL + Redis. Punya sendiri? lihat [catatan](#sudah-punya-mysql-dan-redis-sendiri) |
| Endpoint LLM | engine | 9router/OpenRouter, atau apa pun yang kompatibel OpenAI |
| [`uv`](https://docs.astral.sh/uv/) | engine | opsional, mempercepat pembuatan venv |

Ambil kodenya:

```bash
git clone git@github.com:alexistdev/geosquad.git
cd geosquad
```

---

## 1. Engine

Engine adalah yang benar-benar membangun aplikasi. API menjalankannya sebagai
proses terpisah, jadi **venv dan `.env`-nya harus siap sebelum API dinyalakan.**

### 1.1 Buat venv dan pasang dependency

```bash
cd engine
uv venv .venv --python 3.12
source .venv/bin/activate
uv pip install -r requirements.txt
```

Tanpa `uv`:

```bash
cd engine
python3.12 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

> Engine memakai **dua venv terpisah**: satu untuk dirinya sendiri (`.venv`),
> satu lagi dibuat otomatis per tiket untuk menjalankan test kode hasil squad.
> Jangan digabung — alasannya di [engine/README.md](engine/README.md#kenapa-dua-venv).

### 1.2 Isi konfigurasi

```bash
cp .env.example .env
```

Minimal yang **wajib** diisi supaya engine mau start:

```ini
NINE_ROUTER_API_KEY=<api key kamu>
NINE_ROUTER_BASE_URL=http://localhost:20128/v1
NINE_ROUTER_MODEL=openai/openrouter/minimax/minimax-m3:free
```

Sisanya boleh dibiarkan kosong; defaultnya sudah aman. Daftar lengkapnya ada di
[engine/README.md](engine/README.md#tuning).

`engine/.env` sudah masuk `.gitignore`. Jangan pernah commit file itu.

### 1.3 Pastikan engine jalan

Uji sekali lewat CLI sebelum lanjut, supaya kalau ada yang salah ketahuan di
sini dan bukan nanti saat ditembak dari API:

```bash
python main.py "Buat fungsi Python yang menghitung faktorial, lengkap dengan test"
```

Kalau berhasil, hasilnya muncul di `engine/output/local-<waktu>/` dan kodenya di
`engine/workspace/local-<waktu>/`. Exit code `0` berarti sukses.

Gagal di langkah ini? Lihat
[troubleshooting engine](engine/README.md#troubleshooting).

Setelah selesai, venv boleh dinonaktifkan — API memanggil interpreter-nya
langsung lewat path:

```bash
deactivate
cd ..
```

---

## 2. API

Backend Go yang membungkus engine jadi layanan HTTP: menjalankan engine,
melacak statusnya, mengalirkan log realtime, dan mengurus login.

### 2.1 Siapkan konfigurasi

```bash
cd api
cp .env.example .env
```

Buat `JWT_SECRET` acak, lalu salin hasilnya ke `.env`:

```bash
make secret
```

```ini
JWT_SECRET=<tempel hasil make secret di sini>
```

API **menolak start** kalau `JWT_SECRET` kurang dari 32 karakter.

Periksa juga dua baris ini cocok dengan lokasi engine tadi — defaultnya sudah
benar kalau kamu mengikuti langkah 1:

```ini
ENGINE_DIR=../engine
ENGINE_PYTHON=../engine/.venv/bin/python
```

Terakhir, **isi `DB_PASSWORD`** — jangan dibiarkan kosong seperti di
`.env.example`:

```ini
DB_PASSWORD=geosquad
```

Alasannya: `docker-compose.yml` memakai `${DB_PASSWORD:-geosquad}`, dan nilai
kosong ikut memicu default itu. Kalau dibiarkan kosong, password root container
jadi `geosquad` sementara API menyambung dengan password kosong — koneksi
ditolak. Isi sama di kedua sisi dan masalahnya hilang.

### 2.2 Nyalakan MySQL dan Redis

```bash
make up
```

Ini menjalankan MySQL 8.4 di `:3306` dan Redis 7 di `:6379` lewat Docker.
Tunggu beberapa detik saat pertama kali — MySQL butuh waktu menyiapkan datadir.

> `MYSQL_ROOT_PASSWORD` hanya dibaca **saat volume dibuat pertama kali.**
> Kalau kamu terlanjur `make up` dengan `DB_PASSWORD` kosong, mengubah `.env`
> saja tidak cukup — buang dulu volumenya:
>
> ```bash
> docker compose down -v
> ```

#### Sudah punya MySQL dan Redis sendiri?

Lewati `make up`. Isi `DB_*` dan `REDIS_*` di `.env`, lalu buat database
kosongnya:

```sql
CREATE DATABASE geosquad CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 2.3 Jalankan API

```bash
make run
```

Migrasi database jalan otomatis sebelum server melayani request — tabel dan
akun admin bawaan dibuat di sini, tidak ada langkah manual.

API melayani di `http://localhost:8084`. Biarkan terminal ini terbuka.

### 2.4 Pastikan API hidup

Di terminal lain:

```bash
curl http://localhost:8084/health
```

Balasan `200` berarti MySQL dan Redis keduanya tersambung. Kalau `503`, salah
satunya belum jalan — cek `make up` sudah selesai.

---

## 3. Web

Frontend Angular: tempat menulis request dan menonton lognya mengalir.

### 3.1 Pasang dependency dan jalankan

Di terminal baru, dari akar repo:

```bash
cd web
npm install
npm start
```

Dev server melayani di `http://localhost:4200`.

`npm start` sudah memakai `proxy.conf.json`, yang meneruskan `/api/*` ke
`localhost:8084`. Browser melihat semuanya satu origin, sehingga cookie sesi
terkirim tanpa urusan CORS.

### 3.2 Masuk

Buka `http://localhost:4200`, lalu masuk dengan akun bawaan:

```
admin@gmail.com / password
```

**Ganti password itu sebelum dipakai di luar mesin sendiri.** Repo ini publik,
jadi kombinasi di atas diketahui siapa pun.

---

## Menjalankan squad pertama

1. Buka dashboard, tulis request di formnya, misalnya
   *"Buat REST API untuk mencatat buku dengan endpoint tambah, list, dan hapus"*.
2. Kirim. Run masuk dengan status `PENDING`, lalu `RUNNING` begitu dapat slot.
3. Klik run-nya untuk menonton log mengalir realtime.
4. Selesai saat status jadi `PASSED` (SQA lulus) atau `FAILED`.

Hasilnya ada di mesin kamu, per nomor tiket:

```bash
ls engine/output/geo-1001/       # laporan tiap tahap
cd engine/workspace/geo-1001     # kode yang dibangun squad
git log --oneline
```

---

## Ringkasan: menjalankan ulang besok

Setup di atas cukup sekali. Berikutnya tinggal tiga perintah:

```bash
cd api && make up && make run    # terminal 1
cd web && npm start              # terminal 2
```

Engine tidak perlu dijalankan sendiri — API yang memanggilnya.

Mematikan MySQL dan Redis saat selesai:

```bash
cd api && make down
```

---

## Kalau ada yang tidak beres

| Gejala | Kemungkinan penyebab |
|---|---|
| API berhenti dengan `JWT_SECRET wajib diisi minimal 32 karakter` | `make secret` belum disalin ke `api/.env` |
| `/health` balas `503` | MySQL atau Redis belum jalan. Jalankan `make up`, tunggu sampai selesai |
| `Access denied for user 'root'` | `DB_PASSWORD` di `api/.env` beda dengan yang dipakai container. Lihat [langkah 2.1](#21-siapkan-konfigurasi) |
| Login ditolak terus | Migrasi belum jalan. Cek `make migrate-status` di `api/` |
| Run langsung jadi `ERROR` | Path engine salah, atau `engine/.venv` belum dibuat. Cek `ENGINE_PYTHON` di `api/.env` |
| Run jadi `FAILED` terus | Masalahnya di engine, bukan setup. Lihat [troubleshooting engine](engine/README.md#troubleshooting) |
| Halaman web kosong / 404 saat refresh | Pastikan `npm start` yang dipakai, bukan membuka `dist/` langsung |

Penjelasan lebih dalam per komponen:
[engine/README.md](engine/README.md) · [api/README.md](api/README.md) ·
[web/README.md](web/README.md).
