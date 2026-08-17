package models

import (
	"testing"

	"github.com/gophish/gophish/config"
	"github.com/gophish/gophish/testutil"
	"gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

type ModelsSuite struct {
	config  *config.Config
	cleanup func() error
}

var _ = check.Suite(&ModelsSuite{})
var benchmarkCleanup func() error

func (s *ModelsSuite) SetUpSuite(c *check.C) {
	conf, cleanup, err := testutil.NewTestConfig("models")
	if err != nil {
		c.Fatalf("Failed creating test config: %v", err)
	}
	s.config = conf
	s.cleanup = cleanup
	err = Setup(conf)
	if err != nil {
		c.Fatalf("Failed creating database: %v", err)
	}
}

func (s *ModelsSuite) TearDownSuite(c *check.C) {
	if err := Close(); err != nil {
		c.Fatalf("error closing database: %v", err)
	}
	if s.cleanup != nil {
		if err := s.cleanup(); err != nil {
			c.Fatalf("error cleaning up database: %v", err)
		}
	}
}

func (s *ModelsSuite) TearDownTest(c *check.C) {
	// Clear database tables between each test. If new tables are
	// used in this test suite they will need to be cleaned up here.
	db.Delete(Group{})
	db.Delete(Target{})
	db.Delete(GroupTarget{})
	db.Delete(Header{})
	db.Delete(SMTP{})
	db.Delete(Attachment{})
	db.Delete(Page{})
	db.Delete(Result{})
	db.Delete(Event{})
	db.Delete(MailLog{})
	db.Delete(Campaign{})
	db.Delete(EmailRequest{})
	db.Delete(Tenant{})
	db.Delete(Webhook{})
	db.Delete(IMAP{})

	// Reset users table to default state.
	db.Not("id", 1).Delete(User{})
	db.Model(User{}).Update("username", "admin")
}

func (s *ModelsSuite) createCampaignDependencies(ch *check.C, optional ...string) Campaign {
	// we use the optional parameter to pass an alternative subject
	group := Group{Name: "Test Group"}
	group.Targets = []Target{
		Target{BaseRecipient: BaseRecipient{Email: "test1@example.com", FirstName: "First", LastName: "Example"}},
		Target{BaseRecipient: BaseRecipient{Email: "test2@example.com", FirstName: "Second", LastName: "Example"}},
		Target{BaseRecipient: BaseRecipient{Email: "test3@example.com", FirstName: "Second", LastName: "Example"}},
		Target{BaseRecipient: BaseRecipient{Email: "test4@example.com", FirstName: "Second", LastName: "Example"}},
	}
	group.UserId = 1
	ch.Assert(PostGroup(&group), check.Equals, nil)

	// Add a template
	t := Template{Name: "Test Template"}
	if len(optional) > 0 {
		t.Subject = optional[0]
	} else {
		t.Subject = "{{.RId}} - Subject"
	}
	t.Text = "{{.RId}} - Text"
	t.HTML = "{{.RId}} - HTML"
	t.UserId = 1
	ch.Assert(PostTemplate(&t), check.Equals, nil)

	// Add a landing page
	p := Page{Name: "Test Page"}
	p.HTML = "<html>Test</html>"
	p.UserId = 1
	ch.Assert(PostPage(&p), check.Equals, nil)

	// Add a sending profile
	smtp := SMTP{Name: "Test Page"}
	smtp.UserId = 1
	smtp.Host = "example.com"
	smtp.FromAddress = "test@test.com"
	ch.Assert(PostSMTP(&smtp), check.Equals, nil)

	c := Campaign{Name: "Test campaign"}
	c.UserId = 1
	c.Template = t
	c.Page = p
	c.SMTP = smtp
	c.Groups = []Group{group}
	return c
}

func (s *ModelsSuite) createCampaign(ch *check.C) Campaign {
	c := s.createCampaignDependencies(ch)
	// Setup and "launch" our campaign
	ch.Assert(PostCampaign(&c, c.UserId), check.Equals, nil)

	// For comparing the dates, we need to fetch the campaign again. This is
	// to solve an issue where the campaign object right now has time down to
	// the microsecond, while in MySQL it's rounded down to the second.
	c, _ = GetCampaign(c.Id, c.UserId)
	return c
}

func setupBenchmark(b *testing.B) {
	conf, cleanup, err := testutil.NewTestConfig("benchmark")
	if err != nil {
		b.Fatalf("Failed creating test config: %v", err)
	}
	benchmarkCleanup = cleanup
	err = Setup(conf)
	if err != nil {
		b.Fatalf("Failed creating database: %v", err)
	}
}

func tearDownBenchmark(b *testing.B) {
	err := Close()
	if err != nil {
		b.Fatalf("error closing database: %v", err)
	}
	if benchmarkCleanup != nil {
		if err := benchmarkCleanup(); err != nil {
			b.Fatalf("error cleaning up database: %v", err)
		}
		benchmarkCleanup = nil
	}
}

func resetBenchmark(b *testing.B) {
	db.Delete(Group{})
	db.Delete(Target{})
	db.Delete(GroupTarget{})
	db.Delete(SMTP{})
	db.Delete(Page{})
	db.Delete(Result{})
	db.Delete(MailLog{})
	db.Delete(Campaign{})

	// Reset users table to default state.
	db.Not("id", 1).Delete(User{})
	db.Model(User{}).Update("username", "admin")
}
