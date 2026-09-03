-- "Чекаю": the task hangs on something external; waiting is what it waits for.
alter table tasks add column waiting text not null default '';
alter table tasks add column waiting_since text;
