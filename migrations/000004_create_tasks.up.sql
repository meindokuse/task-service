CREATE TABLE IF NOT EXISTS tasks (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    team_id     BIGINT UNSIGNED NOT NULL,
    title       VARCHAR(255)    NOT NULL,
    description TEXT            NOT NULL,
    status      ENUM('todo', 'in_progress', 'done') NOT NULL DEFAULT 'todo',
    assignee_id BIGINT UNSIGNED NULL,
    created_by  BIGINT UNSIGNED NOT NULL,
    created_at  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_tasks_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    CONSTRAINT fk_tasks_assignee FOREIGN KEY (assignee_id) REFERENCES users(id),
    CONSTRAINT fk_tasks_created_by FOREIGN KEY (created_by) REFERENCES users(id),
    KEY idx_tasks_team_status (team_id, status),
    KEY idx_tasks_assignee_id (assignee_id),
    KEY idx_tasks_created_by_created_at (created_by, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
