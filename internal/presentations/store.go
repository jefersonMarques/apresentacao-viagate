package presentations

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct { pool *pgxpool.Pool }

type Presentation struct {
	ID             string
	ClientName     string
	Title          string
	Status         string
	CurrentVersion int
	PublicToken    string
	CreatedBy      string
	CreatedByName  string
	UpdatedAt      time.Time
}

type EditorInput struct {
	PresentationID       string
	ClientLegalName      string
	ClientTradeName      string
	ClientCNPJ           string
	Title                string
	ContactName          string
	ContactRole          string
	ContactEmail         string
	ShowClientIdentity   bool
	ShowContactSlide     bool
	SelectedModules      []string
	SalespersonName      string
	SalespersonEmail     string
	SalespersonPhone     string
	SalespersonJobTitle  string
	SalespersonPhotoURL  string
	SalespersonLinkedIn  string
	SalespersonInstagram string
	ContentHash          []byte
}

type SavedDraft struct {
	PresentationID string
	VersionID      string
	VersionNumber  int
	PublicToken    string
}

type PublicPresentation struct {
	PresentationID       string
	VersionID            string
	VersionNumber        int
	PublicToken          string
	Title                string
	ClientLegalName      string
	ClientTradeName      string
	ContactName          string
	ContactRole          string
	ContactEmail         string
	SalespersonName      string
	SalespersonEmail     string
	SalespersonPhone     string
	SalespersonJobTitle  string
	SalespersonPhotoURL  string
	SalespersonLinkedIn  string
	SalespersonInstagram string
	ShowClientIdentity   bool
	ShowContactSlide     bool
	SelectedModules      []string
	ContentHash          []byte
}

type presentationContent struct {
	Title string `json:"title"`
	Client struct {
		LegalName string `json:"legal_name"`
		TradeName string `json:"trade_name"`
		CNPJ      string `json:"cnpj"`
	} `json:"client"`
	Contact struct {
		Name  string `json:"name"`
		Role  string `json:"role"`
		Email string `json:"email"`
	} `json:"contact"`
	Salesperson struct {
		Name      string `json:"name"`
		Email     string `json:"email"`
		Phone     string `json:"phone"`
		JobTitle  string `json:"job_title"`
		PhotoURL  string `json:"photo_url"`
		LinkedIn  string `json:"linkedin"`
		Instagram string `json:"instagram"`
	} `json:"salesperson"`
	Settings struct {
		ShowClientIdentity bool     `json:"show_client_identity"`
		ShowContactSlide   bool     `json:"show_contact_slide"`
		SelectedModules    []string `json:"selected_modules"`
	} `json:"settings"`
}

func NewStore(pool *pgxpool.Pool)*Store{return &Store{pool:pool}}

func (s *Store) List(ctx context.Context,userID string,all bool)([]Presentation,error){
	query:=`
		select p.id::text,coalesce(c.legal_name,''),p.title,p.status::text,p.current_version,
		       coalesce(v.public_token::text,''),p.created_by::text,u.name,p.updated_at
		from presentations p
		join users u on u.id=p.created_by
		left join clients c on c.id=p.client_id
		left join presentation_versions v on v.presentation_id=p.id and v.version_number=p.current_version and v.published_at is not null
	`
	args:=[]any{}
	if !all {query+=` where p.created_by=$1`;args=append(args,userID)}
	query+=` order by p.updated_at desc`
	rows,err:=s.pool.Query(ctx,query,args...);if err!=nil{return nil,err};defer rows.Close()
	var items []Presentation
	for rows.Next(){var item Presentation;if err:=rows.Scan(&item.ID,&item.ClientName,&item.Title,&item.Status,&item.CurrentVersion,&item.PublicToken,&item.CreatedBy,&item.CreatedByName,&item.UpdatedAt);err!=nil{return nil,err};items=append(items,item)}
	return items,rows.Err()
}

