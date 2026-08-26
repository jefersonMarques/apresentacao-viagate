alter table contract_templates add column if not exists is_default boolean not null default false;

create unique index if not exists contract_templates_single_default
  on contract_templates(is_default)
  where is_default = true and is_active = true;
