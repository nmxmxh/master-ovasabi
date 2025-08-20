// Command-line tool to create a new campaign record in the OVASABI database.
// Usage: go run create_campaign.go --slug summer2025 --name "Summer 2025 Launch" --features waitlist,referral,leaderboard,broadcast --i18n_keys welcome_banner,signup_cta --broadcast_enabled=true
//
// Requirements:
// - Go 1.18+
// - PostgreSQL database (connection string via env or flag)
//
// This script constructs the robust metadata JSON structure as required by the orchestrator and platform standards.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type CampaignMetadata struct {
	Features        []string               `json:"features"`
	ServiceSpecific map[string]interface{} `json:"service_specific"`
	Scheduling      map[string]interface{} `json:"scheduling,omitempty"`
	Scripts         map[string]interface{} `json:"scripts,omitempty"`
	Versioning      map[string]interface{} `json:"versioning"`
	Audit           map[string]interface{} `json:"audit"`
}

func main() {
	// Flags
	slug := flag.String("slug", "", "Unique campaign slug (required)")
	name := flag.String("name", "", "Campaign name (required)")
	features := flag.String("features", "", "Comma-separated feature list (e.g. waitlist,referral)")
	i18nKeys := flag.String("i18n_keys", "", "Comma-separated i18n keys")
	broadcastEnabled := flag.Bool("broadcast_enabled", false, "Enable broadcast")
	waitlist := flag.Bool("waitlist", false, "Enable waitlist")
	referral := flag.Bool("referral", false, "Enable referral")
	leaderboard := flag.Bool("leaderboard", false, "Enable leaderboard")
	customField := flag.String("custom_field", "", "Custom field value (optional)")
	wsFrequency := flag.String("ws_frequency", "", "WebSocket broadcast frequency (e.g. 5s, 1m, 0 for off)")
	schedStart := flag.String("sched_start", "", "Campaign start time (RFC3339)")
	schedEnd := flag.String("sched_end", "", "Campaign end time (RFC3339)")
	dbURL := flag.String("db_url", os.Getenv("DATABASE_URL"), "Postgres connection string")
	flag.Parse()

	if *slug == "" || *name == "" || *features == "" {
		fmt.Println("slug, name, and features are required.")
		flag.Usage()
		os.Exit(1)
	}

	featuresList := splitAndTrim(*features)
	i18nKeysList := splitAndTrim(*i18nKeys)
	now := time.Now().UTC().Format(time.RFC3339)

	campaignFields := map[string]interface{}{
		"waitlist":          *waitlist,
		"referral":          *referral,
		"leaderboard":       *leaderboard,
		"i18n_keys":         i18nKeysList,
		"broadcast_enabled": *broadcastEnabled,
		"ui_content": map[string]interface{}{
			"banner": "Welcome to the campaign!",
			"cta":    "Join now for rewards!",
			"lead_form": map[string]interface{}{
				"fields":      []string{"name", "email", "referral_code"},
				"submit_text": "Get Started",
			},
		},
		"dialogue_scripts": map[string]interface{}{
			"waitlist": map[string]interface{}{
				"main_text":     "You're on the waitlist!",
				"options_title": "What would you like to do next?",
				"questions": []map[string]interface{}{
					{"question": "Invite friends for early access?", "options": []string{"Yes", "No"}},
				},
			},
			"lead": map[string]interface{}{
				"main_text":     "Request a quote or express interest.",
				"options_title": "Choose your service:",
				"questions": []map[string]interface{}{
					{"question": "Which service are you interested in?", "options": []string{"Design", "Development", "Consulting"}},
				},
			},
			"referral": map[string]interface{}{
				"main_text":     "Enter your referral code.",
				"options_title": "Referral Options",
				"questions": []map[string]interface{}{
					{"question": "Do you have a referral code?", "options": []string{"Yes", "No"}},
				},
			},
		},
		"communication": map[string]interface{}{
			"broadcast": map[string]interface{}{
				"enabled":   *broadcastEnabled,
				"frequency": *wsFrequency,
				"channels":  []string{"ovasabi_website", *slug},
			},
			"user": map[string]interface{}{
				"registration": true,
				"lead_capture": true,
			},
		},
	}
	if *customField != "" {
		campaignFields["custom_field"] = *customField
	}

	scheduling := map[string]interface{}{}
	if *schedStart != "" {
		scheduling["start"] = *schedStart
	}
	if *schedEnd != "" {
		scheduling["end"] = *schedEnd
	}
	if *wsFrequency != "" {
		scheduling["broadcast_frequency"] = *wsFrequency
	}
	if len(scheduling) == 0 {
		scheduling["start"] = now
	}

	// Dialogue scripts, UI content, and communication logic are now included in campaignFields
	scripts := map[string]interface{}{
		"business": map[string]interface{}{
			"main_text":         "Help us understand your business needs",
			"options_title":     "Select what you’d like support with",
			"options_subtitle":  "Your selections help us understand where you’re at, and how we can move you forward.",
			"question_subtitle": "Why this matters",
			"questions": []map[string]interface{}{
				{
					"question":         "How do you <br/> currently establish <br/> your <br/> [light]business[/light] [dark]presence[/dark] <br/> [no-split]online?🌍[/no-split]",
					"why_this_matters": "Many businesses rely on outdated methods or lack a cohesive digital presence, making it difficult for customers to find and engage with them effectively.",
					"options": []string{
						"A well-structured website or landing page with intuitive user experience",
						"Engaging social media channels that align with our business goals",
						"A Google My Business listing to improve local discoverability",
					},
				},
				{
					"question":         "How do you <br/> currently create<br/>  & distribute<br/>  [no-split]your brand's[/no-split] <br />[light]creative &[/light] [dark]advertising[/dark]<br /> content?",
					"why_this_matters": "Customers connect with high-quality, engaging content. Inconsistent branding or unclear messaging leads to missed opportunities.",
					"options": []string{
						"A content strategy aligned with our business goals",
						"AI-powered content tools to scale creation.",
						"A structured content calendar and distribution system.",
					},
				},
				{
					"question":         "[no-split]How do you currently[/no-split]<br /> [no-split]promote your business[/no-split]<br /> &<br /> [light]generate[/light] [dark]leads[/dark]<br /> online?",
					"why_this_matters": "Traditional ads are expensive and hard to track. Digital ads offer precise targeting and measurable performance.",
					"options": []string{
						"Running Google or Meta ads with targeted reach",
						"Using AI to segment audiences for personalized marketing",
						"Conducting A/B tests to boost ad performance",
					},
				},
				{
					"question":         "How do you<br /> [no-split]manage leads,[/no-split] <br />track<br /> [no-split]customer interactions[/no-split]<br /> &<br />  [light]close[/light] [dark]sales?[/dark]",
					"why_this_matters": "Disorganized sales processes lead to missed deals and poor customer follow-up.",
					"options": []string{
						"Using a CRM system to track and manage leads",
						"Adding AI chatbots and lead capture forms",
						"Automating follow-ups to nurture leads effectively.",
					},
				},
				{
					"question":         "[no-split]How do you[/no-split]<br /> [light]track & manage[/light]<br /> your marketing<br /> & <br />sales [dark]performance?[/dark]",
					"why_this_matters": "Without proper analytics, it’s easy to waste money and miss what actually works.",
					"options": []string{
						"Google Analytics for traffic and behavior tracking",
						"Heatmaps & session recordings for user insights",
						"AI dashboards for real-time performance metrics",
					},
				},
				{
					"question":         "How do you<br /> [light]retain customers[/light]<br /> & build<br /> lasting<br /> [dark]relationships?[/dark]",
					"why_this_matters": "Retention costs less than acquisition, but many brands overlook customer experience and loyalty.",
					"options": []string{
						"Loyalty programs and personalized offers",
						"Email/SMS automation for timely follow-ups",
						"AI-powered support chatbots to improve services",
					},
				},
				{
					"question":         "[no-split]How prepared[/no-split]<br /> [no-split]is your business for[/no-split] [light]scaling & adopting[/light]<br /> to future<br /> [dark]digital trends?[/dark]",
					"why_this_matters": "Without proper analytics, it’s easy to waste money and miss what actually works.",
					"options": []string{
						"Google Analytics for traffic and behavior tracking",
						"Heatmaps & session recordings for user insights",
						"AI dashboards for real-time performance metrics",
					},
				},
			},
		},
		"talent": map[string]interface{}{
			"main_text":         "Showcase your talent and join our creative network",
			"options_title":     "What best describes your creative journey?",
			"options_subtitle":  "Your answers help us match you with the right opportunities.",
			"question_subtitle": "Why this matters",
			"questions": []map[string]interface{}{
				{
					"question":         "How do you currently showcase your work and build your creative presence online?",
					"why_this_matters": "A strong digital presence helps you get discovered and connect with collaborators or clients.",
					"options": []string{
						"A personal portfolio website or online gallery",
						"Active social media showcasing your work",
						"Participating in online creative communities",
					},
				},
				{
					"question":         "How do you find and apply for new creative opportunities?",
					"why_this_matters": "Proactive outreach and using the right platforms can open doors to exciting projects.",
					"options": []string{
						"Freelance/job platforms and talent marketplaces",
						"Networking through social media or events",
						"Referrals from past clients or collaborators",
					},
				},
				{
					"question":         "How do you keep your skills sharp and stay inspired?",
					"why_this_matters": "Continuous learning and inspiration fuel creative growth and open new doors.",
					"options": []string{
						"Online courses and tutorials",
						"Mentorship or peer feedback groups",
						"AI-powered creative tools",
					},
				},
				{
					"question":         "How do you manage collaborations and creative projects?",
					"why_this_matters": "Effective project management leads to successful outcomes and lasting partnerships.",
					"options": []string{
						"Project management apps (Trello, Notion, etc.)",
						"Direct messaging and group chats",
						"Freelance platforms for contracts and payments",
					},
				},
				{
					"question":         "How do you promote your work and grow your audience?",
					"why_this_matters": "Promotion and audience engagement are key to building a sustainable creative career.",
					"options": []string{
						"Content strategy and regular posting",
						"Collaborations and cross-promotions",
						"Paid ads or influencer partnerships",
					},
				},
				{
					"question":         "How do you handle feedback and improve your craft?",
					"why_this_matters": "Constructive feedback and iteration help you reach new creative heights.",
					"options": []string{
						"Seeking feedback from peers or mentors",
						"Analyzing audience reactions and analytics",
						"Experimenting with new styles and techniques",
					},
				},
				{
					"question":         "How do you balance creative work with business and self-promotion?",
					"why_this_matters": "Balancing creativity and business is essential for long-term success.",
					"options": []string{
						"Scheduling dedicated time for each",
						"Using tools to automate promotion",
						"Collaborating with managers or agents",
					},
				},
			},
		},
		"pioneer": map[string]interface{}{
			"main_text":         "Shape the future with us as a Pioneer client",
			"options_title":     "What are your biggest innovation goals?",
			"options_subtitle":  "Your vision helps us co-create breakthrough solutions.",
			"question_subtitle": "Why this matters",
			"questions": []map[string]interface{}{
				{
					"question":         "How does your organization currently drive innovation and industry leadership?",
					"why_this_matters": "Understanding your approach helps us align our solutions with your mission.",
					"options": []string{
						"Investing in R&D and new technologies",
						"Building strategic partnerships",
						"Fostering a culture of experimentation",
					},
				},
				{
					"question":         "How do you approach digital transformation and technology adoption?",
					"why_this_matters": "Your approach shapes how we design and deliver innovative solutions.",
					"options": []string{
						"Early adopter, always seeking the latest tech",
						"Strategic, with careful evaluation and pilots",
						"Cautious, preferring proven solutions",
					},
				},
				{
					"question":         "How do you measure the impact of innovation in your organization?",
					"why_this_matters": "Clear metrics and KPIs ensure innovation delivers real value.",
					"options": []string{
						"Business growth and market share",
						"Operational efficiency and cost savings",
						"Social or environmental impact",
					},
				},
				{
					"question":         "What kind of partnership are you seeking?",
					"why_this_matters": "Clear partnership expectations ensure mutual success and value creation.",
					"options": []string{
						"Long-term strategic collaboration",
						"Project-based innovation sprints",
						"Advisory and thought leadership support",
					},
				},
				{
					"question":         "How do you ensure your team is ready for the future?",
					"why_this_matters": "Future readiness is key to sustaining innovation and growth.",
					"options": []string{
						"Continuous learning and upskilling",
						"Hiring and developing top talent",
						"Investing in leadership development",
					},
				},
				{
					"question":         "How do you communicate your innovation story to stakeholders?",
					"why_this_matters": "Effective communication builds support and attracts partners.",
					"options": []string{
						"Thought leadership and public speaking",
						"Case studies and success stories",
						"Media and PR campaigns",
					},
				},
				{
					"question":         "How do you balance risk and reward in your innovation strategy?",
					"why_this_matters": "Balanced risk-taking drives sustainable, impactful innovation.",
					"options": []string{
						"Pilot programs and phased rollouts",
						"Robust risk assessment and mitigation",
						"Learning from failures and iterating quickly",
					},
				},
			},
		},
		"hustler": map[string]interface{}{
			"main_text":         "Accelerate your sales journey as a Hustler",
			"options_title":     "What best describes your sales approach?",
			"options_subtitle":  "Your style helps us empower you with the right tools and leads.",
			"question_subtitle": "Why this matters",
			"questions": []map[string]interface{}{
				{
					"question":         "How do you identify and qualify new leads?",
					"why_this_matters": "Effective lead qualification maximizes your sales efficiency and results.",
					"options": []string{
						"Using CRM and data-driven prospecting",
						"Networking and referrals",
						"Cold outreach and follow-ups",
					},
				},
				{
					"question":         "How do you build and manage your sales pipeline?",
					"why_this_matters": "A well-managed pipeline helps you close more deals and forecast accurately.",
					"options": []string{
						"Sales automation platforms",
						"Spreadsheets and manual tracking",
						"Mobile apps for on-the-go management",
					},
				},
				{
					"question":         "How do you approach closing deals and handling objections?",
					"why_this_matters": "Strong closing skills and objection handling are key to sales success.",
					"options": []string{
						"Consultative selling and value demonstration",
						"Negotiation and relationship building",
						"Persistence and follow-up",
					},
				},
				{
					"question":         "How do you use data and analytics to improve your sales performance?",
					"why_this_matters": "Data-driven insights help you optimize your strategy and results.",
					"options": []string{
						"Tracking KPIs and conversion rates",
						"A/B testing sales scripts and offers",
						"Using dashboards for real-time insights",
					},
				},
				{
					"question":         "How do you stay motivated and keep learning in sales?",
					"why_this_matters": "Continuous improvement is key to long-term sales success.",
					"options": []string{
						"Sales training and mentorship",
						"Setting personal goals and tracking progress",
						"Learning from peers and industry leaders",
					},
				},
				{
					"question":         "How do you leverage technology to scale your sales efforts?",
					"why_this_matters": "The right tools can multiply your reach and productivity.",
					"options": []string{
						"Automating outreach and follow-ups",
						"Integrating CRM with marketing tools",
						"Using AI for lead scoring and personalization",
					},
				},
				{
					"question":         "How do you build lasting relationships with clients?",
					"why_this_matters": "Strong relationships drive repeat business and referrals.",
					"options": []string{
						"Regular check-ins and value delivery",
						"Personalized offers and loyalty programs",
						"Providing excellent post-sale support",
					},
				},
			},
		},
	}

	metadata := CampaignMetadata{
		Features: featuresList,
		ServiceSpecific: map[string]interface{}{
			"campaign": campaignFields,
			"localization": map[string]interface{}{
				"scripts": scripts,
				// This will be populated by the localization service with translation refs/URIs per locale
				"scripts_translations": map[string]interface{}{},
			},
		},
		Scheduling: scheduling,
		// Scripts field at the top level is now deprecated in favor of service_specific.localization.scripts
		Scripts: nil,
		Versioning: map[string]interface{}{
			"version":     now[:10],
			"environment": "dev",
		},
		Audit: map[string]interface{}{
			"created_by": os.Getenv("USER"),
			"created_at": now,
		},
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		fmt.Println("Failed to marshal metadata:", err)
		os.Exit(1)
	}

	if *dbURL == "" {
		fmt.Print("Enter Postgres connection string: ")
		if _, err := fmt.Scanln(dbURL); err != nil {
			fmt.Println("Failed to read Postgres connection string:", err)
			os.Exit(1)
		}
	}

	db, err := sql.Open("postgres", *dbURL)
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		os.Exit(1)
	}
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	var campaignID int
	err = db.QueryRowContext(ctx, `
	      INSERT INTO service_campaign_main (slug, name, metadata, created_at, updated_at)
	      VALUES ($1, $2, $3::jsonb, NOW(), NOW())
	      RETURNING id;
	  `, *slug, *name, string(metadataJSON)).Scan(&campaignID)
	if err != nil {
		fmt.Println("Failed to insert campaign:", err)
		os.Exit(1)
	}
	defer cancel()

	fmt.Printf("Campaign created with id: %d\n", campaignID)
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
