CREATE TABLE IF NOT EXISTS task_comments (
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    task_id    BIGINT UNSIGNED NOT NULL,
    user_id    BIGINT UNSIGNED NOT NULL,
    content    TEXT            NOT NULL,
    created_at DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_task_comments_task FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    CONSTRAINT fk_task_comments_user FOREIGN KEY (user_id) REFERENCES users(id),
    KEY idx_task_comments_task_id (task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
