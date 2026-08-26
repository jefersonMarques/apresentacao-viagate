package contracts

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
)

var placeholderPattern = regexp.MustCompile(`\{([a-zA-Z0-9_.-]+)\}`)
var conditionalPattern = regexp.MustCompile(`(?s)\{%\s*if\s+([a-zA-Z0-9_.-]+)\s*%\}(.*?)\{%\s*endif\s*%\}`)

type Data map[string]any

type Renderer struct {
	markdown goldmark.Markdown
}

func NewRenderer() *Renderer {
	return &Renderer{markdown: goldmark.New()}
}

func (r *Renderer) Render(markdown string, data Data) (renderedMarkdown string, renderedHTML string, err error) {
	text := conditionalPattern.ReplaceAllStringFunc(markdown, func(block string) string {
		matches := conditionalPattern.FindStringSubmatch(block)
		if len(matches) != 3 {
			return ""
		}
		if truthy(resolve(data, matches[1])) {
			return matches[2]
		}
		return ""
	})

	missing := map[string]struct{}{}
	text = placeholderPattern.ReplaceAllStringFunc(text, func(token string) string {
		matches := placeholderPattern.FindStringSubmatch(token)
		if len(matches) != 2 {
			return token
		}
		value := resolve(data, matches[1])
		if value == nil {
			missing[matches[1]] = struct{}{}
			return token
		}
		return fmt.Sprint(value)
	})

	if len(missing) > 0 {
		keys := make([]string, 0, len(missing))
		for key := range missing {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return "", "", fmt.Errorf("missing contract variables: %s", strings.Join(keys, ", "))
	}

	var html bytes.Buffer
	if err := r.markdown.Convert([]byte(text), &html); err != nil {
		return "", "", fmt.Errorf("render markdown: %w", err)
	}

	// The generated HTML is rendered inside a controlled document template.
	page := template.HTML(html.String())
	return text, string(page), nil
}

func resolve(data Data, path string) any {
	parts := strings.Split(path, ".")
	var current any = map[string]any(data)
	for _, part := range parts {
		mapping, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current, ok = mapping[part]
		if !ok {
			return nil
		}
	}
	return current
}

func truthy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.TrimSpace(typed) != ""
	case nil:
		return false
	default:
		return true
	}
}

func AllowedVariables() []string {
	return []string{
		"client.legal_name",
		"client.trade_name",
		"client.cnpj",
		"client.address",
		"client.city",
		"client.state",
		"representative.name",
		"representative.cpf",
		"representative.email",
		"representative.phone",
		"representative.role",
		"proposal.minimum_invoice",
		"proposal.setup_fee",
		"proposal.accepted_at",
		"proposal.valid_until",
		"viagate.legal_name",
		"viagate.cnpj",
		"products.cargo_score",
		"products.cargo_truck",
		"products.prevention",
		"products.monitoring",
	}
}
