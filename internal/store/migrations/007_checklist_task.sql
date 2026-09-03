-- A checklist line can point at a task ("do this one before the release").
alter table release_templates add column task_id integer references tasks(id) on delete set null;