func (s *Store) SaveDraft(ctx context.Context,userID string,allowAll bool,input EditorInput)(SavedDraft,error){
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{IsoLevel:pgx.Serializable});if err!=nil{return SavedDraft{},err};defer tx.Rollback(ctx)
	clientID:=""

	if input.PresentationID==""{
		if input.ClientCNPJ!=""{_ = tx.QueryRow(ctx,`select id::text from clients where cnpj=$1 and created_by=$2`,input.ClientCNPJ,userID).Scan(&clientID)}
		if clientID=="" && input.ClientLegalName!=""{
			if err:=tx.QueryRow(ctx,`
				insert into clients(legal_name,trade_name,cnpj,created_by)
				values($1,nullif($2,''),nullif($3,''),$4) returning id::text
			`,input.ClientLegalName,input.ClientTradeName,input.ClientCNPJ,userID).Scan(&clientID);err!=nil{return SavedDraft{},err}
		}else if clientID!=""&&input.ClientLegalName!=""{
			if _,err:=tx.Exec(ctx,`
				update clients set legal_name=$2,trade_name=nullif($3,''),updated_at=now()
				where id=$1 and created_by=$4
			`,clientID,input.ClientLegalName,input.ClientTradeName,userID);err!=nil{return SavedDraft{},err}
		}
		if err:=tx.QueryRow(ctx,`
			insert into presentations(client_id,title,status,created_by)
			values(nullif($1,'')::uuid,$2,'draft',$3) returning id::text
		`,clientID,input.Title,userID).Scan(&input.PresentationID);err!=nil{return SavedDraft{},err}
	}else{
		var owner,status string
		if err:=tx.QueryRow(ctx,`select coalesce(client_id::text,''),created_by::text,status::text from presentations where id=$1 for update`,input.PresentationID).Scan(&clientID,&owner,&status);err!=nil{return SavedDraft{},err}
		if !allowAll&&owner!=userID{return SavedDraft{},fmt.Errorf("presentation access denied")}
		if status=="accepted"||status=="cancelled"{return SavedDraft{},fmt.Errorf("presentation cannot be changed in its current state")}
		if clientID!=""&&input.ClientLegalName!=""{if _,err:=tx.Exec(ctx,`update clients set legal_name=$2,trade_name=nullif($3,''),cnpj=nullif($4,''),updated_at=now() where id=$1`,clientID,input.ClientLegalName,input.ClientTradeName,input.ClientCNPJ);err!=nil{return SavedDraft{},err}}
		if _,err:=tx.Exec(ctx,`update presentations set title=$2,updated_at=now() where id=$1`,input.PresentationID,input.Title);err!=nil{return SavedDraft{},err}
	}

	content:=presentationContent{Title:input.Title}
	content.Client.LegalName=input.ClientLegalName;content.Client.TradeName=input.ClientTradeName;content.Client.CNPJ=input.ClientCNPJ
	content.Contact.Name=input.ContactName;content.Contact.Role=input.ContactRole;content.Contact.Email=input.ContactEmail
	content.Salesperson.Name=input.SalespersonName;content.Salesperson.Email=input.SalespersonEmail;content.Salesperson.Phone=input.SalespersonPhone;content.Salesperson.JobTitle=input.SalespersonJobTitle
	content.Salesperson.PhotoURL=input.SalespersonPhotoURL;content.Salesperson.LinkedIn=input.SalespersonLinkedIn;content.Salesperson.Instagram=input.SalespersonInstagram
	content.Settings.ShowClientIdentity=input.ShowClientIdentity;content.Settings.ShowContactSlide=input.ShowContactSlide;content.Settings.SelectedModules=input.SelectedModules
	contentJSON,err:=json.Marshal(content);if err!=nil{return SavedDraft{},err}

	var draft SavedDraft
	err=tx.QueryRow(ctx,`
		select id::text,version_number,public_token::text from presentation_versions
		where presentation_id=$1 and published_at is null order by version_number desc limit 1 for update
	`,input.PresentationID).Scan(&draft.VersionID,&draft.VersionNumber,&draft.PublicToken)
	if err==pgx.ErrNoRows{
		if err:=tx.QueryRow(ctx,`select coalesce(max(version_number),0)+1 from presentation_versions where presentation_id=$1`,input.PresentationID).Scan(&draft.VersionNumber);err!=nil{return SavedDraft{},err}
		if err:=tx.QueryRow(ctx,`
			insert into presentation_versions(presentation_id,version_number,content,content_hash,created_by)
			values($1,$2,$3,$4,$5) returning id::text,public_token::text
		`,input.PresentationID,draft.VersionNumber,contentJSON,input.ContentHash,userID).Scan(&draft.VersionID,&draft.PublicToken);err!=nil{return SavedDraft{},err}
	}else if err!=nil{return SavedDraft{},err}else{
		if _,err:=tx.Exec(ctx,`update presentation_versions set content=$2,content_hash=$3 where id=$1 and published_at is null`,draft.VersionID,contentJSON,input.ContentHash);err!=nil{return SavedDraft{},err}
	}
	draft.PresentationID=input.PresentationID
	if err:=tx.Commit(ctx);err!=nil{return SavedDraft{},err}
	return draft,nil
}

