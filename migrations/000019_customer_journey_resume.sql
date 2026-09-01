alter table customer_resume_tokens
  add column if not exists purpose text not null default 'correction',
  add column if not exists revoked_at timestamptz,
  add column if not exists last_used_at timestamptz;

create index if not exists customer_resume_tokens_journey_idx
  on customer_resume_tokens(proposal_acceptance_id,purpose,expires_at desc)
  where revoked_at is null;
