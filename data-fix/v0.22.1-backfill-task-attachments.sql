-- data-fix/v0.22.1-backfill-task-attachments.sql
-- Backfill: перенос вложений входящих в их задачи, которые потерялись
-- до фикса v0.22.1 (шаг 1). ИДЕМПОТЕНТЕН — повторный запуск ничего не добавит
-- (NOT EXISTS по attachment ID).
--
-- Запуск: sqlite3 /path/to/mailbridge.db < this-file
--
-- Диагностика (сколько задач затронуто):
--   SELECT t.id, ti.inbox_item_id,
--     (SELECT COUNT(*) FROM inbox_attachments ia WHERE ia.inbox_item_id=ti.inbox_item_id) AS atts_in,
--     (SELECT COUNT(*) FROM task_attachments ta WHERE ta.task_id=t.id) AS atts_out
--   FROM tasks t
--   JOIN task_inbox_items ti ON ti.task_id=t.id AND ti.inbox_item_id IS NOT NULL
--   WHERE EXISTS (
--       SELECT 1 FROM inbox_attachments ia
--       WHERE ia.inbox_item_id=ti.inbox_item_id
--         AND NOT EXISTS (
--           SELECT 1 FROM task_attachments ta
--           WHERE ta.task_id=t.id AND ta.attachment_id=ia.attachment_id
--         )
--       LIMIT 1
--     )
--   ORDER BY t.id;
--
-- Backfill INSERT:
BEGIN TRANSACTION;

INSERT OR IGNORE INTO task_attachments (task_id, attachment_id)
SELECT t.id, ia.attachment_id
FROM tasks t
JOIN task_inbox_items ti ON ti.task_id=t.id AND ti.inbox_item_id IS NOT NULL
JOIN inbox_attachments ia ON ia.inbox_item_id=ti.inbox_item_id
WHERE NOT EXISTS (
    SELECT 1 FROM task_attachments ta
    WHERE ta.task_id=t.id AND ta.attachment_id=ia.attachment_id
);

-- Сколько перенесли:
SELECT 'backfilled rows' AS what, changes() AS n;

COMMIT;

-- Верификация: должны быть 0 задач с недостающими вложениями:
SELECT COUNT(*) AS tasks_with_missing
FROM tasks t
JOIN task_inbox_items ti ON ti.task_id=t.id AND ti.inbox_item_id IS NOT NULL
WHERE EXISTS (
    SELECT 1 FROM inbox_attachments ia
    WHERE ia.inbox_item_id=ti.inbox_item_id
      AND NOT EXISTS (
        SELECT 1 FROM task_attachments ta
        WHERE ta.task_id=t.id AND ta.attachment_id=ia.attachment_id
      )
);
