package contracts

import "context"

func (s *Store) SetDefaultTemplate(ctx context.Context, templateID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil { return err }
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `update contract_templates set is_default=false where is_default=true`); err != nil { return err }
	if _, err := tx.Exec(ctx, `update contract_templates set is_default=true where id=$1 and is_active=true`, templateID); err != nil { return err }
	return tx.Commit(ctx)
}

func (s *Store) TemplateIsDefault(ctx context.Context, templateID string) (bool,error) {
	var value bool
	err := s.pool.QueryRow(ctx, `select is_default from contract_templates where id=$1`,templateID).Scan(&value)
	return value,err
}
