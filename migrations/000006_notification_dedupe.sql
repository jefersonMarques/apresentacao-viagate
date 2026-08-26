alter table notification_outbox
  add column if not exists dedupe_key text,
  add column if not exists processing_at timestamptz;

create unique index if not exists notification_outbox_dedupe_unique
  on notification_outbox(dedupe_key)
  where dedupe_key is not null;

create index if not exists notification_outbox_processing_idx
  on notification_outbox(status, processing_at)
  where status = 'processing';
