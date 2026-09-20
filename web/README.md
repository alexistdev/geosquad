# GeoSquad Web

Frontend Angular untuk [GeoSquad](../README.md). Tempat menulis request,
menjalankan squad, dan menonton lognya mengalir realtime.

Berbicara ke [`api/`](../api/README.md).

---

## Menjalankan

API harus jalan lebih dulu di `:8084`.

```bash
cd api && make run     # terminal 1
cd web && npm install && npm start   # terminal 2
```

Buka `http://localhost:4200`. Akun bawaan:

```
admin@gmail.com / password
```

`npm start` sudah memakai `proxy.conf.json`, yang meneruskan `/api/*` ke
`localhost:8084`. Artinya browser melihat semuanya satu origin, sehingga cookie
sesi terkirim tanpa urusan CORS saat pengembangan.

---

## Halaman

| Rute | Akses | Isi |
|---|---|---|
| `/login` | tamu | masuk |
| `/register` | tamu | daftar, langsung login setelah berhasil |
| `/user/dashboard` | login | run milik sendiri + form menjalankan squad |
| `/admin/dashboard` | ADMIN | sama, tapi memuat run semua user |
| `/admin/users` | ADMIN | kelola akun, peran, status aktif |
| `/runs/:id` | pemilik atau ADMIN | detail run + log realtime |

Semua halaman di-lazy load. Bundle awal sekitar **89 kB terkompresi** — layar
admin tidak ikut terunduh oleh pengunjung yang belum masuk.

---

## Autentikasi

Sesi dipegang cookie `SID` yang **HttpOnly**, dipasang server saat login.
Artinya JavaScript tidak bisa membacanya, jadi XSS tidak otomatis berarti sesi
tercuri.

Konsekuensinya frontend tidak bisa "melihat" apakah dirinya login hanya dari
cookie. Yang dilakukan:

- Profil user (bukan token) disalin ke `localStorage` supaya guard bisa
  memutuskan seketika saat halaman dibuka, tanpa layar kosong menunggu jaringan.
- Salinan itu **petunjuk, bukan otoritas**. Server memeriksa tiap request, dan
  salinan yang dipalsukan paling jauh menampilkan menu yang API-nya akan menolak.
- Saat tidak ada salinan — tab baru, atau storage dibersihkan — `authGuard`
  bertanya ke `/auth/me`, karena cookie bisa saja masih hidup meski frontend
  tidak mengingat apa-apa.
- `errorInterceptor` menangkap 401 di tengah sesi (kedaluwarsa, dicabut admin,
  logout di tab lain) dan menendang user ke `/login`.

Ini berbeda dari geobill, yang menyimpan kredensial terenkripsi di
`localStorage` karena memakai HTTP Basic Auth. Di sini tidak ada kredensial
apa pun yang menyentuh browser storage — ada test yang menjaga itu tetap benar.

### Aturan otorisasi tidak diduplikasi

Pagar seperti "admin aktif terakhir tidak bisa diturunkan" hanya ada di server.
Frontend menampilkan penolakannya, tidak ikut menghitung sendiri. Menyalin
aturan ke dua tempat hanya menciptakan dua pihak yang bisa berbeda pendapat.

---

## Log realtime

Halaman detail run memakai **Server-Sent Events**, bukan polling.

`EventSource` bawaan browser dipakai langsung, bukan `HttpClient`: Angular
menunggu response selesai sebelum memberikan hasilnya, sedangkan SSE justru
response yang sengaja tidak pernah selesai.

```
buka /runs/:id
   │
   ├─ run sudah selesai  → ambil log sekali dari database
   └─ run masih berjalan → EventSource ke /runs/:id/stream?afterSeq=0
                              │
                              ├─ server kirim riwayat dulu
                              ├─ lalu menyusul baris live
                              └─ event "done" → tutup, ambil ulang detail run
```

`afterSeq` membuat penyambungan ulang tidak mengulang baris yang sudah tampil.
EventSource menyambung sendiri saat koneksi putus.

Log menggulung otomatis ke bawah, kecuali kamu sedang menggulir ke atas untuk
membaca — deteksinya dari posisi scroll, dengan toleransi 40px supaya pembulatan
piksel tidak mematikannya. Tombol "Ikuti baris terbaru" muncul saat itu terjadi.

Daftar run di dashboard tidak memakai SSE (koneksinya per run, bukan per
daftar). Di sana dipakai polling 5 detik yang **berhenti sendiri** begitu tidak
ada run yang berstatus berjalan.

---

## Struktur

