create or replace function public.prevent_published_version_mutation()
returns trigger
language plpgsql
as $$
begin
  if tg_op = 'DELETE'
    and current_setting('app.allow_published_version_delete', true) = 'on'
  then
    return old;
  end if;

  if old.published_at is not null then
    raise exception 'Published proposal versions are immutable';
  end if;

  if tg_op = 'DELETE' then
    return old;
  end if;

  return new;
end;
$$;

create or replace function public.delete_presentation(target_presentation_id uuid)
returns boolean
language plpgsql
security definer
set search_path = public
as $$
declare
  deleted_count integer;
begin
  if auth.uid() is null then
    raise exception 'Authentication required';
  end if;

  if not exists (
    select 1
    from public.presentations presentation
    where presentation.id = target_presentation_id
      and presentation.created_by = auth.uid()
  ) then
    raise exception 'Presentation not found';
  end if;

  perform set_config('app.allow_published_version_delete', 'on', true);

  delete from public.presentations
  where id = target_presentation_id
    and created_by = auth.uid();

  get diagnostics deleted_count = row_count;
  return deleted_count = 1;
end;
$$;

revoke all on function public.delete_presentation(uuid) from public;
grant execute on function public.delete_presentation(uuid) to authenticated;
