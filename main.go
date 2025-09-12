package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	tb "gopkg.in/telebot.v3"
	"gopkg.in/yaml.v3"
)

const (
	dataFile   = "data.json"
	configFile = "config.yaml"
)

// Config holds bot token, topic and keywords from config.yaml
type Config struct {
	BotToken string   `yaml:"token"`
	Topic    string   `yaml:"topic"`
	Keywords []string `yaml:"keywords"`
}

// Storage represents persistent storage for the last mention timestamp
type Storage struct {
	LastMention time.Time `json:"last_mention"`
}

func loadConfig() Config {
	log.Println("[DEBUG] Loading config.yaml...")
	var cfg Config
	file, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("[ERROR] Failed to read %s: %v", configFile, err)
	}
	err = yaml.Unmarshal(file, &cfg)
	if err != nil {
		log.Fatalf("[ERROR] Failed to parse %s: %v", configFile, err)
	}
	log.Printf("[DEBUG] Config loaded: topic=%q, keywords=%d", cfg.Topic, len(cfg.Keywords))
	return cfg
}

func loadStorage() Storage {
	log.Println("[DEBUG] Loading storage from data.json...")
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
		log.Println("[DEBUG] Storage loaded: no last mention recorded")
	} else {
		log.Printf("[DEBUG] Storage loaded: lastMention=%s", s.LastMention.Format(time.RFC3339))
	}
	return s
}

func saveStorage(s Storage) {
	log.Printf("[DEBUG] Saving storage: lastMention=%s", s.LastMention.Format(time.RFC3339))
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
	log.Println("[DEBUG] Compiling regexps...")
	var regs []*regexp.Regexp
	for _, p := range patterns {
		log.Printf("[DEBUG] Compiling regexp: %s", p)
		r, err := regexp.Compile(p)
		if err != nil {
			log.Fatalf("[ERROR] Failed to compile regexp %q: %v", p, err)
		}
		regs = append(regs, r)
	}
	return regs
}

func findKeyword(text string, regs []*regexp.Regexp) string {
	for _, r := range regs {
		if r.MatchString(text) {
			match := r.FindString(text)
			log.Printf("[DEBUG] Keyword matched: %q in message=%q", match, text)
			return match
		}
	}
	log.Printf("[DEBUG] No keyword matched in message=%q", text)
	return ""
}

func main() {
	cfg := loadConfig()
	regs := compileRegexps(cfg.Keywords)

	storage := loadStorage()

	pref := tb.Settings{
		Token:  cfg.BotToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	}

	log.Println("[DEBUG] Initializing bot...")
	b, err := tb.NewBot(pref)
	if err != nil {
		log.Fatalf("[FATAL] Failed to init bot: %v", err)
	}

	log.Printf("[INFO] Authorized as @%s (id=%d)", b.Me.Username, b.Me.ID)

	// Handle /days
	b.Handle("/days", func(c tb.Context) error {
		log.Printf("[DEBUG] Command /days from user=%s chat=%d", c.Sender().Username, c.Chat().ID)
		if storage.LastMention.IsZero() {
			return c.Send(fmt.Sprintf("Ещё ни разу не упоминали '%s'.", cfg.Topic))
		}
		days := int(time.Since(storage.LastMention).Hours() / 24)
		text := fmt.Sprintf(
			"С последнего упоминания %s прошло %d дней.\nПоследнее упоминание было: %s",
			cfg.Topic, days, storage.LastMention.Format("02.01.2006 15:04:05"),
		)
		return c.Send(text)
	})

	// Handle /reset
	b.Handle("/reset", func(c tb.Context) error {
		log.Printf("[DEBUG] Command /reset from user=%s chat=%d", c.Sender().Username, c.Chat().ID)
		storage.LastMention = time.Now()
		saveStorage(storage)
		text := fmt.Sprintf("Кто-то что-то написал про %s %s 💀💀💀 запомнили",
			cfg.Topic, storage.LastMention.Format("02.01.2006 15:04:05"))
		return c.Send(text)
	})

	// Handle all text messages
	b.Handle(tb.OnText, func(c tb.Context) error {
		msg := c.Message()
		log.Printf("[DEBUG] New text message in chat=%d from=%s text=%q",
			msg.Chat.ID, msg.Sender.Username, msg.Text)

		found := findKeyword(msg.Text, regs)
		if found != "" {
			// Ignore if last mention was less than 2 hours ago
			if !storage.LastMention.IsZero() && time.Since(storage.LastMention) < 2*time.Hour {
				log.Printf("[DEBUG] Ignoring mention, lastMention=%s (<2h ago)",
					storage.LastMention.Format(time.RFC3339))
				return nil
			}
			response := fmt.Sprintf(
				"Кто-то сказал «%s»?\nСбросить счётчик дней без %s? Используйте /reset для подтверждения.",
				found, cfg.Topic,
			)
			log.Printf("[DEBUG] Sending trigger message to chat=%d", msg.Chat.ID)
			return c.Send(response)
		}
		return nil
	})

	log.Println("[INFO] Bot started, waiting for updates...")
	b.Start()
}
