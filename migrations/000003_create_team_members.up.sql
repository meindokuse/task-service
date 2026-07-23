CREATE TABLE IF NOT EXISTS team_members (
    team_id   BIGINT UNSIGNED NOT NULL,
    user_id   BIGINT UNSIGNED NOT NULL,
    role      ENUM('owner', 'admin', 'member') NOT NULL DEFAULT 'member',
    joined_at DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (team_id, user_id),
    CONSTRAINT fk_team_members_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    CONSTRAINT fk_team_members_user FOREIGN KEY (user_id) REFERENCES users(id),
    KEY idx_team_members_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
