package domain

import "time"

type User struct {
	ID        string
	Email     string
	Name      string
	Phone     string
	JobTitle  string
	Status    string
	Roles     []string
	CreatedAt time.Time
}

type Proposal struct {
	ID             string
	ClientID       string
	ClientName     string
	Title          string
	Status         string
	CurrentVersion int
	PublicToken    string
	ValidUntil     *time.Time
	CreatedBy      string
	CreatedByName  string
	UpdatedAt      time.Time
}

type ProposalVersion struct {
	ID             string
	ProposalID     string
	VersionNumber  int
	PublicToken    string
	PricingModel   string
	ContentJSON    []byte
	ConditionsJSON []byte
	MinimumInvoice float64
	SetupFee       float64
	ContentHash    []byte
	PublishedAt    *time.Time
}

type ProposalAcceptance struct {
	ID                string
	ProposalID        string
	ProposalVersionID string
	Name              string
	Email             string
	CPF               string
	Phone             string
	Role              string
	AuthorityDeclared bool
	AcceptedAt        time.Time
}

type Onboarding struct {
	ID                         string
	ProposalAcceptanceID       string
	ClientID                   string
	Status                     string
	ReviewNotes                string
	CNPJ                       string
	LegalName                  string
	TradeName                  string
	Street                     string
	StreetNumber               string
	Complement                 string
	District                   string
	City                       string
	State                      string
	PostalCode                 string
	OperationType              string
	Insurer                    string
	PolicyStartDate            string
	PolicyEndDate              string
	BrokerCompany              string
	BrokerProducer             string
	CompanyResponsibleName     string
	CompanyResponsibleCPF      string
	CompanyResponsiblePhone    string
	CompanyResponsibleEmail    string
	CompanyResponsibleRole     string
	AuthorityDeclared          bool
	FinanceResponsibleName     string
	FinanceResponsiblePhone    string
	FinanceResponsibleEmail    string
	Goods                      []string
	SystemUsers                []OnboardingSystemUser
}

type OnboardingSystemUser struct {
	Name  string
	Phone string
	Email string
}

type ContractTemplate struct {
	ID             string
	Name           string
	Description    string
	CurrentVersion int
	IsActive       bool
}

type ContractTemplateVersion struct {
	ID                 string
	ContractTemplateID string
	VersionNumber      int
	Markdown           string
	TemplateHash       []byte
}

type Contract struct {
	ID                       string
	OnboardingID             string
	ProposalVersionID        string
	TemplateVersionID        string
	Status                   string
	RenderedMarkdown         string
	RenderedHTML             string
	PDFStorageKey            string
	DocumentSHA256           []byte
	EvidenceReportStorageKey string
	EvidenceReportSHA256     []byte
	FinalPackageStorageKey   string
	FinalPackageSHA256       []byte
	GeneratedAt              *time.Time
	SentAt                   *time.Time
	FullySignedAt            *time.Time
	FinalizedAt              *time.Time
}

type ContractSigner struct {
	ID                 string
	ContractID         string
	SignerType         string
	Name               string
	Email              string
	CPF                string
	Role               string
	SignOrder          int
	Status             string
	SignedAt           *time.Time
	SignatureSessionID *string
}

type AuditEvent struct {
	ID           int64
	ActorUserID  *string
	ActorType    string
	EventType    string
	ResourceType string
	ResourceID   *string
	MetadataJSON []byte
	CreatedAt    time.Time
}

type PipelineItem struct {
	ProposalID        string
	ProposalTitle     string
	ClientName        string
	CommercialName    string
	ProposalStatus    string
	OnboardingID      string
	OnboardingStatus  string
	ContractID        string
	ContractStatus    string
	AcceptedAt        *time.Time
	SubmittedAt       *time.Time
	FullySignedAt     *time.Time
	UpdatedAt         time.Time
}

type PipelineEvent struct {
	EventType    string
	ActorType    string
	ActorName    string
	MetadataJSON []byte
	CreatedAt    time.Time
}
