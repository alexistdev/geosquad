-- +goose Up
-- Nomor tiket yang memisahkan hasil kerja tiap task.
--
-- Sumbernya AUTO_INCREMENT, bukan dihitung di Go, supaya dua request yang
-- masuk bersamaan tidak mungkin mendapat nomor yang sama. MySQL mengizinkan
-- satu kolom AUTO_INCREMENT per tabel selama kolomnya ter-index, jadi ini
-- tetap bisa berdampingan dengan primary key CHAR(36) yang sudah ada.
--
-- Hanya angkanya yang disimpan. Bentuk "geo-1001" dirakit saat SELECT, bukan
-- disimpan sebagai kolom kedua: satu sumber kebenaran tidak bisa melenceng
-- dari dirinya sendiri. (MySQL juga menolak generated column yang merujuk
-- kolom AUTO_INCREMENT -- error 3109.)
ALTER TABLE runs
    ADD COLUMN ticket_seq BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE AFTER id;

-- Mulai dari 1001 supaya nomornya empat digit sejak awal, bukan geo-1.
ALTER TABLE runs AUTO_INCREMENT = 1001;

-- +goose Down
ALTER TABLE runs DROP COLUMN ticket_seq;
