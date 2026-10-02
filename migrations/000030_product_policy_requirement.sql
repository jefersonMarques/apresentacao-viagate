-- Product-level operational requirements.
-- Existing catalog products preserve the previous global behavior (policy/goods
-- required). New products default to not requiring them unless explicitly set.

alter table products
  add column if not exists requires_policy boolean;

update products
set requires_policy=true
where requires_policy is null;

alter table products
  alter column requires_policy set default false,
  alter column requires_policy set not null;

-- Existing proposal versions preserve the behavior that was active when they
-- were published: policy/goods were globally required. New versions receive an
-- explicit snapshot computed by the proposal editor.
alter table proposal_versions
  add column if not exists requires_policy boolean not null default true;

alter table proposal_versions
  alter column requires_policy set default false;
