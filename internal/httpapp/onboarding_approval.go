package httpapp

import (
	"context"
	"fmt"
)

func (a *App) autoApproveOnboarding(ctx context.Context, onboardingID, source string) error {
	command, err := a.pool.Exec(ctx, `
		update onboardings
		set status='approved',
		    approved_at=coalesce(approved_at,now()),
		    reviewed_at=now(),
		    review_notes='Aprovação automática após validação dos dados da contratação.',
		    updated_at=now()
		where id=$1 and status='submitted'
	`, onboardingID)
	if err != nil {
		return fmt.Errorf("auto approve onboarding: %w", err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("onboarding is not submitted")
	}

	if _, err := a.pool.Exec(ctx, `
		insert into audit_events(actor_type,event_type,resource_type,resource_id,metadata)
		values(
			'system',
			'onboarding.auto_approved',
			'onboarding',
			$1,
			jsonb_build_object('source',$2::text)
		)
	`, onboardingID, source); err != nil {
		a.logger.Error("record onboarding auto approval audit failed", "onboarding_id", onboardingID, "source", source, "error", err)
	}

	return nil
}
