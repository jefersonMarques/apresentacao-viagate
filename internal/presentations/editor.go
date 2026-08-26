package presentations

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type EditorInput struct {
	PresentationID     string
	ClientLegalName    string
	ClientCNPJ         string
	ClientContactName  string
	ClientContactRole  string
	Title              string
	ShowClientIdentity bool
	ShowContact        bool
	ContentHash        []byte
}

type SavedDraft struct {
	PresentationID string
	VersionID      string
	VersionNumber  int
	PublicToken    string
}

type PublicPresentation struct {
	PresentationID string
	VersionID      string
	VersionNumber  int
	PublicToken    string
	Title          string
	ClientName     string
	ContactName    string
	ContactRole    string
	ShowClient     bool
	ShowContact    bool
	SalespersonName string
	SalespersonEmail string
}

func (s *Store) SaveDraft(ctx context.Context,userID string,allowAll bool,input EditorInput)(SavedDraft,error){
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{IsoLevel:pgx.Serializable});if err!=nil{return SavedDraft{},err};defer tx.Rollback(ctx)
	var clientID *string
	if input.ClientLegalName!=""{
		var id string
		if input.ClientCNPJ!=""{_ = tx.QueryRow(ctx,`select id::text from clients where cnpj=$1`,input.ClientCNPJ).Scan(&id)}
		if id==""{if err:=tx.QueryRow(ctx,`insert into clients(legal_name,cnpj,created_by) values($1,nullif($2,''),$3) returning id::text`,input.ClientLegalName,input.ClientCNPJ,userID).Scan(&id);err!=nil{return SavedDraft{},err}}
		clientID=&id
	}
	if input.PresentationID==""{
		if err:=tx.QueryRow(ctx,`insert into presentations(client_id,title,status,created_by) values($1,$2,'draft',$3) returning id::text`,clientID,input.Title,userID).Scan(&input.PresentationID);err!=nil{return SavedDraft{},err}
	}else{
		var owner string
		if err:=tx.QueryRow(ctx,`select created_by::text from presentations where id=$1 for update`,input.PresentationID).Scan(&owner);err!=nil{return SavedDraft{},err}
		if !allowAll&&owner!=userID{return SavedDraft{},fmt.Errorf("presentation access denied")}
		if _,err:=tx.Exec(ctx,`update presentations set client_id=$2,title=$3,updated_at=now() where id=$1`,input.PresentationID,clientID,input.Title);err!=nil{return SavedDraft{},err}
	}
	var salespersonName,salespersonEmail string
	if err:=tx.QueryRow(ctx,`select name,email::text from users where id=$1`,userID).Scan(&salespersonName,&salespersonEmail);err!=nil{return SavedDraft{},err}
	content:=map[string]any{"client":map[string]any{"legal_name":input.ClientLegalName,"cnpj":input.ClientCNPJ,"contact_name":input.ClientContactName,"contact_role":input.ClientContactRole},"settings":map[string]any{"show_client_identity":input.ShowClientIdentity,"show_contact":input.ShowContact},"salesperson":map[string]any{"name":salespersonName,"email":salespersonEmail}}
	contentJSON,_:=json.Marshal(content)
	var draft SavedDraft
	err=tx.QueryRow(ctx,`select id::text,version_number,public_token::text from presentation_versions where presentation_id=$1 and published_at is null order by version_number desc limit 1 for update`,input.PresentationID).Scan(&draft.VersionID,&draft.VersionNumber,&draft.PublicToken)
	if err==pgx.ErrNoRows{
		if err:=tx.QueryRow(ctx,`select coalesce(max(version_number),0)+1 from presentation_versions where presentation_id=$1`,input.PresentationID).Scan(&draft.VersionNumber);err!=nil{return SavedDraft{},err}
		if err:=tx.QueryRow(ctx,`insert into presentation_versions(presentation_id,version_number,content,content_hash,created_by) values($1,$2,$3,$4,$5) returning id::text,public_token::text`,input.PresentationID,draft.VersionNumber,contentJSON,input.ContentHash,userID).Scan(&draft.VersionID,&draft.PublicToken);err!=nil{return SavedDraft{},err}
	}else if err!=nil{return SavedDraft{},err}else{if _,err:=tx.Exec(ctx,`update presentation_versions set content=$2,content_hash=$3 where id=$1 and published_at is null`,draft.VersionID,contentJSON,input.ContentHash);err!=nil{return SavedDraft{},err}}
	draft.PresentationID=input.PresentationID
	if err:=tx.Commit(ctx);err!=nil{return SavedDraft{},err};return draft,nil
}

