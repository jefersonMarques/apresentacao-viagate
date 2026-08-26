package onboarding

import "context"

func (s *Store) DocumentByID(ctx context.Context,onboardingID,documentID string)(Document,error){
	var document Document
	err:=s.pool.QueryRow(ctx,`
		select id::text,document_type,storage_key,original_filename,mime_type,size_bytes,sha256
		from uploaded_documents
		where id=$1 and onboarding_id=$2
	`,documentID,onboardingID).Scan(&document.ID,&document.DocumentType,&document.StorageKey,&document.OriginalFilename,&document.MIMEType,&document.SizeBytes,&document.SHA256)
	return document,err
}
