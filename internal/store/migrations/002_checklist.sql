-- A release is now just the project's checklist being ticked off.
-- The template rows carry the current run's state; releases keeps the history.
alter table release_templates add column done integer not null default 0;
