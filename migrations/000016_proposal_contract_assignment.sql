alter table proposals
  add column if not exists contract_template_id uuid references contract_templates(id);

alter table proposal_versions
  add column if not exists contract_template_version_id uuid references contract_template_versions(id);

create index if not exists proposals_contract_template_idx
  on proposals(contract_template_id)
  where contract_template_id is not null;

create index if not exists proposal_versions_contract_template_version_idx
  on proposal_versions(contract_template_version_id)
  where contract_template_version_id is not null;
