-- Archived checklist lines keep their task link too.
alter table release_items add column task_id integer references tasks(id) on delete set null;
