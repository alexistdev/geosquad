-- +goose Up
-- Log baris per baris dari stdout/stderr engine. Dipisah dari tabel runs karena
-- jumlahnya ribuan per run dan tidak pernah ikut terbawa saat melist run.
CREATE TABLE run_logs (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    run_id      CHAR(36)     NOT NULL,
    -- seq menjaga urutan tampil. AUTO_INCREMENT saja tidak cukup kalau nanti
    -- log ditulis batch dari beberapa goroutine.
    seq         INT          NOT NULL,
    stream      ENUM('STDOUT','STDERR','SYSTEM') NOT NULL DEFAULT 'STDOUT',
    line        TEXT         NOT NULL,
    logged_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    UNIQUE KEY uq_run_logs_seq (run_id, seq),
    CONSTRAINT fk_run_logs_run FOREIGN KEY (run_id) REFERENCES runs (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE run_logs;
