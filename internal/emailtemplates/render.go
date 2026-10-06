package emailtemplates

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

type Draft struct {
	Subject  string
	HTMLBody string
	TextBody string
}

var allowedVariables = []string{
	"brand.logo_url",
	"client.display_name",
	"client.legal_name",
	"client.trade_name",
	"client.email",
	"contact.name",
	"contact.role",
	"contact.email",
	"contact.greeting",
	"contact.phone",
	"proposal.title",
	"proposal.url",
	"proposal.valid_until",
	"proposal.products_html",
	"proposal.products_text",
	"proposal.products_count",
	"proposal.requires_policy",
	"salesperson.name",
	"salesperson.job_title",
	"salesperson.email",
	"salesperson.phone",
	"salesperson.photo_url",
	"salesperson.linkedin",
	"salesperson.instagram",
}

var variablePattern = regexp.MustCompile("\\{([a-zA-Z0-9_.-]+)\\}")
var unsafeHTMLPattern = regexp.MustCompile("(?is)<\\s*(script|iframe|object|embed|form|base|meta|link)\\b|on[a-z]+\\s*=|javascript\\s*:")
var trustedHTMLVariables = map[string]bool{
	"proposal.products_html": true,
}

func AllowedVariables() []string {
	result := make([]string, len(allowedVariables))
	copy(result, allowedVariables)
	return result
}

func Validate(subjectTemplate, htmlTemplate, textTemplate string) error {
	subjectTemplate = strings.TrimSpace(subjectTemplate)
	htmlTemplate = strings.TrimSpace(htmlTemplate)
	if subjectTemplate == "" || htmlTemplate == "" {
		return fmt.Errorf("assunto e conteúdo HTML são obrigatórios")
	}
	if len(subjectTemplate) > 240 {
		return fmt.Errorf("o assunto excede 240 caracteres")
	}
	if strings.ContainsAny(subjectTemplate, "\r\n") {
		return fmt.Errorf("o assunto não pode conter quebras de linha")
	}
	if len(htmlTemplate) > 120000 {
		return fmt.Errorf("o conteúdo HTML excede o limite permitido")
	}
	if len(textTemplate) > 30000 {
		return fmt.Errorf("o conteúdo em texto excede o limite permitido")
	}
	if unsafeHTMLPattern.MatchString(htmlTemplate) {
		return fmt.Errorf("o HTML contém elementos ou atributos não permitidos")
	}

	allowed := map[string]bool{}
	for _, variable := range allowedVariables {
		allowed[variable] = true
	}
	for _, field := range []struct {
		name    string
		content string
	}{
		{name: "assunto", content: subjectTemplate},
		{name: "HTML", content: htmlTemplate},
		{name: "texto", content: textTemplate},
	} {
		for _, match := range variablePattern.FindAllStringSubmatch(field.content, -1) {
			if len(match) < 2 {
				continue
			}
			variable := match[1]
			if !allowed[variable] {
				return fmt.Errorf("variável não permitida: {%s}", variable)
			}
			if trustedHTMLVariables[variable] && field.name != "HTML" {
				return fmt.Errorf("a variável {%s} só pode ser usada no conteúdo HTML", variable)
			}
		}
	}
	return nil
}

func Render(template Template, variables map[string]string) (Draft, error) {
	if err := Validate(template.SubjectTemplate, template.HTMLTemplate, template.TextTemplate); err != nil {
		return Draft{}, err
	}

	subject := template.SubjectTemplate
	htmlBody := template.HTMLTemplate
	textBody := template.TextTemplate
	for _, variable := range allowedVariables {
		token := "{" + variable + "}"
		value := variables[variable]
		subject = strings.ReplaceAll(subject, token, value)
		textBody = strings.ReplaceAll(textBody, token, value)
		if trustedHTMLVariables[variable] {
			htmlBody = strings.ReplaceAll(htmlBody, token, value)
			continue
		}
		htmlBody = strings.ReplaceAll(htmlBody, token, html.EscapeString(value))
	}

	subject = strings.Join(strings.Fields(subject), " ")

	return Draft{
		Subject:  subject,
		HTMLBody: strings.TrimSpace(htmlBody),
		TextBody: strings.TrimSpace(textBody),
	}, nil
}
