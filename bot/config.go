package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	// "github.com/joho/godotenv"

	api "github.com/bigbluebutton-bot/bigbluebutton-bot/api"
)

// Settings holds the configuration settings
type Settings struct {
	Bot struct {
		Limit int
	}
	BBB struct {
		API struct {
			URL    string
			Secret string
			SHA    api.SHA
		}
		Client struct {
			URL string
			WS  string
		}
		Pad struct {
			URL string
			WS  string
		}
		WebRTC struct {
			WS string
		}
	}
	ChangeSet struct {
		External bool
		Host     string
		Port     int
	}
	TranscriptionServer struct {
		ExternalHost    string
		PortTCP         int
		Secret          string
		HealthCheckPort int
	}
	TranslationServer struct {
		URL    string
		Secret string
	}
}

// LoadSettings loads and validates the configuration settings from environment variables.
// It attempts to load variables from a .env file if present and returns an error if required
// variables are missing or invalid.
func LoadSettings() (*Settings, error) {
	// Attempt to load from .env, but continue if not present or invalid
	// if err := godotenv.Overload(); err != nil {
	// 	log.Printf("Warning: could not load .env file: %v", err)
	// }

	var (
		errs []string
		cfg  *Settings = &Settings{}
	)

	// mustString retrieves a string value for a required key.
	// If the value is missing, it records an error.
	mustString := func(key string) string {
		val, ok := os.LookupEnv(key)
		if !ok || val == "" {
			errs = append(errs, fmt.Sprintf("%s is required but not set", key))
		}
		return val
	}

	// mustInt retrieves an integer value for a required key.
	// If the value is missing or not an integer, it records an error.
	mustInt := func(key string) int {
		strVal := mustString(key)
		if strVal == "" {
			return 0
		}
		numVal, err := strconv.Atoi(strVal)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s must be an integer (got: %q)", key, strVal))
		}
		return numVal
	}

	// Assign all settings
	cfg.Bot.Limit = mustInt("BOT_LIMIT")

	cfg.BBB.API.URL = mustString("BBB_API_URL")
	cfg.BBB.API.Secret = mustString("BBB_API_SECRET")
	cfg.BBB.API.SHA = api.SHA(mustString("BBB_API_SHA"))

	cfg.BBB.Client.URL = mustString("BBB_CLIENT_URL")
	cfg.BBB.Client.WS = mustString("BBB_CLIENT_WS")

	cfg.BBB.Pad.URL = mustString("BBB_PAD_URL")
	cfg.BBB.Pad.WS = mustString("BBB_PAD_WS")

	cfg.BBB.WebRTC.WS = mustString("BBB_WEBRTC_WS")

	cfg.ChangeSet.External = mustString("CHANGESET_EXTERNAL") == "true"
	cfg.ChangeSet.Host = mustString("CHANGESET_HOST")
	cfg.ChangeSet.Port = mustInt("CHANGESET_PORT")

	cfg.TranscriptionServer.ExternalHost = mustString("TRANSCRIPTION_SERVER_EXTERNAL_HOST")
	cfg.TranscriptionServer.PortTCP = mustInt("TRANSCRIPTION_SERVER_PORT_TCP")
	cfg.TranscriptionServer.Secret = mustString("TRANSCRIPTION_SERVER_SECRET")
	cfg.TranscriptionServer.HealthCheckPort = mustInt("TRANSCRIPTION_SERVER_HEALTH_CHECK_PORT")

	cfg.TranslationServer.URL = mustString("TRANSLATION_SERVER_URL")


	// If any errors were recorded, return them as a single error
	if len(errs) > 0 {
		return nil, fmt.Errorf("configuration errors:\n- %s", strings.Join(errs, "\n- "))
	}

	return cfg, nil
}
