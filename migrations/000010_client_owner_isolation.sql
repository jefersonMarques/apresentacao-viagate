drop index if exists clients_cnpj_unique;

create unique index clients_owner_cnpj_unique
  on clients(created_by, cnpj)
  where cnpj is not null;
