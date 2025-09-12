package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gopkg.in/yaml.v3"
)

const (
	dataFile   = "data.json"
	configFile = "config.yaml"
)

// Config holds bot token, topic and keywords from config.yaml
type Config struct {
	BotToken string   `yaml:"bot_token"`
	Topic    string   `yaml:"topic"`
	Keywords []string `yaml:"keywords"`
}

// Storage represents persistent storage for the last mention timestamp
type Storage struct {
	LastMention time.Time `json:"last_mention"`
}

// loadConfig reads and parses config.yaml
func loadConfig() Config {
	log.Println("[DEBUG] Loading config...")
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

// loadStorage reads the last mention time from data.json
func loadStorage() Storage {
	log.Println("[DEBUG] Loading storage...")
	var s Storage
	file, err := os.ReadFile(dataFile)
	if err != nil {
		log.Println("[WARN] No data.json found, starting fresh")
		s.LastMention = time.Time{}
		return s
	}
	err = json.Unmarshal(file, &s)
	if err != nil {
		log.Printf("[ERROR] Failed to parse JSON: %v", err)
		s.LastMention = time.Time{}
	}
	if s.LastMention.IsZero() {
		log.Println("[DEBUG] Storage loaded: no last mention recorded")
	} else {
		log.Printf("[DEBUG] Storage loaded: lastMention=%s", s.LastMention.Format(time.RFC3339))
	}
	return s
}

// saveStorage writes the last mention time to data.json
func saveStorage(s Storage) {
	log.Printf("[DEBUG] Saving storage: lastMention=%s", s.LastMention.Format(time.RFC3339))
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Printf("[ERROR] Failed to serialize JSON: %v", err)
		return
	}
	err = os.WriteFile(dataFile, data, 0644)
	if err != nil {
		log.Printf("[ERROR] Failed to write JSON: %v", err)
	}
}

// compileRegexps compiles keyword patterns into regex objects
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

// findKeyword searches for the first matching keyword in a text
func findKeyword(text string, regs []*regexp.Regexp) string {
	for _, r := range regs {
		if r.MatchString(text) {
			log.Printf("[DEBUG] Keyword match: %q in message", r.FindString(text))
			return r.FindString(text)
		}
	}
	return ""
}

func main() {
	cfg := loadConfig()
	regs := compileRegexps(cfg.Keywords)

	log.Println("[DEBUG] Initializing bot...")
	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Panicf("[FATAL] Failed to init bot: %v", err)
	}

	bot.Debug = false
	log.Printf("[INFO] Authorized as %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	log.Println("[DEBUG] Starting updates channel...")
	updates := bot.GetUpdatesChan(u)

	storage := loadStorage()

	log.Println("[INFO] Bot is running and waiting for updates...")

	for update := range updates {
		if update.Message == nil {
			log.Println("[DEBUG] Update received, but no message -> skipping")
			continue
		}

		chatID := update.Message.Chat.ID
		textMsg := update.Message.Text

		log.Printf("[DEBUG] New message in chat=%d from=%s text=%q",
			chatID,
			update.Message.From.UserName,
			textMsg,
		)

		// --- Handle commands ---
		switch update.Message.Command() {
		case "days":
			log.Println("[DEBUG] Command /days triggered")
			if storage.LastMention.IsZero() {
				bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Ещё ни разу не упоминали '%s'.", cfg.Topic)))
				continue
			}
			days := int(time.Since(storage.LastMention).Hours() / 24)
			text := fmt.Sprintf(
				"С последнего упоминания '%s' прошло %d дней.\nПоследнее упоминание было: %s",
				cfg.Topic, days, storage.LastMention.Format("02.01.2006 15:04:05"),
			)
			bot.Send(tgbotapi.NewMessage(chatID, text))

		case "reset":
			log.Println("[DEBUG] Command /reset triggered")
			storage.LastMention = time.Now()
			saveStorage(storage)
			text := fmt.Sprintf("Счётчик обнулён. Последнее упоминание '%s' записано: %s",
				cfg.Topic, storage.LastMention.Format("02.01.2006 15:04:05"))
			bot.Send(tgbotapi.NewMessage(chatID, text))
			continue
		default:
			if update.Message.IsCommand() {
				log.Printf("[DEBUG] Unknown command: %s", update.Message.Command())
			}
		}

		// --- Handle regular messages ---
		found := findKeyword(textMsg, regs)
		if found != "" {
			log.Printf("[DEBUG] Keyword trigger found: %q", found)

			// Ignore if last mention was less than 2 hours ago
			if !storage.LastMention.IsZero() && time.Since(storage.LastMention) < 2*time.Hour {
				log.Printf("[DEBUG] Ignoring mention, last was %s (<2h ago)",
					storage.LastMention.Format(time.RFC3339))
				continue
			}

			response := fmt.Sprintf(
				"Обнаружено упоминание «%s».\nСбросить счётчик '%s'? Используйте /reset для подтверждения.",
				found, cfg.Topic,
			)
			bot.Send(tgbotapi.NewMessage(chatID, response))
		} else {
			log.Println("[DEBUG] No keywords found in message")
		}
	}
}
