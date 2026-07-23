CREATE TABLE IF NOT EXISTS task_history (
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    task_id    BIGINT UNSIGNED NOT NULL,
    changed_by BIGINT UNSIGNED NOT NULL,
    field_name VARCHAR(64)     NOT NULL,
    old_value  VARCHAR(1024)   NULL,
    new_value  VARCHAR(1024)   NULL,
    changed_at DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_task_history_task FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    CONSTRAINT fk_task_history_user FOREIGN KEY (changed_by) REFERENCES users(id),
    KEY idx_task_history_task_id (task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
