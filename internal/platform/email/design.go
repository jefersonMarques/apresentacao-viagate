package email

import (
	"html"
	"strings"
)

func RenderViaGateHTML(subject, htmlBody, textBody string) string {
	body := strings.TrimSpace(htmlBody)
	if isFullHTMLDocument(body) {
		return body
	}
	if body == "" {
		body = plainTextHTML(textBody)
	}
	if body == "" {
		body = "<p>Mensagem enviada pela ViaGate.</p>"
	}

	title := strings.TrimSpace(subject)
	if title == "" {
		title = "ViaGate"
	}

	return `<!doctype html>
<html lang="pt-BR">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>` + html.EscapeString(title) + `</title>
</head>
<body style="margin:0;padding:0;background:#eef3f6;font-family:Arial,Helvetica,sans-serif;color:#102637">
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="width:100%;background:#eef3f6">
<tr>
<td align="center" style="padding:28px 12px">
<table role="presentation" width="620" cellspacing="0" cellpadding="0" border="0" style="width:100%;max-width:620px;background:#ffffff;border:1px solid #d6e0e6">
<tr>
<td style="padding:24px 28px;background:#071827;border-bottom:4px solid #ff6b18">
<div style="color:#ff6b18;font-size:11px;font-weight:700;letter-spacing:2px">VIAGATE</div>
<div style="margin-top:7px;color:#ffffff;font-size:22px;font-weight:700;line-height:1.25">` + html.EscapeString(title) + `</div>
</td>
</tr>
<tr>
<td style="padding:30px 28px;color:#102637;font-size:14px;line-height:1.65">
` + body + `
</td>
</tr>
<tr>
<td style="padding:18px 28px;border-top:1px solid #e2e8ec;background:#f8fafb;color:#82919a;font-size:11px;line-height:1.6">
ViaGate · Comunicação automática da plataforma comercial
</td>
</tr>
</table>
</td>
</tr>
</table>
</body>
</html>`
}

func isFullHTMLDocument(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lower, "<!doctype html") || strings.HasPrefix(lower, "<html")
}

func plainTextHTML(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	escaped := html.EscapeString(value)
	paragraphs := strings.Split(escaped, "\n\n")
	parts := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		paragraph = strings.ReplaceAll(paragraph, "\n", "<br>")
		parts = append(parts, `<p style="margin:0 0 16px">`+paragraph+`</p>`)
	}
	return strings.Join(parts, "")
}
