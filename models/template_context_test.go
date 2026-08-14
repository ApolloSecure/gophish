package models

import (
	"fmt"

	check "gopkg.in/check.v1"
)

type mockTemplateContext struct {
	URL         string
	FromAddress string
}

func (m mockTemplateContext) getFromAddress() string {
	return m.FromAddress
}

func (m mockTemplateContext) getBaseURL() string {
	return m.URL
}

func (s *ModelsSuite) TestNewTemplateContext(c *check.C) {
	r := Result{
		CustomFields: CustomFields{"Department": "Finance"},
		BaseRecipient: BaseRecipient{
			FirstName: "Foo",
			LastName:  "Bar",
			Email:     "foo@bar.com",
		},
		RId: "1234567",
	}
	ctx := mockTemplateContext{
		URL:         "http://example.com",
		FromAddress: "From Address <from@example.com>",
	}
	expected := PhishingTemplateContext{
		URL:           fmt.Sprintf("%s?rid=%s", ctx.URL, r.RId),
		BaseURL:       ctx.URL,
		BaseRecipient: r.BaseRecipient,
		TrackingURL:   fmt.Sprintf("%s/track?rid=%s", ctx.URL, r.RId),
		From:          "From Address",
		RId:           r.RId,
		Custom:        r.CustomFields,
	}
	expected.Tracker = "<img alt='' style='display: none' src='" + expected.TrackingURL + "'/>"
	got, err := NewPhishingTemplateContext(ctx, r.BaseRecipient, r.CustomFields, r.RId)
	c.Assert(err, check.Equals, nil)
	c.Assert(got, check.DeepEquals, expected)
}

func (s *ModelsSuite) TestCustomFieldTemplating(c *check.C) {
	data := PhishingTemplateContext{
		Custom: CustomFields{
			"AccountName": "A & B <Holdings>",
		},
	}

	text, err := ExecuteTemplate("Hello {{.Custom.AccountName}}", data)
	c.Assert(err, check.Equals, nil)
	c.Assert(text, check.Equals, "Hello A & B <Holdings>")

	html, err := ExecuteHTMLTemplate("<p>{{.Custom.AccountName}}</p>", data)
	c.Assert(err, check.Equals, nil)
	c.Assert(html, check.Equals, "<p>A &amp; B &lt;Holdings&gt;</p>")

	conditional, err := ExecuteTemplate("{{if .Custom.Missing}}yes{{else}}no{{end}}", data)
	c.Assert(err, check.Equals, nil)
	c.Assert(conditional, check.Equals, "no")

	c.Assert(ValidateTemplate("{{.Custom.AnyValidKey}}"), check.Equals, nil)
	c.Assert(ValidateTemplate("{{.Custom.invalid-key}}"), check.Equals, errUnsupportedTemplateSyntax)
}