func (s *Store) Publish(ctx context.Context,userID string,allowAll bool,versionID string)(string,error){
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{IsoLevel:pgx.Serializable});if err!=nil{return "",err};defer tx.Rollback(ctx)
	var presentationID,owner,token string;var version int
	if err:=tx.QueryRow(ctx,`select v.presentation_id::text,p.created_by::text,v.version_number,v.public_token::text from presentation_versions v join presentations p on p.id=v.presentation_id where v.id=$1 and v.published_at is null for update of v,p`,versionID).Scan(&presentationID,&owner,&version,&token);err!=nil{return "",err}
	if !allowAll&&owner!=userID{return "",fmt.Errorf("presentation access denied")}
	if _,err:=tx.Exec(ctx,`update presentation_versions set published_at=now() where id=$1`,versionID);err!=nil{return "",err}
	if _,err:=tx.Exec(ctx,`update presentations set status='published',current_version=$2,updated_at=now() where id=$1`,presentationID,version);err!=nil{return "",err}
	if _,err:=tx.Exec(ctx,`insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata) values($1,'presentation.published','presentation',$2,jsonb_build_object('version',$3))`,userID,presentationID,version);err!=nil{return "",err}
	if err:=tx.Commit(ctx);err!=nil{return "",err};return token,nil
}

func (s *Store) EditorByID(ctx context.Context,userID,id string,allowAll bool)(EditorInput,SavedDraft,error){
	var input EditorInput;var draft SavedDraft;var owner string
	var contentJSON []byte
	err:=s.pool.QueryRow(ctx,`select p.id::text,p.title,p.created_by::text,coalesce(v.id::text,''),coalesce(v.version_number,0),coalesce(v.public_token::text,''),coalesce(v.content,'{}'::jsonb),coalesce(v.content_hash,''::bytea) from presentations p left join lateral(select * from presentation_versions where presentation_id=p.id order by version_number desc limit 1)v on true where p.id=$1`,id).Scan(&input.PresentationID,&input.Title,&owner,&draft.VersionID,&draft.VersionNumber,&draft.PublicToken,&contentJSON,&input.ContentHash)
	if err!=nil{return EditorInput{},SavedDraft{},err};if !allowAll&&owner!=userID{return EditorInput{},SavedDraft{},fmt.Errorf("presentation access denied")}
	var content struct{Client struct{LegalName string `json:"legal_name"`;CNPJ string `json:"cnpj"`;ContactName string `json:"contact_name"`;ContactRole string `json:"contact_role"`} `json:"client"`;Settings struct{ShowClient bool `json:"show_client_identity"`;ShowContact bool `json:"show_contact"`} `json:"settings"`}
	_ = json.Unmarshal(contentJSON,&content);input.ClientLegalName=content.Client.LegalName;input.ClientCNPJ=content.Client.CNPJ;input.ClientContactName=content.Client.ContactName;input.ClientContactRole=content.Client.ContactRole;input.ShowClientIdentity=content.Settings.ShowClient;input.ShowContact=content.Settings.ShowContact;draft.PresentationID=id
	return input,draft,nil
}

func (s *Store) PublicByToken(ctx context.Context,token string)(PublicPresentation,error){
	var result PublicPresentation;var contentJSON []byte
	err:=s.pool.QueryRow(ctx,`select p.id::text,v.id::text,v.version_number,v.public_token::text,p.title,v.content from presentation_versions v join presentations p on p.id=v.presentation_id where v.public_token=$1 and v.published_at is not null and p.status='published'`,token).Scan(&result.PresentationID,&result.VersionID,&result.VersionNumber,&result.PublicToken,&result.Title,&contentJSON);if err!=nil{return PublicPresentation{},err}
	var content struct{Client struct{LegalName string `json:"legal_name"`;ContactName string `json:"contact_name"`;ContactRole string `json:"contact_role"`} `json:"client"`;Settings struct{ShowClient bool `json:"show_client_identity"`;ShowContact bool `json:"show_contact"`} `json:"settings"`;Salesperson struct{Name string `json:"name"`;Email string `json:"email"`} `json:"salesperson"`};if err:=json.Unmarshal(contentJSON,&content);err!=nil{return PublicPresentation{},err}
	result.ClientName=content.Client.LegalName;result.ContactName=content.Client.ContactName;result.ContactRole=content.Client.ContactRole;result.ShowClient=content.Settings.ShowClient;result.ShowContact=content.Settings.ShowContact;result.SalespersonName=content.Salesperson.Name;result.SalespersonEmail=content.Salesperson.Email;return result,nil
}
