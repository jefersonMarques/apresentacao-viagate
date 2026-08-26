alter table clients
  add constraint clients_cnpj_raw_check
  check (cnpj is null or cnpj ~ '^[0-9A-Z]{12}[0-9]{2}$') not valid,
  add constraint clients_phone_raw_check
  check (phone is null or phone = '' or phone ~ '^([0-9]{10,11}|55[0-9]{10,11})$') not valid,
  add constraint clients_postal_code_raw_check
  check (postal_code is null or postal_code = '' or postal_code ~ '^[0-9]{8}$') not valid;

alter table proposal_acceptances
  add constraint proposal_acceptances_cpf_raw_check
  check (accepted_by_cpf ~ '^[0-9]{11}$') not valid,
  add constraint proposal_acceptances_phone_raw_check
  check (accepted_by_phone ~ '^([0-9]{10,11}|55[0-9]{10,11})$') not valid;

alter table onboardings
  add constraint onboardings_cnpj_raw_check
  check (cnpj ~ '^[0-9A-Z]{12}[0-9]{2}$') not valid,
  add constraint onboardings_responsible_cpf_raw_check
  check (company_responsible_cpf ~ '^[0-9]{11}$') not valid,
  add constraint onboardings_responsible_phone_raw_check
  check (company_responsible_phone ~ '^([0-9]{10,11}|55[0-9]{10,11})$') not valid,
  add constraint onboardings_finance_phone_raw_check
  check (finance_responsible_phone is null or finance_responsible_phone = '' or finance_responsible_phone ~ '^([0-9]{10,11}|55[0-9]{10,11})$') not valid,
  add constraint onboardings_postal_code_raw_check
  check (postal_code is null or postal_code = '' or postal_code ~ '^[0-9]{8}$') not valid;

alter table onboarding_system_users
  add constraint onboarding_system_users_phone_raw_check
  check (phone is null or phone = '' or phone ~ '^([0-9]{10,11}|55[0-9]{10,11})$') not valid;

alter table contract_signers
  add constraint contract_signers_cpf_raw_check
  check (cpf ~ '^[0-9]{11}$') not valid;
