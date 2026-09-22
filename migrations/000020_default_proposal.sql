alter table proposals
  add column if not exists is_default boolean not null default false;

create unique index if not exists proposals_single_default_idx
  on proposals(is_default)
  where is_default = true;
