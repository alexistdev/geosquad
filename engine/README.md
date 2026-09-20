# GeoSquad Engine

Runner CrewAI yang membangun aplikasi dari satu kalimat request. Bagian dari
[GeoSquad](../README.md); dipanggil oleh [`api/`](../api/README.md) sebagai
proses terpisah, tapi bisa juga dijalankan sendiri lewat CLI.

Cara menyiapkan dan menjalankan seluruh aplikasi ada di
[tutorial di README root](../README.md). Halaman ini dokumentasi acuan engine:
opsi konfigurasi, batas keamanan, dan alasan di balik desainnya.

---

## Cara kerja

Satu squad beranggotakan empat peran, dijalankan berurutan:

```
Tech Lead  ->  Developer  ->  SQA  ->  DevOps
  kontrak       menulis       menguji    deploy
                  kode      (gerbang)   (opsional)
```

Gerbang antar peran ditulis sebagai Python biasa di `main.py`, bukan fitur
CrewAI. Konsekuensinya: **DevOps tidak akan pernah jalan kalau verdict SQA
bukan `PASS`**, dan itu bisa dibaca langsung dari kode.

Yang terjadi saat satu run:

1. **Venv workspace disiapkan lebih dulu**, sebelum satu token pun dibakar —
   supaya kegagalan setup ketahuan di awal, bukan di tengah.
2. Workspace di-`git init` (kalau belum), lalu dibuatkan branch baru
   `squad/YYYYMMDD-HHMMSS`. Tiap run hidup di branch sendiri, jadi hasil buruk
   cukup dibuang.
3. **Tahap 1** — Tech Lead menyusun kontrak kerja (spec + acceptance criteria).
   Jalan sekali, hasilnya jadi acuan semua peran. Tech Lead sengaja tidak
   diberi tool sama sekali karena project dimulai dari nol.
4. **Tahap 2** — Developer menulis kode, lalu SQA menjalankan test. Diulang
   sampai `PASS` atau mentok `SQUAD_MAX_QA_ROUNDS` (default 3). Tiap ronde
   di-commit ke branch.
5. **Tahap 3** — DevOps. Hanya jalan kalau verdict `PASS`, `SQUAD_ENABLE_DEVOPS=1`,
   `deploy.sh` ada, **dan** kamu mengetik `DEPLOY` di prompt konfirmasi.

Exit code: `0` sukses (termasuk saat deploy sengaja dilewati), `1` kalau SQA
masih `FAIL` setelah semua ronde habis.

---

## Menjalankan lewat CLI

Semua perintah dijalankan dari dalam `engine/` dengan venv runner aktif:

```bash
cd engine
source .venv/bin/activate
```

Request ditulis sebagai argumen command line:

```bash
python main.py "Buat REST API untuk mencatat buku, dengan endpoint tambah, list, dan hapus"
```

Tanpa argumen, engine memakai request bawaan (todo API sederhana):

```bash
python main.py
```

### Hasilnya di mana

Tiap task diberi **nomor tiket** (`geo-1001`) yang jadi nama foldernya, sehingga
hasil satu task tidak pernah bercampur atau menimpa task lain.

| Path | Isi |
|---|---|
| `workspace/geo-1001/` | kode yang dibangun squad. Punya git sendiri, tidak ikut repo ini |
| `output/geo-1001/` | laporan markdown per tahap (lihat di bawah) |
| `.venvs/geo-1001/` | venv untuk menjalankan test tiket itu |

Nomor tiket diterbitkan API dari `AUTO_INCREMENT` MySQL, jadi tidak mungkin
bentrok. Engine yang dijalankan manual lewat CLI memakai penanda waktu
(`local-20260920-004404`) supaya jelas bukan tiket yang tercatat di database.

Isi `output/geo-1001/` berurutan sesuai tahap:

```
01-spec.md                    kontrak dari Tech Lead
02-developer-round-{n}.md     laporan Developer tiap ronde
03-sqa-round-{n}.md           verdict SQA tiap ronde
04-devops-deploy.md           laporan deploy (kalau DevOps jalan)
```

Melihat hasil satu tiket:

```bash
ls output/geo-1001/
cd workspace/geo-1001 && git log --oneline
```

---

## Konfigurasi

Semua lewat `engine/.env`, tidak perlu menyentuh kode. Daftar lengkap ada di
`.env.example`.

### Model

```ini
NINE_ROUTER_API_KEY=<api key kamu>
NINE_ROUTER_BASE_URL=http://localhost:20128/v1
NINE_ROUTER_MODEL=openai/openrouter/minimax/minimax-m3:free
```

Kalau `NINE_ROUTER_MODEL` kosong dan tidak ada model per peran, engine berhenti
dengan `RuntimeError: NINE_ROUTER_MODEL belum diset di .env`.

