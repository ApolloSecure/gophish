package controllers

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gophish/gophish/auth"
	"github.com/gophish/gophish/config"
	"github.com/gophish/gophish/models"
	"github.com/gophish/gophish/testutil"
)

// testContext is the data required to test API related functions
type testContext struct {
	apiKey      string
	config      *config.Config
	adminServer *httptest.Server
	phishServer *httptest.Server
	origPath    string
}

func setupTest(t *testing.T) *testContext {
	conf, cleanup, err := testutil.NewTestConfig("controllers")
	if err != nil {
		t.Fatalf("error creating test config: %v", err)
	}
	t.Cleanup(func() {
		if err := models.Close(); err != nil {
			t.Fatalf("error closing database: %v", err)
		}
		if err := cleanup(); err != nil {
			t.Fatalf("error cleaning up database: %v", err)
		}
	})
	err = models.Setup(conf)
	if err != nil {
		t.Fatalf("error setting up database: %v", err)
	}
	ctx := &testContext{}
	ctx.config = conf
	ctx.adminServer = testutil.NewLocalServer(t, NewAdminServer(ctx.config.AdminConf).server.Handler)
	// Get the API key to use for these tests
	u, err := models.GetUser(1)
	// Reset the temporary password for the admin user to a value we control
	hash, err := auth.GeneratePasswordHash("gophish")
	u.Hash = hash
	models.PutUser(&u)
	if err != nil {
		t.Fatalf("error getting first user from database: %v", err)
	}

	// Create a second user to test account locked status
	u2 := models.User{Username: "houdini", Hash: hash, AccountLocked: true}
	models.PutUser(&u2)
	if err != nil {
		t.Fatalf("error creating new user: %v", err)
	}

	ctx.apiKey = u.ApiKey
	// Start the phishing server
	ctx.phishServer = testutil.NewLocalServer(t, NewPhishingServer(ctx.config.PhishConf).server.Handler)
	// Move our cwd up to the project root for help with resolving
	// static assets
	origPath, _ := os.Getwd()
	ctx.origPath = origPath
	err = os.Chdir("../")
	if err != nil {
		t.Fatalf("error changing directories to setup asset discovery: %v", err)
	}
	createTestData(t)
	return ctx
}

func tearDown(t *testing.T, ctx *testContext) {
	// Tear down the admin and phishing servers
	ctx.adminServer.Close()
	ctx.phishServer.Close()
	// Reset the path for the next test
	os.Chdir(ctx.origPath)
}

func createTestData(t *testing.T) {
	// Add a group
	group := models.Group{Name: "Test Group"}
	group.Targets = []models.Target{
		models.Target{BaseRecipient: models.BaseRecipient{Email: "test1@example.com", FirstName: "First", LastName: "Example"}},
		models.Target{BaseRecipient: models.BaseRecipient{Email: "test2@example.com", FirstName: "Second", LastName: "Example"}},
	}
	group.UserId = 1
	models.PostGroup(&group)

	// Add a template
	template := models.Template{Name: "Test Template"}
	template.Subject = "Test subject"
	template.Text = "Text text"
	template.HTML = "<html>Test</html>"
	template.UserId = 1
	models.PostTemplate(&template)

	// Add a landing page
	p := models.Page{Name: "Test Page"}
	p.HTML = "<html>Test</html>"
	p.UserId = 1
	models.PostPage(&p)

	// Add a sending profile
	smtp := models.SMTP{Name: "Test Page"}
	smtp.UserId = 1
	smtp.Host = "example.com"
	smtp.FromAddress = "test@test.com"
	models.PostSMTP(&smtp)

	// Setup and "launch" our campaign
	// Set the status such that no emails are attempted
	c := models.Campaign{Name: "Test campaign"}
	c.UserId = 1
	c.Template = template
	c.Page = p
	c.SMTP = smtp
	c.Groups = []models.Group{group}
	models.PostCampaign(&c, c.UserId)
	c.UpdateStatus(models.CampaignEmailsSent)
}
