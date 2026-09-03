create table projects (
  id         integer primary key,
  name       text not null unique,
  slug       text not null unique,
  color      text not null,
  kind       text not null check (kind in ('work', 'pet')),
  jira_key   text not null default '',
  jira_host  text not null default '',
  repos      text not null default '',
  channels   text not null default '',
  on_board   integer not null default 1,
  position   integer not null default 0
);

create table releases (
  id          integer primary key,
  project_id  integer not null references projects(id),
  name        text not null,
  date        text,
  released_at text,
  created_at  text not null default (datetime('now'))
);

create table tasks (
  id          integer primary key,
  project_id  integer not null references projects(id),
  title       text not null,
  state       text not null default 'backlog' check (state in ('now', 'backlog', 'done')),
  notes       text not null default '',
  position    integer not null default 0,
  release_id  integer references releases(id) on delete set null,
  created_at  text not null default (datetime('now')),
  now_since   text,
  done_at     text
);
create index tasks_state_position on tasks(state, position);

create table task_links (
  id       integer primary key,
  task_id  integer not null references tasks(id) on delete cascade,
  url      text not null,
  kind     text not null default '',
  label    text not null default '',
  meta     text not null default '',
  position integer not null default 0
);

create table task_steps (
  id       integer primary key,
  task_id  integer not null references tasks(id) on delete cascade,
  title    text not null,
  done     integer not null default 0,
  position integer not null default 0
);

create table release_templates (
  id          integer primary key,
  project_id  integer not null references projects(id) on delete cascade,
  phase       text not null check (phase in ('before', 'after')),
  title       text not null,
  detail      text not null default '',
  url         text not null default '',
  command     text not null default '',
  position    integer not null default 0
);

create table release_items (
  id          integer primary key,
  release_id  integer not null references releases(id) on delete cascade,
  phase       text not null check (phase in ('before', 'after')),
  title       text not null,
  detail      text not null default '',
  url         text not null default '',
  command     text not null default '',
  done        integer not null default 0,
  position    integer not null default 0
);