Semua peran default ikut `NINE_ROUTER_MODEL`. Begitu punya model berbayar,
cukup isi salah satu tanpa mengubah kode:

```ini
SQUAD_MODEL_LEAD=
SQUAD_MODEL_DEV=     # isi ini duluan, Developer yang paling butuh model kuat
SQUAD_MODEL_QA=
SQUAD_MODEL_OPS=
```

Temperature dikunci di `squad/config.py` per peran: Lead `0.3`, Dev `0.2`,
QA `0.5`, Ops `0.2`.

### Tuning

| Variabel | Default | Fungsi |
|---|---|---|
| `SQUAD_WORKSPACE` | `engine/workspace` | akar folder tempat squad membangun aplikasi |
| `SQUAD_OUTPUT` | `engine/output` | akar folder laporan per tahap |
| `SQUAD_VENVS_DIR` | `engine/.venvs` | akar folder venv per tiket |
| `SQUAD_VENV_DIR` | `<SQUAD_VENVS_DIR>/<tiket>` | menimpa venv untuk satu tiket saja. **Wajib di luar workspace** |
| `SQUAD_TICKET` | penanda waktu | nomor tiket. Normalnya diisi API |
| `SQUAD_MAX_QA_ROUNDS` | `3` | berapa kali Developer↔SQA boleh berputar |
| `SQUAD_MAX_RPM` | `10` | rate limit request ke LLM. Turunkan kalau kena 429 |
| `SQUAD_ALLOW_DEPS` | `1` | `0` mengunci venv workspace, `requirements.txt` dari agent diabaikan |
| `SQUAD_TEST_COMMAND` | *(otomatis per bahasa)* | memaksa satu perintah dan melewati deteksi |
| `SQUAD_TEST_TIMEOUT` | `300` | batas detik untuk test |
| `SQUAD_DEPS_TIMEOUT` | `600` | batas detik untuk install dependency |
| `SQUAD_DEPLOY_TIMEOUT` | `900` | batas detik untuk `deploy.sh` |
| `SQUAD_MAX_FILE_BYTES` | `200000` | batas ukuran file yang boleh ditulis agent |
| `SQUAD_ENABLE_DEVOPS` | `0` | `1` mengaktifkan tahap deploy |
| `SQUAD_WORKSPACE_PY` | versi Python runner | versi Python untuk venv workspace |

`engine/.env` sudah masuk `.gitignore`. Jangan pernah commit file itu — hanya
`engine/.env.example` yang boleh ikut ke repo.

---

## Mengaktifkan DevOps

DevOps **mati secara default**, dan sebaiknya tetap begitu sampai `deploy.sh`
benar-benar dikonfigurasi. Langkahnya:

1. Buka `deploy.sh`, isi konfigurasinya, lalu **hapus blok penjaga** yang
   menulis `exit 78`. Selama blok itu ada, script sengaja selalu gagal.
2. Isi di `engine/.env`:
   ```ini
   DEPLOY_HOST=deploy@203.0.113.10
   DEPLOY_PATH=/srv/app
   SERVICE_NAME=app
   HEALTH_URL=https://app.example.com/health
   SQUAD_ENABLE_DEVOPS=1
   ```
3. Jalankan `bash deploy.sh` manual sekali untuk memastikan benar.

Agent DevOps hanya boleh memanggil `deploy.sh` apa adanya — tanpa argumen dan
tanpa bisa mengubah isinya. Semua perintah yang menyentuh production ditulis
dan direview manusia di file itu.

Deploy juga selalu minta persetujuan manusia. Di sesi non-interaktif (CI, pipe),
`ask_approval` otomatis menolak dan deploy dilewati.

---

## Catatan desain

### Kenapa dua venv

`requirements.txt` di workspace ditulis oleh agent. Kalau venv-nya sama dengan
yang menjalankan CrewAI, satu pin dari agent bisa menyeret turun dependency
runner dan mematikan squad di tengah run. Ini bukan teori: `fastapi==0.110.0`
menarik `starlette` ke `0.36.3`, padahal `sse-starlette` (lewat `mcp` → `crewai`)
butuh `>=0.49.1`.

### Kenapa venv workspace harus di luar workspace

`write_project_file` mengizinkan tulis ke mana pun di dalam workspace, termasuk
`.venv/bin/python`. Kalau venv diletakkan di dalam, agent bisa menimpa
interpreter yang nanti dieksekusi `run_tests` — dan "menulis file" berubah jadi
"menjalankan kode". `squad/config.py` menolak start kalau `SQUAD_VENV_DIR`
berada di dalam workspace.

### Perintah test mengikuti bahasa project

