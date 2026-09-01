package presentations

import (
	"context"
	"encoding/json"
	"strings"
)

// PublishedByToken loads the immutable current published snapshot without
// applying public-view status rules. It is used to generate internal artifacts
// from a version that has already been published.
func (s *Store) PublishedByToken(ctx context.Context, token string) (PublicPresentation, error) {
	var result PublicPresentation
	var contentJSON []byte
	var currentTitle string
	err := s.pool.QueryRow(ctx, `
		select p.id::text,v.id::text,v.version_number,v.public_token::text,p.title,v.content,v.content_hash
		from presentation_versions v
		join presentations p on p.id=v.presentation_id
		where v.public_token=$1 and v.published_at is not null
		  and v.version_number=p.current_version
	`, token).Scan(
		&result.PresentationID,
		&result.VersionID,
		&result.VersionNumber,
		&result.PublicToken,
		&currentTitle,
		&contentJSON,
		&result.ContentHash,
	)
	if err != nil {
		return PublicPresentation{}, err
	}
	var content presentationContent
	if err := json.Unmarshal(contentJSON, &content); err != nil {
		return PublicPresentation{}, err
	}
	result.Title = currentTitle
	if strings.TrimSpace(content.Title) != "" {
		result.Title = content.Title
	}
	result.ClientLegalName = content.Client.LegalName
	result.ClientTradeName = content.Client.TradeName
	result.ClientLogoURL = content.Client.LogoURL
	result.ContactName = content.Contact.Name
	result.ContactRole = content.Contact.Role
	result.ContactEmail = content.Contact.Email
	result.SalespersonName = content.Salesperson.Name
	result.SalespersonEmail = content.Salesperson.Email
	result.SalespersonPhone = content.Salesperson.Phone
	result.SalespersonJobTitle = content.Salesperson.JobTitle
	result.SalespersonPhotoURL = content.Salesperson.PhotoURL
	result.SalespersonLinkedIn = content.Salesperson.LinkedIn
	result.SalespersonInstagram = content.Salesperson.Instagram
	result.ShowClientIdentity = content.Settings.ShowClientIdentity
	result.ShowContactSlide = content.Settings.ShowContactSlide
	result.SelectedModules = content.Settings.SelectedModules
	return result, nil
}
