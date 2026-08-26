alter table onboardings
  add column if not exists review_notes text,
  add column if not exists reviewed_by uuid references users(id),
  add column if not exists reviewed_at timestamptz;

create unique index if not exists contracts_one_active_per_onboarding
  on contracts(onboarding_id)
  where status <> 'cancelled';

create index if not exists onboardings_review_queue_idx
  on onboardings(status, submitted_at desc)
  where status in ('submitted','under_review','correction_requested','approved');
