alter table users
  add column if not exists phone text,
  add column if not exists job_title text;

alter table users
  add constraint users_phone_raw_check
  check (phone is null or phone = '' or phone ~ '^([0-9]{10,11}|55[0-9]{10,11})$') not valid;
