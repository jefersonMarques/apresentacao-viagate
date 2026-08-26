package auth

import (
	"context"
	"net"
)

const (
	loginRatePairLimit = 8
	loginRateIPLimit   = 30
)

func (s *Store) LoginAllowed(ctx context.Context,email string,ip net.IP)(bool,error){
	var pairFailures,ipFailures int
	err:=s.pool.QueryRow(ctx,`
		select
		  count(*) filter(where email=$1 and ip_address is not distinct from $2::inet),
		  count(*) filter(where ip_address is not distinct from $2::inet)
		from login_failures
		where created_at > now()-interval '15 minutes'
	`,email,nullableIP(ip)).Scan(&pairFailures,&ipFailures)
	if err!=nil{return false,err}
	return pairFailures<loginRatePairLimit&&ipFailures<loginRateIPLimit,nil
}

func (s *Store) RecordLoginFailure(ctx context.Context,email string,ip net.IP) error {
	_,err:=s.pool.Exec(ctx,`insert into login_failures(email,ip_address) values($1,$2)`,email,nullableIP(ip))
	return err
}

func (s *Store) ClearLoginFailures(ctx context.Context,email string,ip net.IP) error {
	_,err:=s.pool.Exec(ctx,`
		delete from login_failures
		where email=$1 and ip_address is not distinct from $2::inet
	`,email,nullableIP(ip))
	return err
}
