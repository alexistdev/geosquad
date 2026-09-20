-- +goose Up
-- Ganti email admin bawaan.
--
-- Ditulis sebagai migrasi baru, bukan dengan menyunting 00004: migrasi yang
-- sudah diterapkan tidak pernah dijalankan ulang, jadi menyuntingnya hanya
-- mengubah instalasi baru dan membuat database yang sudah ada tertinggal.
--
-- Syarat email lama ikut disertakan supaya migrasi ini tidak menimpa nilai
-- yang sudah diganti manual lewat cara lain.
UPDATE users
SET email = 'admin@gmail.com',
    modified_by = 'Migration 00006'
WHERE id = '00000000-0000-0000-0000-000000000001'
  AND email = 'admin@geosquad.local';

-- +goose Down
UPDATE users
SET email = 'admin@geosquad.local',
    modified_by = 'Migration 00006 (rollback)'
WHERE id = '00000000-0000-0000-0000-000000000001'
  AND email = 'admin@gmail.com';
