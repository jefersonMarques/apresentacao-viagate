-- Rascunhos comerciais podem ser salvos antes da identificação completa do cliente.
-- O onboarding continua validando CNPJ e razão social antes do envio final.
alter table clients
  alter column legal_name drop not null;

alter table onboardings
  alter column cnpj drop not null,
  alter column legal_name drop not null;
