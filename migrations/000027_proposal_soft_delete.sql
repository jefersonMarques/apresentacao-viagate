-- Superadmin proposal removal is a logical cascade. Business records disappear
-- from the normal application flow, while immutable proposal versions,
-- signature events, audit events and stored contract artifacts remain retained
-- as evidence.
alter table proposals
  add column if not exists deleted_at timestamptz,
  add column if not exists deleted_by uuid references users(id) on delete set null;

alter table proposal_acceptances
  add column if not exists deleted_at timestamptz,
  add column if not exists deleted_by uuid references users(id) on delete set null;

alter table onboardings
  add column if not exists deleted_at timestamptz,
  add column if not exists deleted_by uuid references users(id) on delete set null;

alter table uploaded_documents
  add column if not exists deleted_at timestamptz,
  add column if not exists deleted_by uuid references users(id) on delete set null;

alter table contracts
  add column if not exists deleted_at timestamptz,
  add column if not exists deleted_by uuid references users(id) on delete set null;

alter table contract_signers
  add column if not exists deleted_at timestamptz,
  add column if not exists deleted_by uuid references users(id) on delete set null;

alter table activation_profiles
  add column if not exists deleted_at timestamptz,
  add column if not exists deleted_by uuid references users(id) on delete set null;

alter table in_app_notifications
  add column if not exists deleted_at timestamptz;

alter table contract_finalization_jobs
  drop constraint if exists contract_finalization_jobs_status_check;

alter table contract_finalization_jobs
  add constraint contract_finalization_jobs_status_check
  check (status in ('pending','processing','completed','failed','cancelled'));

create index if not exists proposals_active_updated_idx
  on proposals(updated_at desc)
  where deleted_at is null;

create index if not exists proposal_acceptances_active_proposal_idx
  on proposal_acceptances(proposal_id,accepted_at desc)
  where deleted_at is null;

create index if not exists onboardings_active_updated_idx
  on onboardings(updated_at desc)
  where deleted_at is null;

create index if not exists uploaded_documents_active_onboarding_idx
  on uploaded_documents(onboarding_id,document_type,uploaded_at desc)
  where deleted_at is null;

create index if not exists contracts_active_updated_idx
  on contracts(updated_at desc)
  where deleted_at is null;

create index if not exists contract_signers_active_public_token_idx
  on contract_signers(public_token)
  where deleted_at is null;

create index if not exists activation_profiles_active_updated_idx
  on activation_profiles(updated_at desc)
  where deleted_at is null;

create index if not exists in_app_notifications_active_recipient_idx
  on in_app_notifications(recipient_user_id,created_at desc)
  where deleted_at is null;
