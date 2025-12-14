package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"
	"strings"

	tb "gopkg.in/telebot.v3"
	"gopkg.in/yaml.v3"
)

const (
	dataFile   = "data.json"
	configFile = "config.yaml"
)

// Config holds bot token, topic, keywords and debug flag
type Config struct {
	BotToken string   `yaml:"bot_token"`
	Topic    string   `yaml:"topic"`
	Keywords []string `yaml:"keywords"`
	NoSuffix []string `yaml:"no_suffix"`
	Debug    bool     `yaml:"debug"`
}

// Storage represents persistent storage for the last mention timestamp
type Storage struct {
	LastMention time.Time `json:"last_mention"`
}

var isDebug bool

func debugLog(format string, v ...any) {
	if isDebug {
		log.Printf("[DEBUG] "+format, v...)
	}
}

func loadConfig() Config {
	log.Println("[INFO] Loading config.yaml...")
	var cfg Config
	file, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("[ERROR] Failed to read %s: %v", configFile, err)
	}
	err = yaml.Unmarshal(file, &cfg)
	if err != nil {
		log.Fatalf("[ERROR] Failed to parse %s: %v", configFile, err)
	}
	if len(cfg.Keywords) == 0 {
	  log.Fatal("[ERROR] keywords is empty in config.yaml")
	}
	log.Printf("[INFO] Config loaded: topic=%q, keywords=%d, debug=%v", cfg.Topic, len(cfg.Keywords), cfg.Debug)
	return cfg
}

func loadStorage() Storage {
	debugLog("Loading storage from data.json...")
	var s Storage
	file, err := os.ReadFile(dataFile)
	if err != nil {
		log.Println("[WARN] No data.json found, starting fresh")
		s.LastMention = time.Time{}
		return s
	}
	err = json.Unmarshal(file, &s)
	if err != nil {
		log.Printf("[ERROR] Failed to parse data.json: %v", err)
		s.LastMention = time.Time{}
	}
	if s.LastMention.IsZero() {
		debugLog("Storage loaded: no last mention recorded")
	} else {
		debugLog("Storage loaded: lastMention=%s", s.LastMention.Format(time.RFC3339))
	}
	return s
}

func saveStorage(s Storage) {
	debugLog("Saving storage: lastMention=%s", s.LastMention.Format(time.RFC3339))
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Printf("[ERROR] Failed to serialize JSON: %v", err)
		return
	}
	err = os.WriteFile(dataFile, data, 0644)
	if err != nil {
		log.Printf("[ERROR] Failed to write data.json: %v", err)
	}
}

func compileRegexps(patterns []string) []*regexp.Regexp {
	debugLog("Compiling regexps...")
	var regs []*regexp.Regexp
	for _, p := range patterns {
		debugLog("Compiling regexp: %s", p)
		r, err := regexp.Compile(p)
		if err != nil {
			log.Fatalf("[ERROR] Failed to compile regexp %q: %v", p, err)
		}
		regs = append(regs, r)
	}
	return regs
}

func buildKeywordRegex(words []string, noSuffix []string) *regexp.Regexp {
	const leftBoundary = `(?:^|[^\p{L}\p{N}_])`
	const rightBoundary = `(?:$|[^\p{L}\p{N}_])`

	noSuffixSet := make(map[string]bool)
	for _, w := range noSuffix {
		noSuffixSet[strings.ToLower(strings.TrimSpace(w))] = true
	}

	var parts []string

	for _, w := range words {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}

		key := strings.ToLower(w)

		quoted := regexp.QuoteMeta(w)

		quoted = regexp.MustCompile(`\\ +`).ReplaceAllString(quoted, `\s+`)

		suffix := `[\p{L}\p{N}_]*`
		if noSuffixSet[key] {
			suffix = ``
		}

		parts = append(parts, fmt.Sprintf(`(?:%s)%s`, quoted, suffix))
	}

	pattern := `(?i)` + leftBoundary + `(` + strings.Join(parts, `|`) + `)` + rightBoundary
	return regexp.MustCompile(pattern)
}

func findKeyword(text string, re *regexp.Regexp) string {
	m := re.FindStringSubmatch(text)
	if len(m) >= 2 && m[1] != "" {
		debugLog("Keyword matched: %q in message=%q", m[1], text)
		return m[1]
	}
	debugLog("No keyword matched in message=%q", text)
	return ""
}

func main() {
	cfg := loadConfig()
	isDebug = cfg.Debug

	keywordRe := buildKeywordRegex(cfg.Keywords, cfg.NoSuffix)

	storage := loadStorage()

	pref := tb.Settings{
		Token:  cfg.BotToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	}

	log.Println("[INFO] Initializing bot...")
	b, err := tb.NewBot(pref)
	if err != nil {
		log.Fatalf("[FATAL] Failed to init bot: %v", err)
	}

	log.Printf("[INFO] Authorized as @%s (id=%d)", b.Me.Username, b.Me.ID)

	// Handle /days
	b.Handle("/days", func(c tb.Context) error {
		log.Printf("[INFO] Command /days from user=%s chat=%d", c.Sender().Username, c.Chat().ID)
		if storage.LastMention.IsZero() {
			return c.Send(fmt.Sprintf("Ещё ни разу не упоминали '%s'.", cfg.Topic))
		}
		days := int(time.Since(storage.LastMention).Hours() / 24)
		text := fmt.Sprintf(
			"%d дней без упоминания %s.\nПоследнее упоминание было: %s",
			days, cfg.Topic, storage.LastMention.Format("02.01.2006 15:04:05"),
		)
		return c.Send(text)
	})

	// Handle /reset
	b.Handle("/reset", func(c tb.Context) error {
		log.Printf("[INFO] Command /reset from user=%s chat=%d", c.Sender().Username, c.Chat().ID)
		
		// previous mention info
		prevLastMention := storage.LastMention
		prevText := "никогда"
		if !prevLastMention.IsZero() {
		  prevText = prevLastMention.Format("02.01.2006 15:04:05")
		}
		daysWas := 0
		if !prevLastMention.IsZero() {
		  daysWas = int(time.Since(prevLastMention).Hours() / 24)
		}
		
		storage.LastMention = time.Now()
		saveStorage(storage)
		text := fmt.Sprintf("Кто-то что-то написал про %s %s 💀💀💀 запомнили, мы продержались %d дней.\nПоследнее упоминание до этого было: %s",
			cfg.Topic, storage.LastMention.Format("02.01.2006 15:04:05"), daysWas, prevText,
		)
		return c.Send(text)
	})

	// Handle all text messages
	b.Handle(tb.OnText, func(c tb.Context) error {
		msg := c.Message()
		debugLog("New text message in chat=%d from=%s text=%q", msg.Chat.ID, msg.Sender.Username, msg.Text)

		found := findKeyword(msg.Text, keywordRe)
		if found != "" {
			if !storage.LastMention.IsZero() && time.Since(storage.LastMention) < 2*time.Hour {
				debugLog("Ignoring mention, lastMention=%s (<2h ago)", storage.LastMention.Format(time.RFC3339))
				return nil
			}
			response := fmt.Sprintf(
				"Кто-то сказал «%s»?\nСбросить счётчик дней без %s? Используйте /reset для подтверждения.",
				found, cfg.Topic,
			)
			log.Printf("[INFO] Triggered by keyword=%q in chat=%d", found, msg.Chat.ID)
			return c.Send(response)
		}
		return nil
	})

	log.Println("[INFO] Bot started, waiting for updates...")
	b.Start()
}
