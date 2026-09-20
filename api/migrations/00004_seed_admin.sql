-- +goose Up
-- Admin awal supaya API bisa dipakai tanpa perlu registrasi manual dulu.
-- Password: password  (bcrypt cost 10)
-- WAJIB diganti setelah login pertama di production.
INSERT INTO users (id, full_name, email, password, role, created_by, modified_by)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Administrator',
    'admin@geosquad.local',
    '$2a$10$lYQQgHdoCtAhFyenhmDyZ.o9u1WkHfcSCUyDbAWxOZ1N34b/Pmdjy',
    'ADMIN',
    'System',
    'System'
);

-- +goose Down
DELETE FROM users WHERE id = '00000000-0000-0000-0000-000000000001';
