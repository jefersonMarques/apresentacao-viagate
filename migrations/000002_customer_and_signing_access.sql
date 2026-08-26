alter table contract_signers add column if not exists public_token uuid not null default gen_random_uuid();
create unique index if not exists contract_signers_public_token_unique on contract_signers(public_token);

create table if not exists customer_sessions (
  id uuid primary key default gen_random_uuid(),
  proposal_acceptance_id uuid not null references proposal_acceptances(id) on delete cascade,
  token_hash bytea not null unique,
  ip_address inet,
  user_agent text,
  expires_at timestamptz not null,
  created_at timestamptz not null default now(),
  revoked_at timestamptz
);

create index if not exists customer_sessions_acceptance_idx
  on customer_sessions(proposal_acceptance_id, expires_at desc);