```
web/src/app/
├── core/
│   ├── models/         # BaseResponse, User, Run, RunLog
│   ├── services/       # auth, run, user
│   ├── guards/         # authGuard, roleGuard, guestGuard
│   └── interceptors/   # credentials (cookie), error (pesan + 401)
├── shared/
│   ├── layout/         # kerangka halaman setelah login
│   └── ui/             # komponen kecil dipakai ulang
└── pages/              # satu folder per halaman
```

State memakai **signal**, bukan `BehaviorSubject`. `AuthService.user()` adalah
satu-satunya sumber kebenaran status login, dan `isAdmin()` diturunkan darinya
lewat `computed` — tidak ada dua tempat yang bisa tidak sinkron.

---

## Tampilan

Memakai template **Velzon** (Bootstrap 5), varian layout *vertical sidebar*.

### Aset

Hanya bagian yang terpakai yang dibawa ke `public/assets/` — **1,7 MB**, bukan
135 MB versi lengkapnya:

| Folder | Isi |
|---|---|
| `css/` | `bootstrap.min`, `app.min`, `icons.min`, `custom.min` |
| `fonts/` | hanya `.woff2`, dan hanya keluarga yang dipakai |
| `images/` | logo dan favicon |

Format font lama (`eot`, `svg`, `ttf`, `woff`) tidak dibawa; semua browser yang
relevan mendukung woff2. Ikon disatukan ke **satu keluarga** (Remix Icon) —
Boxicons dan Material Design Icons sempat dipakai 1-2 kali, tapi masing-masing
menyeret font ratusan KB.

CSS dimuat lewat `<link>` di `index.html`, **bukan** lewat array `styles` di
`angular.json`. Alasannya `icons.min.css` merujuk fontnya secara relatif
(`../fonts/…`); kalau Angular membundelnya jadi satu file di root, jalur itu
menunjuk ke tempat yang salah dan semua ikon hilang.

### JavaScript Velzon tidak dipakai

Velzon membawa `app.js` yang mengatur tema dan sidebar dengan memanipulasi DOM
langsung serta memasang listener sendiri. Itu tidak dimuat: Angular yang
memiliki DOM di sini, dan dua pihak yang sama-sama mengubahnya akan saling
menimpa. Yang diambil hanya **kontraknya** — nama atribut `data-*` pada
`<html>` dan nilai yang diterimanya — lalu ditulis ulang sebagai
`ThemeService`.

Hal yang sama berlaku untuk Bootstrap JS: dropdown user di topbar dikelola
Angular sendiri, karena satu dropdown tidak sebanding dengan menambah library
yang memasang listener di elemen yang dikendalikan Angular.

### Yang ditulis sendiri

`src/styles.css` hanya memuat yang khas GeoSquad: logo teks, kotak log run, dan
penanda aliran langsung. Kelas bawaan Velzon (`.btn`, `.card`, `.badge`,
`.table`) **tidak** didefinisikan ulang — menimpanya berarti memelihara dua
sistem desain yang akan saling menyalip.

Mode terang/gelap mengikuti setelan sistem saat pertama dibuka, lalu bisa
diganti lewat tombol di topbar dan diingat. Animasi dimatikan untuk yang
memilih gerakan minimal (`prefers-reduced-motion`).

### Menyiapkan ulang aset

Aset hasil salinan sudah ikut di repo, jadi `npm install && npm start` langsung
jalan. Kalau suatu saat perlu menyalin ulang dari paket Velzon aslinya, sumber
yang dipakai adalah varian `corporate` versi HTML.

---

## Pengembangan

```bash
npm start      # dev server + proxy ke API
npm run build  # build produksi ke dist/
npm test       # 20 test (Karma + Jasmine)
```

Test menyasar bagian yang paling mudah rusak diam-diam: perilaku guard,
penerjemahan error, dan jaminan bahwa tidak ada token yang bocor ke
`localStorage`.

### Deploy

`npm run build` menghasilkan file statis di `dist/geosquad-web/browser/`.
Layani dengan web server apa pun, dengan dua syarat:

1. **Semua rute diarahkan ke `index.html`** (SPA fallback), kalau tidak refresh
   di `/runs/abc` akan menghasilkan 404 dari web server.
2. **`/api` di-proxy ke API**, supaya tetap satu origin dan cookie sesi bekerja.
   Kalau API dilayani dari domain berbeda, isi `apiUrl` di
   `src/environments/environment.production.ts` dan tambahkan domain frontend ke
   `CORS_ALLOWED_ORIGINS` di `.env` API.
