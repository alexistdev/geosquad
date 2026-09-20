-- +goose Up
CREATE TABLE runs (
    id              CHAR(36)     NOT NULL,
    user_id         CHAR(36)     NOT NULL,
    request         TEXT         NOT NULL,
    status          ENUM('PENDING','RUNNING','PASSED','FAILED','CANCELLED','ERROR')
                                 NOT NULL DEFAULT 'PENDING',
    -- Branch git yang dibuat engine di dalam workspace, format squad/<timestamp>.
    branch          VARCHAR(120) NULL,
    exit_code       INT          NULL,
    error_message   TEXT         NULL,
    started_at      DATETIME     NULL,
    finished_at     DATETIME     NULL,

    created_by      VARCHAR(190) NOT NULL DEFAULT 'System',
    modified_by     VARCHAR(190) NOT NULL DEFAULT 'System',
    created_date    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_deleted      TINYINT(1)   NOT NULL DEFAULT 0,

    PRIMARY KEY (id),
    KEY idx_runs_user (user_id, is_deleted, created_date),
    KEY idx_runs_status (status, is_deleted),
    CONSTRAINT fk_runs_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE runs;
