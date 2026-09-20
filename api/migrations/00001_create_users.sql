-- +goose Up
CREATE TABLE users (
    id              CHAR(36)     NOT NULL,
    full_name       VARCHAR(150) NOT NULL,
    email           VARCHAR(190) NOT NULL,
    password        VARCHAR(100) NOT NULL,
    role            ENUM('ADMIN','USER') NOT NULL DEFAULT 'USER',
    is_suspended    TINYINT(1)   NOT NULL DEFAULT 0,

    -- Kolom audit, mengikuti BaseEntity yang dipakai project lain.
    created_by      VARCHAR(190) NOT NULL DEFAULT 'System',
    modified_by     VARCHAR(190) NOT NULL DEFAULT 'System',
    created_date    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_deleted      TINYINT(1)   NOT NULL DEFAULT 0,

    PRIMARY KEY (id),
    -- Email unik hanya di antara baris yang belum dihapus. Tanpa is_deleted di
    -- dalam index, email bekas user terhapus akan terkunci selamanya.
    UNIQUE KEY uq_users_email_active (email, is_deleted),
    KEY idx_users_role (role, is_deleted)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE users;
