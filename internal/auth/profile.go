package auth

import (
	"context"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

func (s *Store) Profile(ctx context.Context, userID string) (domain.User, error) {
	var user domain.User
	var roles []string
	err := s.pool.QueryRow(ctx, `
		select u.id::text,u.email::text,u.name,coalesce(u.phone,''),coalesce(u.job_title,''),
		       coalesce(u.photo_url,''),coalesce(u.linkedin_url,''),coalesce(u.instagram_url,''),
		       u.status::text,u.created_at,
		       coalesce(array_agg(r.code order by r.code) filter (where r.code is not null), '{}')
		from users u
		left join user_roles ur on ur.user_id=u.id
		left join roles r on r.id=ur.role_id
		where u.id=$1
		group by u.id
	`, userID).Scan(
		&user.ID,&user.Email,&user.Name,&user.Phone,&user.JobTitle,
		&user.PhotoURL,&user.LinkedInURL,&user.InstagramURL,
		&user.Status,&user.CreatedAt,&roles,
	)
	if err != nil { return domain.User{}, err }
	user.Roles=roles
	return user,nil
}

func (s *Store) UpdateProfile(ctx context.Context,userID,name,email,phone,jobTitle,photoURL,linkedInURL,instagramURL string) error {
	_,err:=s.pool.Exec(ctx,`
		update users
		set name=$2,email=$3,phone=nullif($4,''),job_title=nullif($5,''),
		    photo_url=nullif($6,''),linkedin_url=nullif($7,''),instagram_url=nullif($8,''),updated_at=now()
		where id=$1 and status='active'
	`,userID,name,email,phone,jobTitle,photoURL,linkedInURL,instagramURL)
	return err
}
