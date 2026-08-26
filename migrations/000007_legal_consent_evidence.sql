alter table proposal_acceptances
  add column if not exists acceptance_text text,
  add column if not exists acceptance_text_sha256 bytea;

alter table contract_signers
  add column if not exists signature_consent_version text,
  add column if not exists signature_consent_text text,
  add column if not exists signature_consent_sha256 bytea;

create or replace function prevent_proposal_acceptance_mutation()
returns trigger language plpgsql as $$
begin
  raise exception 'proposal acceptances are immutable';
end;
$$;

drop trigger if exists proposal_acceptances_immutable on proposal_acceptances;
create trigger proposal_acceptances_immutable
before update or delete on proposal_acceptances
for each row execute function prevent_proposal_acceptance_mutation();

create or replace function protect_signed_signer_evidence()
returns trigger language plpgsql as $$
begin
  if old.status = 'signed' and (
    new.name is distinct from old.name or
    new.email is distinct from old.email or
    new.cpf is distinct from old.cpf or
    new.role is distinct from old.role or
    new.signed_at is distinct from old.signed_at or
    new.signed_document_hash is distinct from old.signed_document_hash or
    new.signature_session_id is distinct from old.signature_session_id or
    new.signature_consent_version is distinct from old.signature_consent_version or
    new.signature_consent_text is distinct from old.signature_consent_text or
    new.signature_consent_sha256 is distinct from old.signature_consent_sha256
  ) then
    raise exception 'signed signer evidence is immutable';
  end if;
  return new;
end;
$$;

drop trigger if exists contract_signers_evidence_immutable on contract_signers;
create trigger contract_signers_evidence_immutable
before update on contract_signers
for each row execute function protect_signed_signer_evidence();
