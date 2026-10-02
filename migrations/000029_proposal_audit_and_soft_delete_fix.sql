-- Proposal ownership remains immutable. Editing history is tracked separately
-- through updated_by and append-only audit events. Legal acceptance evidence
-- remains immutable while soft-delete metadata may be added administratively.

alter table proposals
  add column if not exists updated_by uuid references users(id) on delete set null;

update proposals
set updated_by=created_by
where updated_by is null;

create index if not exists proposals_updated_by_idx
  on proposals(updated_by,updated_at desc)
  where deleted_at is null;

alter table proposal_versions
  add column if not exists updated_by uuid references users(id) on delete set null,
  add column if not exists updated_at timestamptz;

update proposal_versions
set updated_by=created_by
where updated_by is null;

update proposal_versions
set updated_at=created_at
where updated_at is null;

alter table proposal_versions
  alter column updated_at set default now(),
  alter column updated_at set not null;

create index if not exists proposal_versions_updated_by_idx
  on proposal_versions(updated_by,updated_at desc);

create or replace function prevent_proposal_acceptance_mutation()
returns trigger language plpgsql as $$
begin
  if tg_op = 'DELETE' then
    raise exception 'proposal acceptances are immutable';
  end if;

  if (
    to_jsonb(new) - 'deleted_at' - 'deleted_by'
  ) is distinct from (
    to_jsonb(old) - 'deleted_at' - 'deleted_by'
  ) then
    raise exception 'proposal acceptance evidence is immutable';
  end if;

  return new;
end;
$$;

drop trigger if exists proposal_acceptances_immutable on proposal_acceptances;
create trigger proposal_acceptances_immutable
before update or delete on proposal_acceptances
for each row execute function prevent_proposal_acceptance_mutation();

create or replace function protect_proposal_owner()
returns trigger language plpgsql as $$
begin
  if new.created_by is distinct from old.created_by then
    raise exception 'proposal owner is immutable';
  end if;
  return new;
end;
$$;

drop trigger if exists proposals_owner_immutable on proposals;
create trigger proposals_owner_immutable
before update on proposals
for each row execute function protect_proposal_owner();

create or replace function protect_proposal_version_creator()
returns trigger language plpgsql as $$
begin
  if new.created_by is distinct from old.created_by then
    raise exception 'proposal version creator is immutable';
  end if;
  return new;
end;
$$;

drop trigger if exists proposal_versions_creator_immutable on proposal_versions;
create trigger proposal_versions_creator_immutable
before update on proposal_versions
for each row execute function protect_proposal_version_creator();
