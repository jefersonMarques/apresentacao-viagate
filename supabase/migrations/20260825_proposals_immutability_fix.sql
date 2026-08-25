create or replace function public.prevent_published_version_mutation()
returns trigger
language plpgsql
as $$
begin
  if old.published_at is not null then
    raise exception 'Published proposal versions are immutable';
  end if;

  if tg_op = 'DELETE' then
    return old;
  end if;

  return new;
end;
$$;
