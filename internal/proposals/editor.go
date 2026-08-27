package proposals

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type EditorItem struct {
	CatalogID  string
	GroupName  string
	Label      string
	Unit       string
	Price      float64
	IsOptional bool
	SortOrder  int
}

type EditorInput struct {
	ProposalID          string
	ClientLegalName     string
	ClientTradeName     string
	ClientCNPJ          string
	ClientEmail         string
	ClientPhone         string
	ClientLogoURL       string
	ClientStreet        string
	ClientStreetNumber  string
	ClientComplement    string
	ClientDistrict      string
	ClientCity          string
	ClientState         string
	ClientPostalCode    string
	ContactName         string
	ContactRole         string
	ContactEmail        string
	ContactPhone        string
	Title               string
	ValidUntil          *time.Time
	OperationContext    string
	CustomerPriorities  []string
	SolutionTitle       string
	SolutionScope       []string
	PricingModel        string
	MinimumInvoice      float64
	SetupFee            float64
	Conditions          []string
	Items               []EditorItem
	Content             map[string]any
	ContentHash         []byte
}

type SavedDraft struct {
	ProposalID    string
	VersionID     string
	VersionNumber int
	PublicToken   string
}

func (s *Store) SaveDraft(ctx context.Context, userID string, allowAll bool, input EditorInput) (SavedDraft, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return SavedDraft{}, err
	}
	defer tx.Rollback(ctx)

	clientID := ""
	if input.ProposalID == "" {
		if input.ClientCNPJ != "" {
			_ = tx.QueryRow(ctx, `select id::text from clients where cnpj=$1 and created_by=$2`, input.ClientCNPJ, userID).Scan(&clientID)
		}
		if clientID == "" {
			if err := tx.QueryRow(ctx, `
				insert into clients(
					legal_name,trade_name,cnpj,email,phone,street,street_number,complement,district,city,state,postal_code,created_by
				) values(
					nullif($1,''),nullif($2,''),nullif($3,''),nullif($4,'')::citext,nullif($5,''),nullif($6,''),nullif($7,''),nullif($8,''),nullif($9,''),nullif($10,''),nullif($11,''),nullif($12,''),$13
				) returning id::text
			`,
				input.ClientLegalName,
				input.ClientTradeName,
				input.ClientCNPJ,
				input.ClientEmail,
				input.ClientPhone,
				input.ClientStreet,
				input.ClientStreetNumber,
				input.ClientComplement,
				input.ClientDistrict,
				input.ClientCity,
				input.ClientState,
				input.ClientPostalCode,
				userID,
			).Scan(&clientID); err != nil {
				return SavedDraft{}, fmt.Errorf("create client: %w", err)
			}
		} else {
			if _, err := tx.Exec(ctx, `
				update clients set
					legal_name=nullif($2,''),trade_name=nullif($3,''),email=nullif($4,'')::citext,phone=nullif($5,''),
					street=nullif($6,''),street_number=nullif($7,''),complement=nullif($8,''),district=nullif($9,''),
					city=nullif($10,''),state=nullif($11,''),postal_code=nullif($12,''),updated_at=now()
				where id=$1 and created_by=$13
			`,
				clientID,
				input.ClientLegalName,
				input.ClientTradeName,
				input.ClientEmail,
				input.ClientPhone,
				input.ClientStreet,
				input.ClientStreetNumber,
				input.ClientComplement,
				input.ClientDistrict,
				input.ClientCity,
				input.ClientState,
				input.ClientPostalCode,
				userID,
			); err != nil {
				return SavedDraft{}, fmt.Errorf("update client: %w", err)
			}
		}
		if err := tx.QueryRow(ctx, `
			insert into proposals(client_id,title,status,valid_until,created_by)
			values($1,$2,'draft',$3,$4) returning id::text
		`, clientID, input.Title, input.ValidUntil, userID).Scan(&input.ProposalID); err != nil {
			return SavedDraft{}, fmt.Errorf("create proposal: %w", err)
		}
	} else {
		var owner, status string
		if err := tx.QueryRow(ctx, `select client_id::text,created_by::text,status::text from proposals where id=$1 for update`, input.ProposalID).Scan(&clientID, &owner, &status); err != nil {
			return SavedDraft{}, err
		}
		if !allowAll && owner != userID {
			return SavedDraft{}, fmt.Errorf("proposal access denied")
		}
		if status == "accepted" || status == "cancelled" {
			return SavedDraft{}, fmt.Errorf("accepted or cancelled proposals cannot be changed")
		}
		if _, err := tx.Exec(ctx, `
			update clients set
				legal_name=nullif($2,''),trade_name=nullif($3,''),cnpj=nullif($4,''),email=nullif($5,'')::citext,phone=nullif($6,''),
				street=nullif($7,''),street_number=nullif($8,''),complement=nullif($9,''),district=nullif($10,''),
				city=nullif($11,''),state=nullif($12,''),postal_code=nullif($13,''),updated_at=now()
			where id=$1
		`,
			clientID,
			input.ClientLegalName,
			input.ClientTradeName,
			input.ClientCNPJ,
			input.ClientEmail,
			input.ClientPhone,
			input.ClientStreet,
			input.ClientStreetNumber,
			input.ClientComplement,
			input.ClientDistrict,
			input.ClientCity,
			input.ClientState,
			input.ClientPostalCode,
		); err != nil {
			return SavedDraft{}, err
		}
		if _, err := tx.Exec(ctx, `update proposals set title=$2,valid_until=$3,updated_at=now() where id=$1`, input.ProposalID, input.Title, input.ValidUntil); err != nil {
			return SavedDraft{}, err
		}
	}

	contentJSON, err := json.Marshal(input.Content)
	if err != nil {
		return SavedDraft{}, err
	}
	conditionsJSON, err := json.Marshal(input.Conditions)
	if err != nil {
		return SavedDraft{}, err
	}

	var draft SavedDraft
	err = tx.QueryRow(ctx, `
		select id::text,version_number,public_token::text
		from proposal_versions
		where proposal_id=$1 and published_at is null
		order by version_number desc limit 1 for update
	`, input.ProposalID).Scan(&draft.VersionID, &draft.VersionNumber, &draft.PublicToken)
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, `select coalesce(max(version_number),0)+1 from proposal_versions where proposal_id=$1`, input.ProposalID).Scan(&draft.VersionNumber)
		if err != nil {
			return SavedDraft{}, err
		}
		err = tx.QueryRow(ctx, `
			insert into proposal_versions(proposal_id,version_number,pricing_model,content,conditions,minimum_invoice,setup_fee,content_hash,created_by)
			values($1,$2,$3,$4,$5,$6,$7,$8,$9)
			returning id::text,public_token::text
		`, input.ProposalID, draft.VersionNumber, input.PricingModel, contentJSON, conditionsJSON, input.MinimumInvoice, input.SetupFee, input.ContentHash, userID).Scan(&draft.VersionID, &draft.PublicToken)
		if err != nil {
			return SavedDraft{}, err
		}
	} else if err != nil {
		return SavedDraft{}, err
	} else {
		if _, err := tx.Exec(ctx, `
			update proposal_versions set pricing_model=$2,content=$3,conditions=$4,minimum_invoice=$5,setup_fee=$6,content_hash=$7
			where id=$1 and published_at is null
		`, draft.VersionID, input.PricingModel, contentJSON, conditionsJSON, input.MinimumInvoice, input.SetupFee, input.ContentHash); err != nil {
			return SavedDraft{}, err
		}
		if _, err := tx.Exec(ctx, `delete from proposal_items where proposal_version_id=$1`, draft.VersionID); err != nil {
			return SavedDraft{}, err
		}
	}

	for _, item := range input.Items {
		metadata, _ := json.Marshal(map[string]any{"catalog_id": item.CatalogID})
		if _, err := tx.Exec(ctx, `
			insert into proposal_items(proposal_version_id,group_name,label,unit,price,is_optional,sort_order,metadata)
			values($1,$2,$3,$4,$5,$6,$7,$8)
		`, draft.VersionID, item.GroupName, item.Label, item.Unit, item.Price, item.IsOptional, item.SortOrder, metadata); err != nil {
			return SavedDraft{}, err
		}
	}
	draft.ProposalID = input.ProposalID
	if err := tx.Commit(ctx); err != nil {
		return SavedDraft{}, err
	}
	return draft, nil
}

