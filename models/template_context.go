package models

import (
	"bytes"
	"errors"
	htmltemplate "html/template"
	"net/mail"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// TemplateContext is an interface that allows both campaigns and email
// requests to have a PhishingTemplateContext generated for them.
type TemplateContext interface {
	getFromAddress() string
	getBaseURL() string
}

// PhishingTemplateContext is the context that is sent to any template, such
// as the email or landing page content.
type PhishingTemplateContext struct {
	From        string
	URL         string
	Tracker     string
	TrackingURL string
	RId         string
	BaseURL     string
	BaseRecipient
}

// NewPhishingTemplateContext returns a populated PhishingTemplateContext,
// parsing the correct fields from the provided TemplateContext and recipient.
func NewPhishingTemplateContext(ctx TemplateContext, r BaseRecipient, rid string) (PhishingTemplateContext, error) {
	f, err := mail.ParseAddress(ctx.getFromAddress())
	if err != nil {
		return PhishingTemplateContext{}, err
	}
	fn := f.Name
	if fn == "" {
		fn = f.Address
	}
	templateURL, err := ExecuteTemplate(ctx.getBaseURL(), newPhishingTemplateContext("", "", "", "", "", "", r))
	if err != nil {
		return PhishingTemplateContext{}, err
	}

	// For the base URL, we'll reset the the path and the query.
	baseURL, err := url.Parse(templateURL)
	if err != nil {
		return PhishingTemplateContext{}, err
	}
	baseURL.Path = ""
	baseURL.RawQuery = ""

	phishURL, _ := url.Parse(templateURL)
	q := phishURL.Query()
	q.Set(RecipientParameter, rid)
	phishURL.RawQuery = q.Encode()

	trackingURL, _ := url.Parse(templateURL)
	trackingURL.Path = path.Join(trackingURL.Path, "/track")
	trackingURL.RawQuery = q.Encode()

	return newPhishingTemplateContext(
		fn,
		phishURL.String(),
		trackingURL.String(),
		"<img alt='' style='display: none' src='"+trackingURL.String()+"'/>",
		rid,
		baseURL.String(),
		r,
	), nil
}

func newPhishingTemplateContext(from, url, trackingURL, tracker, rid, baseURL string, recipient BaseRecipient) PhishingTemplateContext {
	return PhishingTemplateContext{
		From:          from,
		URL:           url,
		TrackingURL:   trackingURL,
		Tracker:       tracker,
		RId:           rid,
		BaseURL:       baseURL,
		BaseRecipient: recipient,
	}
}

type templateEscaper func(field, value string) string

var (
	errUnsupportedTemplateSyntax = errors.New("unsupported template syntax")
	templateTagPattern           = regexp.MustCompile(`{{\s*(.*?)\s*}}`)
)

func textTemplateEscaper(_ string, value string) string {
	return value
}

func htmlTemplateEscaper(field, value string) string {
	if field == "Tracker" {
		return value
	}
	return htmltemplate.HTMLEscapeString(value)
}

func templateFieldValue(data PhishingTemplateContext, field string) (string, bool) {
	switch field {
	case "From":
		return data.From, true
	case "URL":
		return data.URL, true
	case "Tracker":
		return data.Tracker, true
	case "TrackingURL", "TrackingUrl":
		return data.TrackingURL, true
	case "RId":
		return data.RId, true
	case "BaseURL":
		return data.BaseURL, true
	case "FirstName":
		return data.FirstName, true
	case "LastName":
		return data.LastName, true
	case "Email":
		return data.Email, true
	case "Position":
		return data.Position, true
	default:
		return "", false
	}
}

func renderTemplateBlock(parts []string, idx int, data PhishingTemplateContext, escaper templateEscaper) (string, int, string, error) {
	var buff bytes.Buffer
	for idx < len(parts) {
		part := parts[idx]
		if idx%2 == 0 {
			buff.WriteString(part)
			idx++
			continue
		}

		tag := strings.TrimSpace(part)
		switch {
		case tag == "end":
			return buff.String(), idx + 1, "end", nil
		case tag == "else":
			return buff.String(), idx + 1, "else", nil
		case strings.HasPrefix(tag, "."):
			field := strings.TrimPrefix(tag, ".")
			value, ok := templateFieldValue(data, field)
			if !ok {
				return "", idx, "", errUnsupportedTemplateSyntax
			}
			buff.WriteString(escaper(field, value))
			idx++
		case strings.HasPrefix(tag, "if "):
			fieldRef := strings.TrimSpace(strings.TrimPrefix(tag, "if "))
			if !strings.HasPrefix(fieldRef, ".") {
				return "", idx, "", errUnsupportedTemplateSyntax
			}
			field := strings.TrimPrefix(fieldRef, ".")
			value, ok := templateFieldValue(data, field)
			if !ok {
				return "", idx, "", errUnsupportedTemplateSyntax
			}

			thenBlock, nextIdx, terminator, err := renderTemplateBlock(parts, idx+1, data, escaper)
			if err != nil {
				return "", idx, "", err
			}

			elseBlock := ""
			if terminator == "else" {
				elseBlock, nextIdx, terminator, err = renderTemplateBlock(parts, nextIdx, data, escaper)
				if err != nil {
					return "", idx, "", err
				}
			}
			if terminator != "end" {
				return "", idx, "", errUnsupportedTemplateSyntax
			}

			if value != "" {
				buff.WriteString(thenBlock)
			} else {
				buff.WriteString(elseBlock)
			}
			idx = nextIdx
		default:
			return "", idx, "", errUnsupportedTemplateSyntax
		}
	}
	return buff.String(), idx, "", nil
}

func executeTemplate(text string, data PhishingTemplateContext, escaper templateEscaper) (string, error) {
	matches := templateTagPattern.FindAllStringSubmatchIndex(text, -1)
	parts := make([]string, 0, len(matches)*2+1)
	last := 0
	for _, match := range matches {
		parts = append(parts, text[last:match[0]])
		parts = append(parts, text[match[2]:match[3]])
		last = match[1]
	}
	parts = append(parts, text[last:])
	rendered, idx, terminator, err := renderTemplateBlock(parts, 0, data, escaper)
	if err != nil {
		return "", err
	}
	if idx != len(parts) || terminator != "" {
		return "", errUnsupportedTemplateSyntax
	}
	return rendered, nil
}

// ExecuteTemplate creates a templated string based on the provided template
// body and data.
func ExecuteTemplate(text string, data PhishingTemplateContext) (string, error) {
	return executeTemplate(text, data, textTemplateEscaper)
}

// ExecuteHTMLTemplate renders supported phishing template fields into HTML and
// escapes interpolated values by default.
func ExecuteHTMLTemplate(text string, data PhishingTemplateContext) (string, error) {
	return executeTemplate(text, data, htmlTemplateEscaper)
}

// ValidationContext is used for validating templates and pages.
type ValidationContext struct {
	FromAddress string
	BaseURL     string
}

func (vc ValidationContext) getFromAddress() string {
	return vc.FromAddress
}

func (vc ValidationContext) getBaseURL() string {
	return vc.BaseURL
}

// ValidateTemplate ensures that the provided text in the page or template uses
// the supported template variables correctly.
func ValidateTemplate(text string) error {
	return validateTemplate(text, false)
}

// ValidateHTMLTemplate ensures that HTML content uses only supported template
// constructs and fields.
func ValidateHTMLTemplate(text string) error {
	return validateTemplate(text, true)
}

func validateTemplate(text string, isHTML bool) error {
	vc := ValidationContext{
		FromAddress: "foo@bar.com",
		BaseURL:     "http://example.com",
	}
	td := Result{
		BaseRecipient: BaseRecipient{
			Email:     "foo@bar.com",
			FirstName: "Foo",
			LastName:  "Bar",
			Position:  "Test",
		},
		RId: "123456",
	}
	ptx, err := NewPhishingTemplateContext(vc, td.BaseRecipient, td.RId)
	if err != nil {
		return err
	}
	if isHTML {
		_, err = ExecuteHTMLTemplate(text, ptx)
	} else {
		_, err = ExecuteTemplate(text, ptx)
	}
	return err
}
