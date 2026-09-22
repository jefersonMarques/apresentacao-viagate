drop index if exists proposals_single_default_idx;

create index if not exists proposals_default_idx
  on proposals(updated_at desc)
  where is_default = true;
