create table login_failures (
  id bigint generated always as identity primary key,
  email citext not null,
  ip_address inet,
  created_at timestamptz not null default now()
);

create index login_failures_email_ip_created_idx
  on login_failures(email, ip_address, created_at desc);

create index login_failures_ip_created_idx
  on login_failures(ip_address, created_at desc);

create index login_failures_created_idx
  on login_failures(created_at);