func (s *Store) Publish(ctx context.Context, userID string, allowAll bool, versionID string) (string, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var proposalID, owner, token, status string
	var versionNumber int
	err = tx.QueryRow(ctx, `
		select v.proposal_id::text,p.created_by::text,v.version_number,v.public_token::text,p.status::text
		from proposal_versions v join proposals p on p.id=v.proposal_id
		where v.id=$1 and v.published_at is null for update of v,p
	`, versionID).Scan(&proposalID, &owner, &versionNumber, &token, &status)
	if err != nil {
		return "", err
	}
	if !allowAll && owner != userID {
		return "", fmt.Errorf("proposal access denied")
	}
	if status == "accepted" || status == "cancelled" {
		return "", fmt.Errorf("proposal cannot be published in its current state")
	}
	if _, err := tx.Exec(ctx, `update proposal_versions set published_at=now() where id=$1`, versionID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `update proposals set status='published',current_version=$2,updated_at=now() where id=$1`, proposalID, versionNumber); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata) values($1,'proposal.published','proposal',$2,jsonb_build_object('version',$3::integer,'version_id',$4::uuid))`, userID, proposalID, versionNumber, versionID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) EditorByID(ctx context.Context, userID, proposalID string, allowAll bool) (EditorInput, SavedDraft, error) {
	var input EditorInput
	var draft SavedDraft
	var owner string
	err := s.pool.QueryRow(ctx, `
		select p.id::text,p.title,p.valid_until,p.created_by::text,
		       coalesce(c.legal_name,''),coalesce(c.trade_name,''),coalesce(c.cnpj,''),coalesce(c.email::text,''),coalesce(c.phone,''),
		       coalesce(c.street,''),coalesce(c.street_number,''),coalesce(c.complement,''),coalesce(c.district,''),
		       coalesce(c.city,''),coalesce(c.state,''),coalesce(c.postal_code,'')
		from proposals p join clients c on c.id=p.client_id where p.id=$1
	`, proposalID).Scan(
		&input.ProposalID,
		&input.Title,
		&input.ValidUntil,
		&owner,
		&input.ClientLegalName,
		&input.ClientTradeName,
		&input.ClientCNPJ,
		&input.ClientEmail,
		&input.ClientPhone,
		&input.ClientStreet,
		&input.ClientStreetNumber,
		&input.ClientComplement,
		&input.ClientDistrict,
		&input.ClientCity,
		&input.ClientState,
		&input.ClientPostalCode,
	)
	if err != nil {
		return EditorInput{}, SavedDraft{}, err
	}
	if !allowAll && owner != userID {
		return EditorInput{}, SavedDraft{}, fmt.Errorf("proposal access denied")
	}
	var contentJSON, conditionsJSON []byte
	err = s.pool.QueryRow(ctx, `
		select id::text,version_number,public_token::text,pricing_model,minimum_invoice,setup_fee,content,conditions,content_hash
		from proposal_versions where proposal_id=$1 order by version_number desc limit 1
	`, proposalID).Scan(&draft.VersionID, &draft.VersionNumber, &draft.PublicToken, &input.PricingModel, &input.MinimumInvoice, &input.SetupFee, &contentJSON, &conditionsJSON, &input.ContentHash)
	if err != nil && err != pgx.ErrNoRows {
		return EditorInput{}, SavedDraft{}, err
	}
	draft.ProposalID = proposalID
	if len(contentJSON) > 0 {
		_ = json.Unmarshal(contentJSON, &input.Content)
		input.ClientLogoURL = contentString(input.Content, "client", "logo_url")
		input.ContactName = contentString(input.Content, "contact", "name")
		input.ContactRole = contentString(input.Content, "contact", "role")
		input.ContactEmail = contentString(input.Content, "contact", "email")
		input.ContactPhone = contentString(input.Content, "contact", "phone")
		input.OperationContext = contentRootString(input.Content, "operation_context")
		input.CustomerPriorities = contentRootStrings(input.Content, "customer_priorities")
		input.SolutionTitle = contentRootString(input.Content, "solution_title")
		input.SolutionScope = contentRootStrings(input.Content, "solution_scope")
	}
	if len(conditionsJSON) > 0 {
		_ = json.Unmarshal(conditionsJSON, &input.Conditions)
	}
	if draft.VersionID != "" {
		rows, err := s.pool.Query(ctx, `select coalesce(metadata->>'catalog_id',''),group_name,label,coalesce(unit,''),price,is_optional,sort_order from proposal_items where proposal_version_id=$1 order by sort_order,id`, draft.VersionID)
		if err != nil {
			return EditorInput{}, SavedDraft{}, err
		}
		defer rows.Close()
		for rows.Next() {
			var item EditorItem
			if err := rows.Scan(&item.CatalogID, &item.GroupName, &item.Label, &item.Unit, &item.Price, &item.IsOptional, &item.SortOrder); err != nil {
				return EditorInput{}, SavedDraft{}, err
			}
			input.Items = append(input.Items, item)
		}
	}
	return input, draft, nil
}

func contentString(content map[string]any, section, key string) string {
	group, ok := content[section].(map[string]any)
	if !ok {
		return ""
	}
	value, _ := group[key].(string)
	return value
}

func contentRootString(content map[string]any, key string) string {
	value, _ := content[key].(string)
	return value
}

func contentRootStrings(content map[string]any, key string) []string {
	values, ok := content[key].([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok && text != "" {
			result = append(result, text)
		}
	}
	return result
}
