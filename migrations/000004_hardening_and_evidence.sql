alter table contracts
  add column if not exists evidence_report_storage_key text,
  add column if not exists evidence_report_sha256 bytea,
  add column if not exists final_package_storage_key text,
  add column if not exists final_package_sha256 bytea,
  add column if not exists finalized_at timestamptz;

create table if not exists document_events (
  id bigint generated always as identity primary key,
  document_kind text not null check (document_kind in ('proposal','presentation')),
  document_version_id uuid not null,
  event_type text not null,
  viewer_session uuid,
  ip_address inet,
  user_agent text,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create index if not exists document_events_version_idx
  on document_events(document_kind, document_version_id, created_at desc);

create or replace function prevent_published_proposal_version_mutation()
returns trigger language plpgsql as $$
begin
  if tg_op = 'DELETE' and old.published_at is not null then
    raise exception 'published proposal versions are immutable';
  end if;
  if tg_op = 'UPDATE' and old.published_at is not null then
    raise exception 'published proposal versions are immutable';
  end if;
  return case when tg_op = 'DELETE' then old else new end;
end;
$$;

drop trigger if exists proposal_versions_immutable on proposal_versions;
create trigger proposal_versions_immutable
before update or delete on proposal_versions
for each row execute function prevent_published_proposal_version_mutation();

create or replace function prevent_published_presentation_version_mutation()
returns trigger language plpgsql as $$
begin
  if tg_op = 'DELETE' and old.published_at is not null then
    raise exception 'published presentation versions are immutable';
  end if;
  if tg_op = 'UPDATE' and old.published_at is not null then
    raise exception 'published presentation versions are immutable';
  end if;
  return case when tg_op = 'DELETE' then old else new end;
end;
$$;

drop trigger if exists presentation_versions_immutable on presentation_versions;
create trigger presentation_versions_immutable
before update or delete on presentation_versions
for each row execute function prevent_published_presentation_version_mutation();

create or replace function prevent_contract_template_version_mutation()
returns trigger language plpgsql as $$
begin
  raise exception 'contract template versions are immutable';
end;
$$;

drop trigger if exists contract_template_versions_immutable on contract_template_versions;
create trigger contract_template_versions_immutable
before update or delete on contract_template_versions
for each row execute function prevent_contract_template_version_mutation();

create or replace function protect_generated_contract_document()
returns trigger language plpgsql as $$
begin
  if old.generated_at is not null and (
    new.onboarding_id is distinct from old.onboarding_id or
    new.proposal_version_id is distinct from old.proposal_version_id or
    new.template_version_id is distinct from old.template_version_id or
    new.rendered_markdown is distinct from old.rendered_markdown or
    new.rendered_html is distinct from old.rendered_html or
    new.pdf_storage_key is distinct from old.pdf_storage_key or
    new.document_sha256 is distinct from old.document_sha256 or
    new.generated_at is distinct from old.generated_at
  ) then
    raise exception 'generated contract document is immutable';
  end if;
  return new;
end;
$$;

drop trigger if exists contracts_document_immutable on contracts;
create trigger contracts_document_immutable
before update on contracts
for each row execute function protect_generated_contract_document();

create or replace function prevent_append_only_mutation()
returns trigger language plpgsql as $$
begin
  raise exception 'append-only records cannot be changed';
end;
$$;

drop trigger if exists signature_events_append_only on signature_events;
create trigger signature_events_append_only
before update or delete on signature_events
for each row execute function prevent_append_only_mutation();

drop trigger if exists audit_events_append_only on audit_events;
create trigger audit_events_append_only
before update or delete on audit_events
for each row execute function prevent_append_only_mutation();

create index if not exists contract_signers_contract_status_idx
  on contract_signers(contract_id, status, sign_order);

create index if not exists proposal_acceptances_accepted_at_idx
  on proposal_acceptances(accepted_at desc);

create index if not exists uploaded_documents_onboarding_idx
  on uploaded_documents(onboarding_id, document_type, uploaded_at desc);
