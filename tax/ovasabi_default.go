package tax

import (
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

func DefaultOvasabiMetadata(domain string, projectCount int) *commonpb.Metadata {
	now := time.Now().Format(time.RFC3339)
	return &commonpb.Metadata{
		Owner:    &commonpb.OwnerMetadata{Id: "nmxmxh", Wallet: "0xnmxmxh..."},
		Referral: &commonpb.ReferralMetadata{Id: "nmxmxh", Wallet: "0xnmxmxh..."},
		ServiceSpecific: func() *structpb.Struct {
			taxationMap := map[string]interface{}{
				"connectors": []interface{}{
					map[string]interface{}{"type": "creator", "recipient": "nmxmxh", "recipient_wallet": "0xnmxmxh...", "percentage": 0.06, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Main creator"},
					map[string]interface{}{"type": "referral", "recipient": "nmxmxh", "recipient_wallet": "0xnmxmxh...", "percentage": 0.04, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Main referral"},
					map[string]interface{}{"type": "company", "recipient": "ovasabi", "recipient_wallet": "0xOvasabi...", "percentage": 0.05, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Ovasabi company fee"},
					map[string]interface{}{"type": "government", "recipient": "government", "recipient_wallet": "0xGov...", "percentage": 0.04, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Government VAT"},
					map[string]interface{}{"type": "ubi", "recipient": "ubi_pool", "recipient_wallet": "0xUBI...", "percentage": 0.01, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Universal Basic Income"},
					// AI/Tech
					map[string]interface{}{"type": "ai", "recipient": "cursor", "recipient_wallet": "0xCursor...", "percentage": 0.005, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Cursor AI product contribution"},
					map[string]interface{}{"type": "ai", "recipient": "chatgpt", "recipient_wallet": "0xChatGPT...", "percentage": 0.005, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "ChatGPT AI product contribution"},
					map[string]interface{}{"type": "ai", "recipient": "deepseek", "recipient_wallet": "0xDeepSeek...", "percentage": 0.005, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "DeepSeek AI for polishing"},
					map[string]interface{}{"type": "ai", "recipient": "general_ai", "recipient_wallet": "0xGeneralAI...", "percentage": 0.003, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "General AI contribution"},
					map[string]interface{}{"type": "reference", "recipient": "blog_references", "recipient_wallet": "0xBlogRefs...", "percentage": 0.005, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "References and blogs cited"},
					// Family
					map[string]interface{}{"type": "family", "recipient": "duke", "recipient_wallet": "0xDuke...", "percentage": 0.005, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Father"},
					map[string]interface{}{"type": "family", "recipient": "amina", "recipient_wallet": "0xAmina...", "percentage": 0.005, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Mother"},
					map[string]interface{}{"type": "family", "recipient": "martha", "recipient_wallet": "0xMartha...", "percentage": 0.005, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Sister"},
					map[string]interface{}{"type": "family", "recipient": "osi", "recipient_wallet": "0xOsi...", "percentage": 0.005, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Brother"},
					map[string]interface{}{"type": "family", "recipient": "momoh", "recipient_wallet": "0xMomoh...", "percentage": 0.005, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Family"},
					// Best friend and wife/lover
					map[string]interface{}{"type": "friend", "recipient": "best_friend", "recipient_wallet": "0xDrew...", "percentage": 0.005, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Best friend"},
					map[string]interface{}{"type": "spouse", "recipient": "lover", "recipient_wallet": "0xDabby...", "percentage": 0.005, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Lover"},
					// Dynamic friend logic placeholder
					// TODO: Implement dynamic friend connectors
					// Neighbors, friends, and others
					map[string]interface{}{"type": "neighbor", "recipient": "neighbor", "recipient_wallet": "0xNeighbor...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Neighbor support"},
					map[string]interface{}{"type": "friend", "recipient": "Somto", "recipient_wallet": "0xSomto...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Friend Somto"},
					map[string]interface{}{"type": "servant", "recipient": "Solomon", "recipient_wallet": "0xSolomon...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Servant Solomon"},
					map[string]interface{}{"type": "master", "recipient": "Jesus", "recipient_wallet": "0xJesus...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Master Jesus"},
					map[string]interface{}{"type": "friend", "recipient": "Andrew", "recipient_wallet": "0xAndrew...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Friend Andrew"},
					map[string]interface{}{"type": "friend", "recipient": "Anthony", "recipient_wallet": "0xAnthony...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Friend Anthony"},
					map[string]interface{}{"type": "outsource", "recipient": "Outsource", "recipient_wallet": "0xOutsource...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Outsource"},
					map[string]interface{}{"type": "wallet", "recipient": "CloutWallet", "recipient_wallet": "0xCloutWallet...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "CloutWallet"},
					map[string]interface{}{"type": "wallet", "recipient": "BitClout", "recipient_wallet": "0xBitClout...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "BitClout"},
					map[string]interface{}{"type": "company", "recipient": "Whire", "recipient_wallet": "0xWhire...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Whire"},
					map[string]interface{}{"type": "company", "recipient": "GrooveJones", "recipient_wallet": "0xGrooveJones...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "GrooveJones"},
					map[string]interface{}{"type": "company", "recipient": "Headliner", "recipient_wallet": "0xHeadliner...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Headliner"},
					map[string]interface{}{"type": "company", "recipient": "Google", "recipient_wallet": "0xGoogle...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Google"},
					map[string]interface{}{"type": "company", "recipient": "Go", "recipient_wallet": "0xGo...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Go"},
					map[string]interface{}{"type": "company", "recipient": "PirateBay", "recipient_wallet": "0xPirateBay...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "PirateBay"},
					map[string]interface{}{"type": "family", "recipient": "Uncle", "recipient_wallet": "0xUncle...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Uncle"},
					map[string]interface{}{"type": "family", "recipient": "Aunty", "recipient_wallet": "0xAunty...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Aunty"},
					map[string]interface{}{"type": "friend", "recipient": "Chisom", "recipient_wallet": "0xChisom...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Friend Chisom"},
					map[string]interface{}{"type": "friend", "recipient": "Diseye", "recipient_wallet": "0xDiseye...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Friend Diseye"},
					map[string]interface{}{"type": "friend", "recipient": "Chidera", "recipient_wallet": "0xChidera...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Friend Chidera"},
					map[string]interface{}{"type": "friend", "recipient": "Imbia", "recipient_wallet": "0xImbia...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Friend Imbia"},
					map[string]interface{}{"type": "friend", "recipient": "Ebuka", "recipient_wallet": "0xEbuka...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Friend Ebuka"},
					map[string]interface{}{"type": "lawyer", "recipient": "Valentine", "recipient_wallet": "0xValentine...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Lawyer Valentine"},
					map[string]interface{}{"type": "lawyer", "recipient": "AbdulSalllam", "recipient_wallet": "0xAbdulSalllam...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Lawyer AbdulSalllam"},
					// Main industries
					map[string]interface{}{"type": "industry", "recipient": "favored_school", "recipient_wallet": "0xCKC...", "percentage": 0.005, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Favored school"},
					map[string]interface{}{"type": "industry", "recipient": "favored_church", "recipient_wallet": "0xHolyFamily...", "percentage": 0.005, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Favored church"},
					map[string]interface{}{"type": "industry", "recipient": "favored_hospital", "recipient_wallet": "0xJuliusBerger...", "percentage": 0.005, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Favored hospital"},
					map[string]interface{}{"type": "industry", "recipient": "favored_bank", "recipient_wallet": "0xGTBank...", "percentage": 0.005, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Favored bank"},
					// Minority, Black, Continent, African, Country/Location
					map[string]interface{}{"type": "minority", "recipient": "minority_group", "recipient_wallet": "0xMinority...", "percentage": 0.0025, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Minority group support"},
					map[string]interface{}{"type": "black", "recipient": "black_community", "recipient_wallet": "0xBlack...", "percentage": 0.0025, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Black community support"},
					map[string]interface{}{"type": "continent", "recipient": "africa", "recipient_wallet": "0xAfrica...", "percentage": 0.0025, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "African continent support"},
					map[string]interface{}{"type": "african", "recipient": "african_union", "recipient_wallet": "0xAfricanUnion...", "percentage": 0.0025, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "African Union support"},
					map[string]interface{}{"type": "country", "recipient": "country_location", "recipient_wallet": "0xNigeria...", "percentage": 0.0025, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Country/location support (expandable)"},
					// Example for unimplemented industry/relation
					map[string]interface{}{"type": "industry", "recipient": "system", "recipient_wallet": "0xSystem...", "percentage": 0.001, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "System default for unimplemented relation"},
					// After family/friends
					map[string]interface{}{"type": "sleep", "recipient": "sleep", "recipient_wallet": "0xSleep...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Sleep and rest support"},
					// After sleep and before unimplemented/system
					map[string]interface{}{"type": "wake", "recipient": "wake", "recipient_wallet": "0xWake...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Wakefulness and new beginnings"},
					map[string]interface{}{"type": "computer", "recipient": "computer", "recipient_wallet": "0xComputer...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Computing power and digital infrastructure"},
					map[string]interface{}{"type": "sun", "recipient": "sun", "recipient_wallet": "0xSun...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Solar energy and life source"},
					map[string]interface{}{"type": "solar", "recipient": "solar", "recipient_wallet": "0xSolar...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Solar technology and sustainability"},
					map[string]interface{}{"type": "desalination", "recipient": "desalination", "recipient_wallet": "0xDesalination...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Water purification and access"},
					map[string]interface{}{"type": "company", "recipient": "Wyobi", "recipient_wallet": "0xWyobi...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Wyobi company contribution"},
					map[string]interface{}{"type": "person", "recipient": "Salisu", "recipient_wallet": "0xSalisu...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Salisu personal contribution"},
					map[string]interface{}{"type": "art", "recipient": "Art", "recipient_wallet": "0xArt...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Art and creativity"},
					map[string]interface{}{"type": "person", "recipient": "Rapheal", "recipient_wallet": "0xRapheal...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Rapheal personal contribution"},
					map[string]interface{}{"type": "person", "recipient": "Sunkamani", "recipient_wallet": "0xSunkamani...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Sunkamani personal contribution"},
					map[string]interface{}{"type": "person", "recipient": "Osaro", "recipient_wallet": "0xOsaro...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Osaro personal contribution"},
					map[string]interface{}{"type": "community", "recipient": "LifeCamp", "recipient_wallet": "0xLifeCamp...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "LifeCamp community support"},
					// Add new individuals and groups
					map[string]interface{}{"type": "person", "recipient": "Janice", "recipient_wallet": "0xJanice...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Janice personal contribution"},
					map[string]interface{}{"type": "person", "recipient": "Kiki", "recipient_wallet": "0xKiki...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Kiki personal contribution"},
					map[string]interface{}{"type": "person", "recipient": "Eris", "recipient_wallet": "0xEris...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Eris personal contribution"},
					map[string]interface{}{"type": "person", "recipient": "Uloma", "recipient_wallet": "0xUloma...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Uloma personal contribution"},
					map[string]interface{}{"type": "person", "recipient": "Moyo", "recipient_wallet": "0xMoyo...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Moyo personal contribution"},
					map[string]interface{}{"type": "person", "recipient": "Ifunanya", "recipient_wallet": "0xIfunanya...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Ifunanya personal contribution"},
					map[string]interface{}{"type": "person", "recipient": "Nicole", "recipient_wallet": "0xNicole...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Nicole personal contribution"},
					map[string]interface{}{"type": "person", "recipient": "Sarah", "recipient_wallet": "0xSarah...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Sarah personal contribution"},
					map[string]interface{}{"type": "person", "recipient": "Nancy", "recipient_wallet": "0xNancy...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Nancy personal contribution"},
					map[string]interface{}{"type": "person", "recipient": "Lucy", "recipient_wallet": "0xLucy...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Lucy personal contribution"},
					map[string]interface{}{"type": "person", "recipient": "Raymond", "recipient_wallet": "0xRaymond...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Raymond personal contribution"},
					map[string]interface{}{"type": "person", "recipient": "Ibrahim", "recipient_wallet": "0xIbrahim...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Ibrahim personal contribution"},
					map[string]interface{}{"type": "person", "recipient": "Muhammed", "recipient_wallet": "0xMuhammed...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Muhammed personal contribution"},
					map[string]interface{}{"type": "home", "recipient": "home", "recipient_wallet": "0xHome...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Home and shelter"},
					map[string]interface{}{"type": "friends", "recipient": "friends", "recipient_wallet": "0xFriends...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Friends and social support"},
					map[string]interface{}{"type": "clique", "recipient": "clique", "recipient_wallet": "0xClique...", "percentage": 0.002, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Clique and close circle"},
					// Push to 30%: limited, growing, referral, leaderboard, campaign, user, talent, pioneer, hustler, business
					map[string]interface{}{"type": "growth", "recipient": "limited", "recipient_wallet": "0xLimited...", "percentage": 0.01, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Limited growth allocation"},
					map[string]interface{}{"type": "growth", "recipient": "growing", "recipient_wallet": "0xGrowing...", "percentage": 0.01, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Growing allocation"},
					map[string]interface{}{"type": "referral", "recipient": "referral", "recipient_wallet": "0xReferral...", "percentage": 0.01, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Referral bonus"},
					map[string]interface{}{"type": "leaderboard", "recipient": "leaderboard", "recipient_wallet": "0xLeaderboard...", "percentage": 0.01, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Leaderboard rewards"},
					map[string]interface{}{"type": "campaign", "recipient": "campaign", "recipient_wallet": "0xCampaign...", "percentage": 0.01, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Campaign allocation"},
					map[string]interface{}{"type": "user", "recipient": "user", "recipient_wallet": "0xUser...", "percentage": 0.01, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "User allocation"},
					map[string]interface{}{"type": "talent", "recipient": "talent", "recipient_wallet": "0xTalent...", "percentage": 0.01, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Talent allocation"},
					map[string]interface{}{"type": "pioneer", "recipient": "pioneer", "recipient_wallet": "0xPioneer...", "percentage": 0.01, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Pioneer allocation"},
					map[string]interface{}{"type": "hustler", "recipient": "hustler", "recipient_wallet": "0xHustler...", "percentage": 0.01, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Hustler allocation"},
					map[string]interface{}{"type": "business", "recipient": "business", "recipient_wallet": "0xBusiness...", "percentage": 0.01, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Business allocation"},
					// Accessibility and compliance
					map[string]interface{}{"type": "disability", "recipient": "disability_support", "recipient_wallet": "0xDisability...", "percentage": 0.003, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Disability support and inclusion"},
					map[string]interface{}{"type": "compliance", "recipient": "wcag", "recipient_wallet": "0xWCAG...", "percentage": 0.003, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "WCAG accessibility compliance"},
					map[string]interface{}{"type": "compliance", "recipient": "ada", "recipient_wallet": "0xADA...", "percentage": 0.003, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "ADA accessibility compliance"},
					map[string]interface{}{"type": "compliance", "recipient": "section508", "recipient_wallet": "0xSection508...", "percentage": 0.003, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "Section 508 accessibility compliance"},
					map[string]interface{}{"type": "compliance", "recipient": "en301549", "recipient_wallet": "0xEN301549...", "percentage": 0.003, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "EN 301 549 accessibility compliance"},
					map[string]interface{}{"type": "compliance", "recipient": "accessibility", "recipient_wallet": "0xAccessibility...", "percentage": 0.003, "applied_on": now, "domain": domain, "default": true, "enforced": true, "justification": "General accessibility and inclusion"},
				},
				"project_count":   utils.ToInt32(projectCount),
				"total_tax":       0.2725,
				"tax_from_system": true,
			}
			profileMap := map[string]interface{}{
				"talent":   true,
				"pioneer":  true,
				"business": true,
				"hustler":  true,
			}
			// Dynamic blocks for relations (empty arrays or TODO for now)
			dynamicBlocks := map[string]interface{}{
				"friend":   []interface{}{}, // TODO: Populate dynamically
				"family":   []interface{}{}, // TODO: Populate dynamically
				"lover":    []interface{}{}, // TODO: Populate dynamically
				"children": []interface{}{}, // TODO: Populate dynamically
			}
			contract := "Balanced, fair, and system-defaulted contract. Unimplemented relations default to system."
			// Directory structure reference for inspiration and best practices
			// See: docs/amadeus/amadeus_context.md and project root for amazing file/directory organization
			fileStructure := "See project root and docs/amadeus/amadeus_context.md for canonical directory structure."
			// Digital will: error, success, packages
			errorMsg := "If this contract fails, revert to system default and notify all heirs."
			successMsg := "Legacy secured: all relations and assets distributed as intended."
			packages := []string{"pkg/utils/safeint.go", "pkg/graceful/", "internal/service/", "docs/amadeus/amadeus_context.md"}
			ss := metadata.NewStructFromMap(map[string]interface{}{
				"taxation":       taxationMap,
				"profile":        profileMap,
				"dynamic_blocks": dynamicBlocks,
				"contract":       contract,
				"file_structure": fileStructure,
				"error":          errorMsg,
				"success":        successMsg,
				"packages":       packages,
			}, nil)
			return ss
		}(),
	}
}

// MinimalUserTaxMetadata returns a minimal connectors array for a new user: only creator, referrer (if any), government, and ubi.
func MinimalUserTaxMetadata(userID, userWallet, referrerID, referrerWallet string) *commonpb.Metadata {
	now := time.Now().Format(time.RFC3339)
	connectors := []interface{}{
		map[string]interface{}{
			"type":             "creator",
			"recipient":        userID,
			"recipient_wallet": userWallet,
			"percentage":       0.06,
			"applied_on":       now,
			"default":          true,
			"enforced":         true,
			"justification":    "Main creator",
		},
	}
	if referrerID != "" && referrerWallet != "" {
		connectors = append(connectors, map[string]interface{}{
			"type":             "referral",
			"recipient":        referrerID,
			"recipient_wallet": referrerWallet,
			"percentage":       0.04,
			"applied_on":       now,
			"default":          true,
			"enforced":         true,
			"justification":    "Referral",
		})
	}
	// Always add government and ubi
	connectors = append(connectors,
		map[string]interface{}{
			"type":             "government",
			"recipient":        "government",
			"recipient_wallet": "0xGov...",
			"percentage":       0.04,
			"applied_on":       now,
			"default":          true,
			"enforced":         true,
			"justification":    "Government VAT",
		},
		map[string]interface{}{
			"type":             "ubi",
			"recipient":        "ubi_pool",
			"recipient_wallet": "0xUBI...",
			"percentage":       0.01,
			"applied_on":       now,
			"default":          true,
			"enforced":         true,
			"justification":    "Universal Basic Income",
		},
	)
	ss := metadata.NewStructFromMap(map[string]interface{}{
		"taxation": map[string]interface{}{
			"connectors":      connectors,
			"project_count":   0,
			"total_tax":       0.15,
			"tax_from_system": true,
		},
	}, nil)
	return &commonpb.Metadata{
		ServiceSpecific: ss,
	}
}
