package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// BotConfig ...
type Config struct {
	TelegramBotToken string
	AllowedChatIDs   map[int64]bool
	MaxItems         int
	IgnoreTags       bool
	RadarrProtocol   string
	RadarrHostname   string
	RadarrPort       int
	RadarrAPIKey     string
	RadarrBaseUrl    string
	RadarrUsername   string
	RadarrPassword   string
}

func LoadConfig() (Config, error) {
	var config Config

	config.TelegramBotToken = os.Getenv("RBOT_TELEGRAM_BOT_TOKEN")
	allowedUserIDs := os.Getenv("RBOT_BOT_ALLOWED_USERIDS")
	botMaxItems := os.Getenv("RBOT_BOT_MAX_ITEMS")
	botIgnoreTags := os.Getenv("RBOT_BOT_IGNORE_TAGS")
	config.RadarrProtocol = os.Getenv("RBOT_RADARR_PROTOCOL")
	config.RadarrHostname = os.Getenv("RBOT_RADARR_HOSTNAME")
	radarrPort := os.Getenv("RBOT_RADARR_PORT")
	config.RadarrAPIKey = os.Getenv("RBOT_RADARR_API_KEY")
	config.RadarrBaseUrl = os.Getenv("RBOT_RADARR_BASE_URL")
	config.RadarrUsername = os.Getenv("RBOT_RADARR_USERNAME")
	Config.RadarrPassword = os.Getenv("RBOT_RADARR_PASSWORD")

	// Validate required fields
	if config.TelegramBotToken == "" {
		return config, errors.New("RBOT_TELEGRAM_BOT_TOKEN is empty or not set")
	}
	if allowedUserIDs == "" {
		return config, errors.New("RBOT_BOT_ALLOWED_USERIDS is empty or not set")
	}
	if botMaxItems == "" {
		return config, errors.New("RBOT_BOT_MAX_ITEMS is empty or not set")
	}
	if botIgnoreTags == "" {
		return config, errors.New("RBOT_BOT_IGNORE_TAGS is empty or not set")
	}
	// Normalize and validate RBOT_RADARR_PROTOCOL
	config.RadarrProtocol = strings.ToLower(config.RadarrProtocol)
	if config.RadarrProtocol != "http" && config.RadarrProtocol != "https" {
		return config, errors.New("RBOT_RADARR_PROTOCOL must be http or https")
	}
	if config.RadarrHostname == "" {
		return config, errors.New("RBOT_RADARR_HOSTNAME is empty or not set")
	}
	if config.RadarrUsername == "" {
		return Config, errors.New("RBOT_RADARR_USERNAME is empty or not set")	
	}
	if config.RadarrPassword == "" {
		return Config, errors.New("RBOT_RADARR_PASSWORD is empty or not set")
	if radarrPort == "" {
		return config, errors.New("RBOT_RADARR_PORT is empty or not set")
	}
	if config.RadarrAPIKey == "" {
		return config, errors.New("RBOT_RADARR_API_KEY is empty or not set")
	}

	// Parsing RBOT_BOT_MAX_ITEMS as a number
	maxItems, err := strconv.Atoi(botMaxItems)
	if err != nil {
		return config, errors.New("RBOT_BOT_MAX_ITEMS is not a valid number")
	}
	config.MaxItems = maxItems

	// Parsing RBOT_BOT_IGNORE_TAGS as a boolean
	ignoreTags, err := strconv.ParseBool(botIgnoreTags)
	if err != nil {
		return config, errors.New("RBOT_BOT_IGNORE_TAGS is not a valid boolean")
	}
	config.IgnoreTags = ignoreTags

	// Parsing RBOT_BOT_ALLOWED_USERIDS as a list of integers
	userIDs := strings.Split(allowedUserIDs, ",")
	parsedUserIDs := make(map[int64]bool)
	for _, id := range userIDs {
		parsedID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return config, fmt.Errorf("RBOT_BOT_ALLOWED_USERIDS contains non-integer value: %s", err)
		}
		parsedUserIDs[parsedID] = true
	}
	config.AllowedChatIDs = parsedUserIDs

	// Parsing RBOT_RADARR_PORT as a number
	port, err := strconv.Atoi(radarrPort)
	if err != nil {
		return config, errors.New("RBOT_RADARR_PORT is not a valid number")
	}
	config.RadarrPort = port

	return config, nil
}
