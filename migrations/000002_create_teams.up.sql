CREATE TABLE IF NOT EXISTS teams (
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name       VARCHAR(255)    NOT NULL,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_teams_created_by FOREIGN KEY (created_by) REFERENCES users(id),
    KEY idx_teams_created_by (created_by)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
