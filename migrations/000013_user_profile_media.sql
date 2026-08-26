alter table users
  add column if not exists photo_url text,
  add column if not exists linkedin_url text,
  add column if not exists instagram_url text;