Squad boleh memilih bahasa, dan perintah test menyesuaikan sendiri:

| Bahasa | Penanda | Perintah |
|---|---|---|
| Go | `go.mod` | `go test ./...` |
| Java | `pom.xml` | `mvn -q -B test` |
| Rust | `Cargo.toml` | `cargo test --quiet` |
| PHP | `composer.json` | `composer install` lalu `vendor/bin/phpunit` |
| Ruby | `Gemfile` | `bundle install` lalu `bundle exec rspec` |
| Node | `package.json` | `npm install` lalu `npm test` |
| Python | *(bawaan)* | `<venv tiket>/bin/python -m pytest -q` |

Yang penting di sini bukan daftarnya, tapi dari mana perintahnya berasal.
**Agent tidak pernah mengarang perintah test.** Kalau boleh, "menulis file"
berubah jadi "menjalankan perintah apa pun", dan seluruh pembatasan di
`tools.py` jadi percuma. Yang bisa dilakukan agent hanyalah memilih file apa
yang ia tulis; perintahnya sudah ditetapkan di `squad/stacks.py`, ditulis dan
direview manusia.

Konsekuensinya bahasa baru tidak bisa ditambahkan agent saat runtime. Itu
disengaja: menambah bahasa berarti menambah perintah yang boleh dijalankan di
mesin ini, dan keputusan itu milik manusia.

Tech Lead diberi tahu di prompt bahasa apa saja yang **toolchain-nya benar-benar
terpasang** di mesin, diambil saat runtime. Mesin tanpa Maven tidak akan
menawarkan Java, sehingga tidak ada run yang terbuang memilih bahasa yang tidak
bisa diuji.

### Batas yang dipegang tool

- Semua operasi file dikurung di dalam `WORKSPACE`. Path absolut dan path yang
  keluar workspace **ditolak**, bukan dipotong diam-diam.
- Tool yang menjalankan perintah (`run_tests`, `run_deploy`) **tidak menerima
  argumen apa pun dari LLM**. Perintahnya sudah ditentukan di `squad/config.py`.
  Ini menutup command injection, sekaligus meringankan model gratis yang sering
  keliru memformat argumen.
- `pytest` selalu dipasang ke venv workspace oleh runner. Ia alat ukur SQA,
  bukan dependency aplikasi, jadi tidak boleh bergantung pada apakah agent
  ingat menulisnya di `requirements.txt`.
- Stamp hash `requirements.txt` disimpan di dalam venv, di luar jangkauan agent,
  supaya agent tidak bisa memalsukannya agar install dilewati.

### Tool per peran

Sengaja diberi seirit mungkin — tiap tool tambahan memperluas apa yang bisa rusak.

| Peran | Tool |
|---|---|
| Tech Lead | *(tidak ada)* |
| Developer | `list_project_files`, `read_project_file`, `write_project_file` |
| SQA | `list_project_files`, `read_project_file`, `run_tests` |
| DevOps | `list_project_files`, `read_project_file`, `run_deploy` |

---

## Troubleshooting

**`NINE_ROUTER_MODEL belum diset di .env`**
`engine/.env` belum ada atau `NINE_ROUTER_MODEL` kosong. Salin dari `.env.example`.

**`SQUAD_VENV_DIR (...) berada di dalam workspace`**
Pindahkan `SQUAD_VENV_DIR` ke luar `SQUAD_WORKSPACE`, atau kosongkan saja
supaya pakai default.

**`deploy.sh masih template` (exit 78)**
Blok penjaga di `deploy.sh` belum dihapus. Ini memang disengaja — isi dulu
konfigurasinya.

**SQA selalu `FAIL` sampai ronde habis**
Model gratis sering gagal menghasilkan JSON valid. `parse_verdict` punya jaring
pengaman yang membaca `PASS`/`FAIL` dari teks bebas, tapi kalau sering kena,
isi `SQUAD_MODEL_DEV` dengan model yang lebih kuat.

**Kena rate limit (429)**
Turunkan `SQUAD_MAX_RPM`.

---

## Struktur

```
engine/
├── main.py           # orkestrator + gerbang antar tahap
├── deploy.sh         # script deploy, ditulis & direview manusia
├── requirements.txt  # dependency runner, bukan dependency aplikasi squad
├── .env.example      # template konfigurasi
└── squad/
    ├── config.py     # path, batas keamanan, LLM per peran
    ├── agents.py     # definisi empat peran
    ├── tasks.py      # deskripsi tugas per tahap
    ├── tools.py      # tool agent, dengan batas workspace
    ├── stacks.py     # deteksi bahasa & perintah test
    ├── models.py     # output terstruktur (Spec, DevReport, QAVerdict)
    └── venv.py       # kelola venv workspace yang terpisah
```
