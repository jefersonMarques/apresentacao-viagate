package contracts

import (
	"bytes"
	"context"
	"fmt"
	"net"

	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/legaltext"
)

// ConfirmAndSign validates the current OTP challenge and records the signature in
// one serializable transaction. A signer can never be left in a verified-but-not-
// signed state because of a failure between two independent commits.
func (s *Store) ConfirmAndSign(ctx context.Context, signerID string, otpHash, documentHash []byte, sessionID string, ip net.IP, userAgent string) (string, bool, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback(ctx)

	var challengeID, contractID, signerStatus string
	var expectedOTP, storedDocumentHash []byte
	var attempts int
	err = tx.QueryRow(ctx, `
		select ch.id::text,ch.otp_hash,ch.attempts,
		       s.contract_id::text,s.status::text,c.document_sha256
		from signature_challenges ch
		join contract_signers s on s.id=ch.contract_signer_id
		join contracts c on c.id=s.contract_id
		where ch.contract_signer_id=$1
		  and ch.verified_at is null
		  and ch.expires_at>now()
		  and c.status in ('generated','sent','partially_signed')
		order by ch.created_at desc
		limit 1
		for update of ch,s,c
	`, signerID).Scan(&challengeID, &expectedOTP, &attempts, &contractID, &signerStatus, &storedDocumentHash)
	if err != nil {
		return "", false, err
	}
	if signerStatus == "signed" {
		return contractID, false, fmt.Errorf("contract signer already signed")
	}
	if attempts >= 5 {
		return "", false, fmt.Errorf("maximum OTP attempts reached")
	}
	if !bytes.Equal(expectedOTP, otpHash) {
		if _, err := tx.Exec(ctx, `update signature_challenges set attempts=attempts+1 where id=$1`, challengeID); err != nil {
			return "", false, err
		}
		if err := tx.Commit(ctx); err != nil {
			return "", false, err
		}
		return "", false, fmt.Errorf("invalid OTP")
	}
	if !bytes.Equal(storedDocumentHash, documentHash) {
		return "", false, fmt.Errorf("contract document hash changed")
	}

	if _, err := tx.Exec(ctx, `update signature_challenges set verified_at=now() where id=$1`, challengeID); err != nil {
		return "", false, err
	}

	identityCommand, err := tx.Exec(ctx, `
		update identity_verifications
		set status='verified',verified_at=now(),
		    evidence=evidence||jsonb_build_object(
		        'session_id',$2::uuid,
		        'ip',$3::inet,
		        'user_agent',$4::text
		    )
		where id=(
			select id from identity_verifications
			where contract_signer_id=$1 and mode='email_otp' and status='pending'
			order by created_at desc limit 1
		)
	`, signerID, sessionID, nullableIP(ip), userAgent)
	if err != nil {
		return "", false, fmt.Errorf("record OTP identity verification: %w", err)
	}
	if identityCommand.RowsAffected() != 1 {
		return "", false, fmt.Errorf("pending OTP identity verification not found")
	}

	if _, err := tx.Exec(ctx, `
		insert into signature_events(contract_id,contract_signer_id,event_type,document_hash,ip_address,user_agent,session_id,metadata)
		values($1,$2,'otp.verified',$3,$4,$5,$6,jsonb_build_object('channel','email'))
	`, contractID, signerID, storedDocumentHash, nullableIP(ip), userAgent, sessionID); err != nil {
		return "", false, fmt.Errorf("record OTP verification event: %w", err)
	}

	consentHash := legaltext.SHA256(legaltext.SignatureConsentText)
	if _, err := tx.Exec(ctx, `
		update contract_signers
		set status='signed',signed_at=now(),signed_document_hash=$2,signature_session_id=$3,
		    signature_consent_version=$4,signature_consent_text=$5,signature_consent_sha256=$6
		where id=$1
	`, signerID, storedDocumentHash, sessionID, legaltext.SignatureConsentVersion, legaltext.SignatureConsentText, consentHash); err != nil {
		return "", false, err
	}

	if _, err := tx.Exec(ctx, `
		insert into signature_events(contract_id,contract_signer_id,event_type,document_hash,ip_address,user_agent,session_id,metadata)
		values($1,$2,'contract.signed',$3,$4,$5,$6,
		       jsonb_build_object('consent_version',$7::text,'consent_sha256',$8::text))
	`, contractID, signerID, storedDocumentHash, nullableIP(ip), userAgent, sessionID, legaltext.SignatureConsentVersion, fmt.Sprintf("%x", consentHash)); err != nil {
		return "", false, fmt.Errorf("record contract signature event: %w", err)
	}

	var pending int
	if err := tx.QueryRow(ctx, `select count(*) from contract_signers where contract_id=$1 and status<>'signed'`, contractID).Scan(&pending); err != nil {
		return "", false, err
	}
	fullySigned := pending == 0
	if fullySigned {
		if _, err := tx.Exec(ctx, `update contracts set status='signed',fully_signed_at=now(),updated_at=now() where id=$1`, contractID); err != nil {
			return "", false, err
		}
		if _, err := tx.Exec(ctx, `
			insert into contract_finalization_jobs(contract_id,status,available_at,updated_at)
			values($1,'pending',now(),now())
			on conflict (contract_id) do update
			set status=case when contract_finalization_jobs.status='completed' then 'completed' else 'pending' end,
			    available_at=case when contract_finalization_jobs.status='completed' then contract_finalization_jobs.available_at else now() end,
			    processing_at=null,
			    last_error=case when contract_finalization_jobs.status='completed' then contract_finalization_jobs.last_error else null end,
			    updated_at=now()
		`, contractID); err != nil {
			return "", false, fmt.Errorf("queue contract evidence finalization: %w", err)
		}
	} else {
		if _, err := tx.Exec(ctx, `update contracts set status='partially_signed',updated_at=now() where id=$1`, contractID); err != nil {
			return "", false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", false, err
	}
	return contractID, fullySigned, nil
}
