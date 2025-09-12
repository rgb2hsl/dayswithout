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
	var cfg Config
	file, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to read %s: %v", configFile, err)
	}
	err = yaml.Unmarshal(file, &cfg)
	if err != nil {
		log.Fatalf("Failed to parse %s: %v", configFile, err)
	}
	if cfg.BotToken == "" {
		log.Fatal("bot_token is missing in config.yaml")
	}
	if cfg.Topic == "" {
		log.Fatal("topic is missing in config.yaml")
	}
	if len(cfg.Keywords) == 0 {
		log.Fatal("keywords list is missing in config.yaml")
	}
	return cfg
}

// loadStorage reads the last mention time from data.json
func loadStorage() Storage {
	var s Storage
	file, err := os.ReadFile(dataFile)
	if err != nil {
		s.LastMention = time.Time{}
		return s
	}
	err = json.Unmarshal(file, &s)
	if err != nil {
		log.Printf("Failed to parse JSON: %v", err)
		s.LastMention = time.Time{}
	}
	return s
}

// saveStorage writes the last mention time to data.json
func saveStorage(s Storage) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Printf("Failed to serialize JSON: %v", err)
		return
	}
	err = os.WriteFile(dataFile, data, 0644)
	if err != nil {
		log.Printf("Failed to write JSON: %v", err)
	}
}

// compileRegexps compiles keyword patterns into regex objects
func compileRegexps(patterns []string) []*regexp.Regexp {
	var regs []*regexp.Regexp
	for _, p := range patterns {
		r, err := regexp.Compile(p)
		if err != nil {
			log.Fatalf("Failed to compile regexp %q: %v", p, err)
		}
		regs = append(regs, r)
	}
	return regs
}

// findKeyword searches for the first matching keyword in a text
func findKeyword(text string, regs []*regexp.Regexp) string {
	for _, r := range regs {
		if r.MatchString(text) {
			return r.FindString(text)
		}
	}
	return ""
}

func main() {
	cfg := loadConfig()
	regs := compileRegexps(cfg.Keywords)

	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false
	log.Printf("Authorized as %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	storage := loadStorage()

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		textMsg := update.Message.Text

		// --- Handle commands ---
		switch update.Message.Command() {
		case "days":
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
			storage.LastMention = time.Now()
			saveStorage(storage)
			text := fmt.Sprintf("Счётчик обнулён. Последнее упоминание '%s' записано: %s",
				cfg.Topic, storage.LastMention.Format("02.01.2006 15:04:05"))
			bot.Send(tgbotapi.NewMessage(chatID, text))
			continue
		}

		// --- Handle regular messages ---
		found := findKeyword(textMsg, regs)
		if found != "" {
			// Ignore if last mention was less than 2 hours ago
			if !storage.LastMention.IsZero() && time.Since(storage.LastMention) < 2*time.Hour {
				continue
			}

			response := fmt.Sprintf(
				"Обнаружено упоминание «%s».\nСбросить счётчик '%s'? Используйте /reset для подтверждения.",
				found, cfg.Topic,
			)
			bot.Send(tgbotapi.NewMessage(chatID, response))
		}
	}
}