func (s *Store) Publish(ctx context.Context,userID string,allowAll bool,versionID string)(string,error){
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{IsoLevel:pgx.Serializable});if err!=nil{return "",err};defer tx.Rollback(ctx)
	var presentationID,owner,token string;var versionNumber int
	if err:=tx.QueryRow(ctx,`
		select v.presentation_id::text,p.created_by::text,v.version_number,v.public_token::text
		from presentation_versions v join presentations p on p.id=v.presentation_id
		where v.id=$1 and v.published_at is null for update of v,p
	`,versionID).Scan(&presentationID,&owner,&versionNumber,&token);err!=nil{return "",err}
	if !allowAll&&owner!=userID{return "",fmt.Errorf("presentation access denied")}
	if _,err:=tx.Exec(ctx,`update presentation_versions set published_at=now() where id=$1`,versionID);err!=nil{return "",err}
	if _,err:=tx.Exec(ctx,`update presentations set status='published',current_version=$2,updated_at=now() where id=$1`,presentationID,versionNumber);err!=nil{return "",err}
	if _,err:=tx.Exec(ctx,`insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata) values($1,'presentation.published','presentation',$2,jsonb_build_object('version',$3::integer,'version_id',$4::uuid))`,userID,presentationID,versionNumber,versionID);err!=nil{return "",err}
	if err:=tx.Commit(ctx);err!=nil{return "",err};return token,nil
}

func (s *Store) EditorByID(ctx context.Context,userID,presentationID string,allowAll bool)(EditorInput,SavedDraft,error){
	var input EditorInput;var draft SavedDraft;var owner string
	if err:=s.pool.QueryRow(ctx,`
		select p.id::text,p.title,p.created_by::text,coalesce(c.legal_name,''),coalesce(c.trade_name,''),coalesce(c.cnpj,'')
		from presentations p left join clients c on c.id=p.client_id where p.id=$1
	`,presentationID).Scan(&input.PresentationID,&input.Title,&owner,&input.ClientLegalName,&input.ClientTradeName,&input.ClientCNPJ);err!=nil{return EditorInput{},SavedDraft{},err}
	if !allowAll&&owner!=userID{return EditorInput{},SavedDraft{},fmt.Errorf("presentation access denied")}
	var contentJSON []byte
	err:=s.pool.QueryRow(ctx,`select id::text,version_number,public_token::text,content,content_hash from presentation_versions where presentation_id=$1 order by version_number desc limit 1`,presentationID).Scan(&draft.VersionID,&draft.VersionNumber,&draft.PublicToken,&contentJSON,&input.ContentHash)
	if err!=nil&&err!=pgx.ErrNoRows{return EditorInput{},SavedDraft{},err}
	draft.PresentationID=presentationID
	if len(contentJSON)>0{
		var content presentationContent
		if json.Unmarshal(contentJSON,&content)==nil{
			if strings.TrimSpace(content.Title)!=""{input.Title=content.Title}
			input.ContactName=content.Contact.Name;input.ContactRole=content.Contact.Role;input.ContactEmail=content.Contact.Email
			input.ShowClientIdentity=content.Settings.ShowClientIdentity;input.ShowContactSlide=content.Settings.ShowContactSlide;input.SelectedModules=content.Settings.SelectedModules
			input.SalespersonName=content.Salesperson.Name;input.SalespersonEmail=content.Salesperson.Email;input.SalespersonPhone=content.Salesperson.Phone;input.SalespersonJobTitle=content.Salesperson.JobTitle
			input.SalespersonPhotoURL=content.Salesperson.PhotoURL;input.SalespersonLinkedIn=content.Salesperson.LinkedIn;input.SalespersonInstagram=content.Salesperson.Instagram
		}
	}
	return input,draft,nil
}

func (s *Store) PublicByToken(ctx context.Context,token string)(PublicPresentation,error){
	var result PublicPresentation;var contentJSON []byte;var currentTitle string
	err:=s.pool.QueryRow(ctx,`
		select p.id::text,v.id::text,v.version_number,v.public_token::text,p.title,v.content,v.content_hash
		from presentation_versions v join presentations p on p.id=v.presentation_id
		where v.public_token=$1 and v.published_at is not null and p.status='published'
	`,token).Scan(&result.PresentationID,&result.VersionID,&result.VersionNumber,&result.PublicToken,&currentTitle,&contentJSON,&result.ContentHash)
	if err!=nil{return PublicPresentation{},err}
	var content presentationContent;if err:=json.Unmarshal(contentJSON,&content);err!=nil{return PublicPresentation{},err}
	result.Title=currentTitle;if strings.TrimSpace(content.Title)!=""{result.Title=content.Title}
	result.ClientLegalName=content.Client.LegalName;result.ClientTradeName=content.Client.TradeName
	result.ContactName=content.Contact.Name;result.ContactRole=content.Contact.Role;result.ContactEmail=content.Contact.Email
	result.SalespersonName=content.Salesperson.Name;result.SalespersonEmail=content.Salesperson.Email;result.SalespersonPhone=content.Salesperson.Phone;result.SalespersonJobTitle=content.Salesperson.JobTitle
	result.SalespersonPhotoURL=content.Salesperson.PhotoURL;result.SalespersonLinkedIn=content.Salesperson.LinkedIn;result.SalespersonInstagram=content.Salesperson.Instagram
	result.ShowClientIdentity=content.Settings.ShowClientIdentity;result.ShowContactSlide=content.Settings.ShowContactSlide;result.SelectedModules=content.Settings.SelectedModules
	return result,nil
}
