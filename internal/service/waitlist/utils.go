package waitlist

import (
	"regexp"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
)

// Email validation regex.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Username validation regex (alphanumeric, underscore, hyphen, 3-30 chars).
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,30}$`)

// ValidateEmail validates an email address format.
func ValidateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

// ValidateUsername validates a username format.
func ValidateUsername(username string) error {
	if len(username) < 3 || len(username) > 30 {
		return ErrInvalidUsernameLength
	}
	if !usernameRegex.MatchString(username) {
		return ErrInvalidUsernameLength
	}
	// Check for reserved usernames
	reserved := []string{
		// System/Admin terms
		"0", "about", "access", "account", "accounts", "activate", "activities", "activity", "ad", "add", "address", "adm", "admin", "administration", "administrator", "ads", "adult", "advertising", "affiliate", "affiliates", "ajax", "all", "alpha", "analysis", "analytics", "android", "anon", "anonymous", "api", "app", "apps", "archive", "archives", "article", "asct", "asset", "atom", "auth", "authentication", "avatar", "backup", "balancer-manager", "banner", "banners", "beta", "billing", "bin", "blog", "blogs", "board", "book", "bookmark", "bot", "bots", "bug", "business", "cache", "cadastro", "calendar", "call", "campaign", "cancel", "captcha", "career", "careers", "cart", "categories", "category", "cgi", "cgi-bin", "changelog", "chat", "check", "checking", "checkout", "client", "cliente", "clients", "code", "codereview", "comercial", "comment", "comments", "communities", "community", "company", "compare", "compras", "config", "configuration", "connect", "contact", "contact-us", "contact_us", "contactus", "contest", "contribute", "corp", "create", "css", "dashboard", "data", "db", "default", "delete", "demo", "design", "designer", "destroy", "dev", "devel", "developer", "developers", "diagram", "diary", "dict", "dictionary", "die", "dir", "direct_messages", "directory", "dist", "doc", "docs", "documentation", "domain", "download", "downloads", "ecommerce", "edit", "editor", "edu", "education", "email", "employment", "empty", "end", "enterprise", "entries", "entry", "error", "errors", "eval", "event", "everyone", "exit", "explore", "facebook", "faq", "favorite", "favorites", "feature", "features", "feed", "feedback", "feeds", "file", "files", "first", "flash", "fleet", "fleets", "flog", "follow", "followers", "following", "forgot", "form", "forum", "forums", "founder", "free", "friend", "friends", "ftp", "gadget", "gadgets", "game", "games", "get", "ghost", "gift", "gifts", "gist", "github", "graph", "group", "groups", "guest", "guests", "help", "home", "homepage", "host", "hosting", "hostmaster", "hostname", "howto", "hpg", "html", "http", "httpd", "https", "i", "iamges", "icon", "icons", "id", "idea", "ideas", "image", "images", "imap", "img", "index", "indice", "info", "information", "inquiry", "instagram", "intranet", "invitations", "invite", "ipad", "iphone", "irc", "is", "issue", "issues", "it", "item", "items", "java", "javascript", "job", "jobs", "join", "js", "json", "jump", "knowledgebase", "language", "languages", "last", "ldap-status", "legal", "license", "link", "links", "linux", "list", "lists", "log", "log-in", "log-out", "log_in", "log_out", "login", "logout", "logs", "m", "mac", "mail", "mail1", "mail2", "mail3", "mail4", "mail5", "mailer", "mailing", "maintenance", "manager", "manual", "map", "maps", "marketing", "master", "me", "media", "member", "members", "message", "messages", "messenger", "microblog", "microblogs", "mine", "mis", "mob", "mobile", "movie", "movies", "mp3", "msg", "msn", "music", "musicas", "mx", "my", "mysql", "name", "named", "nan", "navi", "navigation", "net", "network", "new", "news", "newsletter", "nick", "nickname", "notes", "noticias", "notification", "notifications", "notify", "ns", "ns1", "ns10", "ns2", "ns3", "ns4", "ns5", "ns6", "ns7", "ns8", "ns9", "null", "oauth", "oauth_clients", "offer", "offers", "official", "old", "online", "openid", "operator", "order", "orders", "organization", "organizations", "overview", "owner", "owners", "page", "pager", "pages", "panel", "password", "payment", "perl", "phone", "photo", "photoalbum", "photos", "php", "phpmyadmin", "phppgadmin", "phpredisadmin", "pic", "pics", "ping", "plan", "plans", "plugin", "plugins", "policy", "pop", "pop3", "popular", "portal", "post", "postfix", "postmaster", "posts", "pr", "premium", "press", "price", "pricing", "privacy", "privacy-policy", "privacy_policy", "privacypolicy", "private", "product", "products", "profile", "project", "projects", "promo", "pub", "public", "purpose", "put", "python", "query", "random", "ranking", "read", "readme", "recent", "recruit", "recruitment", "register", "registration", "release", "remove", "replies", "report", "reports", "repositories", "repository", "req", "request", "requests", "reset", "roc", "root", "rss", "ruby", "rule", "sag", "sale", "sales", "sample", "samples", "save", "school", "script", "scripts", "search", "secure", "security", "self", "send", "server", "server-info", "server-status", "service", "services", "session", "sessions", "setting", "settings", "setup", "share", "shop", "show", "sign-in", "sign-up", "sign_in", "sign_up", "signin", "signout", "signup", "site", "sitemap", "sites", "smartphone", "smtp", "soporte", "source", "spec", "special", "sql", "src", "ssh", "ssl", "ssladmin", "ssladministrator", "sslwebmaster", "staff", "stage", "staging", "start", "stat", "state", "static", "stats", "status", "store", "stores", "stories", "style", "styleguide", "stylesheet", "stylesheets", "subdomain", "subscribe", "subscriptions", "suporte", "support", "svn", "swf", "sys", "sysadmin", "sysadministrator", "system", "tablet", "tablets", "tag", "talk", "task", "tasks", "team", "teams", "tech", "telnet", "term", "terms", "terms-of-service", "terms_of_service", "termsofservice", "test", "test1", "test2", "test3", "teste", "testing", "tests", "theme", "themes", "thread", "threads", "tmp", "todo", "tool", "tools", "top", "topic", "topics", "tos", "tour", "translations", "trends", "tutorial", "tux", "tv", "twitter", "undef", "unfollow", "unsubscribe", "update", "upload", "uploads", "url", "usage", "user", "username", "users", "usuario", "vendas", "ver", "version", "video", "videos", "visitor", "watch", "weather", "web", "webhook", "webhooks", "webmail", "webmaster", "website", "websites", "welcome", "widget", "widgets", "wiki", "win", "windows", "word", "work", "works", "workshop", "ww", "wws", "www", "www1", "www2", "www3", "www4", "www5", "www6", "www7", "wwws", "wwww", "xfn", "xml", "xmpp", "xpg", "xxx", "yaml", "year", "yml", "you", "yourdomain", "yourname", "yoursite", "yourusername",

		// Specific Platform Names
		"nmxmxh", "inos", "sage", "ovasabi", "ovasabi_studios", "nobert", "nobert_momoh", "okhai",

		// Religious Terms (Christianity, Islam, Judaism, Hinduism, Buddhism, etc.)
		"god", "jesus", "christ", "allah", "muhammad", "mohammed", "buddha", "krishna", "vishnu", "shiva", "brahma", "yahweh", "jehovah", "moses", "abraham", "isaac", "jacob", "david", "solomon", "mary", "virgin", "holy", "sacred", "divine", "prophet", "apostle", "saint", "angel", "archangel", "gabriel", "michael", "raphael", "lucifer", "satan", "devil", "demon", "heaven", "hell", "paradise", "nirvana", "karma", "dharma", "enlightenment", "meditation", "prayer", "worship", "faith", "belief", "religion", "spiritual", "church", "mosque", "temple", "synagogue", "cathedral", "chapel", "altar", "cross", "crucifix", "star_of_david", "crescent", "om", "yin_yang", "bible", "quran", "torah", "vedas", "upanishads", "bhagavad_gita", "tripitaka", "pope", "cardinal", "bishop", "priest", "imam", "rabbi", "monk", "nun", "guru", "lama", "dalai_lama", "christian", "muslim", "jewish", "hindu", "buddhist", "sikh", "jain", "zoroastrian", "baptist", "catholic", "protestant", "orthodox", "sunni", "shia", "hasidic", "evangelical", "pentecostal", "methodist", "presbyterian", "episcopal", "lutheran", "anglican", "calvinist", "missionary", "pilgrimage", "hajj", "ramadan", "easter", "christmas", "passover", "diwali", "hanukkah", "sabbath", "sunday", "friday", "shabbat", "kosher", "halal", "communion", "baptism", "confirmation", "bar_mitzvah", "bat_mitzvah", "wedding", "funeral", "blessing", "miracle", "resurrection", "reincarnation", "afterlife", "soul", "spirit", "trinity", "incarnation", "salvation", "redemption", "forgiveness", "sin", "virtue", "grace", "mercy", "compassion", "love", "peace", "hope", "charity", "wisdom", "truth", "justice", "righteousness", "purity", "humility", "devotion", "reverence", "awe", "wonder", "mystery", "eternity", "infinity", "omnipotent", "omniscient", "omnipresent", "almighty", "creator", "father", "son", "mother", "blessed", "sanctified", "consecrated", "ordained", "anointed", "chosen", "covenant", "commandment", "scripture", "gospel", "revelation", "prophecy", "parable", "sermon", "hymn", "psalm", "chant", "litany", "rosary", "novena", "pillar", "foundation", "cornerstone", "shepherd", "flock", "lamb", "lion", "dove", "olive_branch", "ark", "covenant", "exodus", "genesis", "revelation", "apocalypse", "judgment", "rapture", "tribulation", "millennium", "armageddon", "zion", "jerusalem", "mecca", "medina", "vatican", "tibet", "ganges", "jordan", "sinai", "galilee", "nazareth", "bethlehem", "calvary", "golgotha", "gethsemane", "mount_olive", "mount_sinai", "mount_zion",

		// Major Organizations & Companies
		"google", "microsoft", "apple", "amazon", "facebook", "meta", "twitter", "x", "linkedin", "youtube", "instagram", "tiktok", "snapchat", "pinterest", "reddit", "discord", "slack", "zoom", "teams", "skype", "whatsapp", "telegram", "signal", "viber", "wechat", "line", "netflix", "disney", "hulu", "prime", "spotify", "pandora", "soundcloud", "twitch", "tinder", "bumble", "uber", "lyft", "airbnb", "booking", "expedia", "paypal", "venmo", "stripe", "square", "visa", "mastercard", "amex", "discover", "chase", "wells_fargo", "bank_of_america", "citibank", "goldman_sachs", "morgan_stanley", "jpmorgan", "blackrock", "vanguard", "fidelity", "charles_schwab", "robinhood", "etrade", "td_ameritrade", "coinbase", "binance", "kraken", "gemini", "bitfinex", "opensea", "uniswap", "chainlink", "polygon", "ethereum", "bitcoin", "dogecoin", "cardano", "solana", "avalanche", "polkadot", "algorand", "cosmos", "tezos", "stellar", "ripple", "litecoin", "monero", "zcash", "dash", "bitcoin_cash", "ethereum_classic", "binance_coin", "tether", "usd_coin", "dai", "maker", "compound", "aave", "yearn", "curve", "balancer", "synthetix", "0x", "kyber", "bancor", "loopring", "matic", "zilliqa", "enjin", "decentraland", "sandbox", "axie_infinity", "cryptokitties", "nba_topshot", "sorare", "gods_unchained", "splinterlands", "alien_worlds", "my_neighbor_alice", "star_atlas", "illuvium", "gala", "enjin", "chiliz", "flow", "wax", "hive", "steem", "eos", "tron", "neo", "ontology", "vechain", "iota", "nano", "hedera", "elrond", "near", "fantom", "harmony", "theta", "filecoin", "siacoin", "storj", "arweave", "helium", "internet_computer", "dfinity", "kusama", "moonbeam", "moonriver", "acala", "karura", "shiden", "astar", "clover", "manta", "calamari", "kilt", "centrifuge", "nodle", "bifrost", "crust", "darwinia", "edgeware", "subsocial", "phala", "khala", "litentry", "integritee", "ternoa", "unique", "rmrk", "singular", "kodadot", "nftmart", "unifty", "async", "foundation", "superrare", "rarible", "makersplace", "knownorigin", "nifty_gateway", "veve", "terra_virtua", "vulcanverse", "somnium_space", "cryptovoxels", "upland", "blankos", "f1_delta_time", "revv_racing", "polymarket", "augur", "gnosis", "kleros", "aragon", "dao", "moloch", "metacartel", "flamingo", "pleasr", "constitution", "people", "gitcoin", "grants", "quadratic", "funding", "retroactive", "public", "goods", "optimism", "arbitrum", "polygon", "avalanche", "fantom", "binance_smart_chain", "heco", "okex", "huobi", "kucoin", "gate", "bittrex", "bitfinex", "kraken", "bitstamp", "gemini", "celsius", "blockfi", "nexo", "crypto", "ledn", "hodlnaut", "vauld", "anchor", "maker", "compound", "aave", "cream", "venus", "benqi", "geist", "hundred", "radiant", "granary", "euler", "morpho", "notional", "pendle", "ribbon", "friktion", "katana", "tulip", "francium", "larix", "port", "jet", "mango", "drift", "zeta", "phoenix", "hedge", "entropy", "flash", "flash_loans", "liquidation", "collateral", "leverage", "margin", "futures", "options", "perpetual", "swap", "spot", "limit", "market", "stop", "loss", "take", "profit", "order", "book", "depth", "spread", "slippage", "arbitrage", "mev", "frontrun", "sandwich", "rug", "pull", "exit", "scam", "honeypot", "flash", "loan", "attack", "exploit", "hack", "bridge", "cross", "chain", "multichain", "anyswap", "synapse", "hop", "across", "connext", "celer", "stargate", "layerzero", "wormhole", "rainbow", "portal", "allbridge", "multichain", "router", "thorchain", "rune", "sifchain", "osmosis", "gravity", "dex", "injective", "serum", "raydium", "orca", "saber", "sunny", "quarry", "mercurial", "port", "socean", "marinade", "lido", "rocket", "pool", "frax", "liquity", "lusd", "fei", "tribe", "reflexer", "rai", "olympus", "ohm", "klima", "time", "memo", "wonderland", "abracadabra", "spell", "ice", "convex", "cvx", "vlcvx", "crv", "fxs", "vefxs", "vecrv", "gauge", "bribe", "votium", "hidden", "hand", "redacted", "cartel", "tokemak", "toke", "reactor", "dydx", "gains", "kwenta", "lyra", "premia", "ribbon", "friktion", "katana", "tulip", "mango", "drift", "zeta", "phoenix",

		// Historic Figures
		"napoleon", "alexander", "caesar", "cleopatra", "shakespeare", "leonardo", "michelangelo", "galileo", "newton", "einstein", "darwin", "columbus", "magellan", "washington", "lincoln", "roosevelt", "churchill", "gandhi", "mandela", "king", "luther", "confucius", "socrates", "plato", "aristotle", "homer", "virgil", "dante", "beethoven", "mozart", "bach", "picasso", "van_gogh", "da_vinci", "rembrandt", "monet", "renoir", "cezanne", "van_gogh", "goya", "velazquez", "el_greco", "rubens", "titian", "raphael", "donatello", "rodin", "henry", "elizabeth", "victoria", "louis", "charles", "william", "john", "paul", "peter", "james", "robert", "richard", "thomas", "george", "edward", "francis", "anthony", "stephen", "michael", "david", "joseph", "daniel", "samuel", "benjamin", "christopher", "matthew", "mark", "luke", "andrew", "simon", "philip", "bartholomew", "catherine", "margaret", "anne", "jane", "elizabeth", "mary", "sarah", "rebecca", "rachel", "leah", "ruth", "esther", "judith", "joan", "teresa", "agnes", "barbara", "dorothy", "helen", "betty", "patricia", "linda", "susan", "karen", "nancy", "lisa", "sandra", "donna", "carol", "ruth", "sharon", "michelle", "laura", "sarah", "kimberly", "deborah", "dorothy", "lisa", "nancy", "karen", "betty", "helen", "sandra", "donna", "carol", "ruth", "sharon", "michelle", "laura", "sarah", "kimberly", "deborah", "amy", "angela", "brenda", "emma", "olivia", "sophia", "isabella", "charlotte", "amelia", "harper", "evelyn", "abigail", "emily", "elizabeth", "mila", "ella", "avery", "sofia", "camila", "aria", "scarlett", "victoria", "madison", "luna", "grace", "chloe", "penelope", "layla", "riley", "zoey", "nora", "lily", "eleanor", "hannah", "lillian", "addison", "aubrey", "ellie", "stella", "natalie", "zoe", "leah", "hazel", "violet", "aurora", "savannah", "audrey", "brooklyn", "bella", "claire", "skylar", "lucy", "paisley", "everly", "anna", "caroline", "nova", "genesis", "emilia", "kennedy", "samantha", "maya", "willow", "kinsley", "naomi", "aaliyah", "elena", "sarah", "ariana", "allison", "gabriella", "alice", "madelyn", "cora", "ruby", "eva", "serenity", "autumn", "adeline", "hailey", "gianna", "valentina", "isla", "eliana", "quinn", "nevaeh", "ivy", "sadie", "piper", "lydia", "alexa", "josephine", "emery", "julia", "delilah", "arianna", "vivian", "kaylee", "sophie", "brielle", "madeline", "peyton", "rylee", "clara", "hadley", "melanie", "mackenzie", "reagan", "adalynn", "liliana", "maria", "lilly", "alexandra", "denver", "london", "brooklyn", "paris", "milan", "sydney", "tokyo", "berlin", "madrid", "rome", "vienna", "prague", "budapest", "warsaw", "stockholm", "oslo", "copenhagen", "helsinki", "dublin", "edinburgh", "glasgow", "manchester", "liverpool", "birmingham", "leeds", "sheffield", "bristol", "cardiff", "belfast", "york", "bath", "oxford", "cambridge", "canterbury", "winchester", "salisbury", "exeter", "plymouth", "brighton", "portsmouth", "southampton", "nottingham", "leicester", "coventry", "hull", "stoke", "wolverhampton", "derby", "preston", "bradford", "oldham", "sunderland", "middlesbrough", "huddersfield", "blackpool", "bolton", "stockport", "rochdale", "rotherham", "luton", "northampton", "southend", "warrington", "crawley", "ipswich", "wigan", "mansfield", "oxford", "cambridge", "reading", "slough", "woking", "watford", "basildon", "burnley", "hastings", "scunthorpe", "gloucester", "carlisle", "worcester", "lancaster", "st_albans", "hereford", "salisbury", "chester", "chichester", "durham", "ely", "exeter", "gloucester", "hereford", "leicester", "lichfield", "lincoln", "norwich", "peterborough", "portsmouth", "ripon", "rochester", "st_albans", "st_davids", "truro", "wakefield", "wells", "winchester", "worcester", "york", "bangor", "newport", "st_asaph", "st_davids", "swansea", "armagh", "belfast", "derry", "down", "kilmore", "raphoe", "tuam", "cork", "dublin", "galway", "kerry", "kildare", "kilkenny", "laois", "leitrim", "limerick", "longford", "louth", "mayo", "meath", "monaghan", "offaly", "roscommon", "sligo", "tipperary", "waterford", "westmeath", "wexford", "wicklow", "carlow", "cavan", "clare", "donegal",

		// Countries and Major Cities
		"afghanistan", "albania", "algeria", "andorra", "angola", "antigua", "argentina", "armenia", "australia", "austria", "azerbaijan", "bahamas", "bahrain", "bangladesh", "barbados", "belarus", "belgium", "belize", "benin", "bhutan", "bolivia", "bosnia", "botswana", "brazil", "brunei", "bulgaria", "burkina", "burundi", "cambodia", "cameroon", "canada", "cape_verde", "central_african_republic", "chad", "chile", "china", "colombia", "comoros", "congo", "costa_rica", "croatia", "cuba", "cyprus", "czech", "denmark", "djibouti", "dominica", "dominican", "ecuador", "egypt", "el_salvador", "equatorial_guinea", "eritrea", "estonia", "eswatini", "ethiopia", "fiji", "finland", "france", "gabon", "gambia", "georgia", "germany", "ghana", "greece", "grenada", "guatemala", "guinea", "guyana", "haiti", "honduras", "hungary", "iceland", "india", "indonesia", "iran", "iraq", "ireland", "israel", "italy", "jamaica", "japan", "jordan", "kazakhstan", "kenya", "kiribati", "kosovo", "kuwait", "kyrgyzstan", "laos", "latvia", "lebanon", "lesotho", "liberia", "libya", "liechtenstein", "lithuania", "luxembourg", "madagascar", "malawi", "malaysia", "maldives", "mali", "malta", "marshall", "mauritania", "mauritius", "mexico", "micronesia", "moldova", "monaco", "mongolia", "montenegro", "morocco", "mozambique", "myanmar", "namibia", "nauru", "nepal", "netherlands", "new_zealand", "nicaragua", "niger", "nigeria", "north_korea", "north_macedonia", "norway", "oman", "pakistan", "palau", "palestine", "panama", "papua_new_guinea", "paraguay", "peru", "philippines", "poland", "portugal", "qatar", "romania", "russia", "rwanda", "saint_kitts", "saint_lucia", "saint_vincent", "samoa", "san_marino", "sao_tome", "saudi_arabia", "senegal", "serbia", "seychelles", "sierra_leone", "singapore", "slovakia", "slovenia", "solomon", "somalia", "south_africa", "south_korea", "south_sudan", "spain", "sri_lanka", "sudan", "suriname", "sweden", "switzerland", "syria", "taiwan", "tajikistan", "tanzania", "thailand", "timor", "togo", "tonga", "trinidad", "tunisia", "turkey", "turkmenistan", "tuvalu", "uganda", "ukraine", "united_arab_emirates", "united_kingdom", "united_states", "uruguay", "uzbekistan", "vanuatu", "vatican", "venezuela", "vietnam", "yemen", "zambia", "zimbabwe", "new_york", "los_angeles", "chicago", "houston", "phoenix", "philadelphia", "san_antonio", "san_diego", "dallas", "san_jose", "austin", "jacksonville", "fort_worth", "columbus", "charlotte", "san_francisco", "indianapolis", "seattle", "denver", "washington", "boston", "el_paso", "detroit", "nashville", "memphis", "portland", "oklahoma_city", "las_vegas", "louisville", "baltimore", "milwaukee", "albuquerque", "tucson", "fresno", "mesa", "sacramento", "atlanta", "kansas_city", "colorado_springs", "miami", "raleigh", "omaha", "long_beach", "virginia_beach", "oakland", "minneapolis", "tulsa", "arlington", "tampa", "new_orleans", "wichita", "cleveland", "bakersfield", "aurora", "anaheim", "honolulu", "santa_ana", "riverside", "corpus_christi", "lexington", "stockton", "henderson", "saint_paul", "st_louis", "cincinnati", "pittsburgh", "greensboro", "lincoln", "plano", "anchorage", "orlando", "irvine", "newark", "durham", "chula_vista", "toledo", "fort_wayne", "st_petersburg", "laredo", "jersey_city", "chandler", "madison", "lubbock", "scottsdale", "reno", "buffalo", "gilbert", "glendale", "north_las_vegas", "winston_salem", "chesapeake", "norfolk", "fremont", "garland", "irving", "hialeah", "richmond", "boise", "spokane", "baton_rouge",
	}
	for _, r := range reserved {
		if strings.EqualFold(username, r) {
			return ErrInvalidUsernameLength
		}
	}
	return nil
}

// structToMap converts a protobuf Struct to a Go map.
func structToMap(s *structpb.Struct) map[string]interface{} {
	if s == nil {
		return nil
	}
	return s.AsMap()
}

// mapToStruct converts a Go map to a protobuf Struct.
func mapToStruct(m map[string]interface{}) *structpb.Struct {
	if m == nil {
		return nil
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		// Optionally log or handle error
		return nil
	}
	return s
}

// metadataToMap converts protobuf Metadata to a Go map.
func metadataToMap(meta *commonpb.Metadata) map[string]interface{} {
	if meta == nil {
		return nil
	}

	result := make(map[string]interface{})

	if meta.ServiceSpecific != nil {
		result["service_specific"] = meta.ServiceSpecific.AsMap()
	}
	if len(meta.Features) > 0 {
		result["features"] = meta.Features
	}
	if meta.CustomRules != nil {
		result["custom_rules"] = meta.CustomRules.AsMap()
	}
	if meta.Audit != nil {
		result["audit"] = meta.Audit.AsMap()
	}
	if len(meta.Tags) > 0 {
		result["tags"] = meta.Tags
	}

	return result
}

// mapToMetadata converts a Go map to protobuf Metadata.
func mapToMetadata(m map[string]interface{}) *commonpb.Metadata {
	if m == nil {
		return nil
	}

	meta := &commonpb.Metadata{}

	if serviceSpecific, ok := m["service_specific"].(map[string]interface{}); ok {
		s, err := structpb.NewStruct(serviceSpecific)
		if err == nil {
			meta.ServiceSpecific = s
		}
	}
	if features, ok := m["features"].([]string); ok {
		meta.Features = features
	}
	if customRules, ok := m["custom_rules"].(map[string]interface{}); ok {
		s, err := structpb.NewStruct(customRules)
		if err == nil {
			meta.CustomRules = s
		}
	}
	if audit, ok := m["audit"].(map[string]interface{}); ok {
		s, err := structpb.NewStruct(audit)
		if err == nil {
			meta.Audit = s
		}
	}
	if tags, ok := m["tags"].([]string); ok {
		meta.Tags = tags
	}

	return meta
}

// timestampProto converts a time.Time to a protobuf Timestamp.
func timestampProto(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}
