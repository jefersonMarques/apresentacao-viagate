create table customer_resume_tokens (
  id uuid primary key default gen_random_uuid(),
  proposal_acceptance_id uuid not null references proposal_acceptances(id) on delete cascade,
  token_hash bytea not null unique,
  expires_at timestamptz not null,
  used_at timestamptz,
  created_at timestamptz not null default now()
);

create index customer_resume_tokens_active_idx
  on customer_resume_tokens(expires_at)
  where used_at is null;
