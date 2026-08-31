alter table contracts
  add column if not exists verification_token text;

update contracts
set verification_token = encode(gen_random_bytes(24), 'hex')
where verification_token is null or btrim(verification_token) = '';

alter table contracts
  alter column verification_token set not null;

create unique index if not exists contracts_verification_token_unique
  on contracts(verification_token);
