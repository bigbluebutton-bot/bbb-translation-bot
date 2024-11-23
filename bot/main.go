package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	api "github.com/bigbluebutton-bot/bigbluebutton-bot/api"

	"github.com/joho/godotenv"

	bot "github.com/bigbluebutton-bot/bigbluebutton-bot"

	bbb "github.com/bigbluebutton-bot/bigbluebutton-bot/bbb"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v4/pkg/media/oggwriter"
)

func main() {

	conf := loadConfig()

	// Wait for the transcription server to start by making a http request to http://{conf.TranscriptionServer.Host}:8001/health
	// Retry 10 times with 10 second delay
	waitForServer(conf)

	bbbapi, err := api.NewRequest(conf.BBB.API.URL, conf.BBB.API.Secret, conf.BBB.API.SHA)
	if err != nil {
		panic(err)
	}

	//API-Requests
	newmeeting, err := bbbapi.CreateMeeting("name", "meetingID", "attendeePW", "moderatorPW", "welcome text", false, false, false, 12345)
	if err != nil {
		panic(err)
	}
	fmt.Printf("New meeting \"%s\" was created.\n", newmeeting.MeetingName)

	url, err := bbbapi.JoinGetURL(newmeeting.MeetingID, "TestUser", true)
	if err != nil {
		panic(err)
	}
	fmt.Println("Moderator join url: " + url)

	time.Sleep(1 * time.Second)

	fmt.Println("-----------------------------------------------")

	clients := make([]*bot.Client, 0)
	clientsMutex := &sync.Mutex{}

	client, err := bot.NewClient(conf.BBB.Client.URL, conf.BBB.Client.WS, conf.BBB.Pad.URL, conf.BBB.Pad.WS, conf.BBB.API.URL, conf.BBB.API.Secret, conf.BBB.WebRTC.WS)
	if err != nil {
		panic(err)
	}

	client.OnStatus(func(status bot.StatusType) {
		fmt.Printf("Bot status: %s\n", status)
	})

	fmt.Println("Bot joins " + newmeeting.MeetingName + " as moderator:")
	err = client.Join(newmeeting.MeetingID, "Bot", true)
	if err != nil {
		panic(err)
	}

	chsetExternal := conf.ChangeSet.External
	chsetHost := conf.ChangeSet.Host
	chsetPort := conf.ChangeSet.Port

	err = client.OnGroupChatMsg(func(msg bbb.Message) {
		fmt.Println("[" + msg.SenderName + "]: " + msg.Message)

		if msg.Sender != client.InternalUserID {
			message := msg.Message

			if message == "ping" {
				client.SendChatMsg("pong", msg.ChatId)
			}

			if strings.HasPrefix(message, "!help") {
				helpText := "Available commands:\n" +
					"!help - List of all commands\n" +
					"!translate - Displays help for !translate and a list of all possible languages\n" +
					"!translate [language] - Starts translating into the specified language\n" +
					"!translate [language] stop - Stops translating into the specified language and the bot leaves the meeting"
				client.SendChatMsg(helpText, msg.ChatId)
			} else if strings.HasPrefix(message, "!translate") {
				args := strings.Split(message, " ")
				if len(args) < 2 {
					translateHelp := "Usage: !translate [language]\nLanguages: " + strings.Join(getSupportedLanguages(), ", ")
					client.SendChatMsg(translateHelp, msg.ChatId)
				} else {
					language := args[1]
					if args[1] == "stop" {
						language = args[0]
						// Stop translation and remove bot from meeting
						removeBot(client, newmeeting.MeetingID, "Bot-"+language)
					} else {
						name := client.LanguageShortToName(bot.Language(language))
						if name != "" {
							// Create new client
							newClient, err := bot.NewClient(conf.BBB.Client.URL, conf.BBB.Client.WS, conf.BBB.Pad.URL, conf.BBB.Pad.WS, conf.BBB.API.URL, conf.BBB.API.Secret, conf.BBB.WebRTC.WS)
							if err := newClient.Join(newmeeting.MeetingID, "Bot-"+language, true); err != nil {
								return
							}

							// Create new capture
							_, err = newClient.CreateCapture(bot.Language(language), chsetExternal, chsetHost, chsetPort)
							if err != nil {
								panic(err)
							}

							// Add to clientsPad
							clientsMutex.Lock()
							clients = append(clients, newClient)
							clientsMutex.Unlock()
						}
					}
				}
			}
		}
	})
	if err != nil {
		panic(err)
	}

	transcriptionHost := conf.TranscriptionServer.ExternalHost
	transcriptionPort := conf.TranscriptionServer.PortTCP
	transcriptionSecret := conf.TranscriptionServer.Secret

	sc := NewStreamClient(transcriptionHost, transcriptionPort, true, transcriptionSecret)

	sc.OnConnected(func(message string) {
		fmt.Println("Connected to server.")
	})

	sc.OnDisconnected(func(message string) {
		fmt.Println("Disconnected from server.")
		os.Exit(1)
	})

	sc.OnTimeout(func(message string) {
		fmt.Println("Connection to server timed out.")
		os.Exit(1)
	})

	sc.OnTCPMessage(func(text string) {
		fmt.Println("TCP message event:", text)
		validtext := strings.ToValidUTF8(text, "")

		clientsMutex.Lock()
		clientsTemp := clients
		clientsMutex.Unlock()
		if clientsTemp != nil {
			for _, cl := range clientsTemp {
				pads := cl.GetCaptures()
				for _, pad := range pads {
					translatedText := validtext
					if  pad.ShortLanguageName != "en" {
						translatedText, err = translate(conf.TranslationServer.URL, validtext, "en", pad.ShortLanguageName)
						if err != nil {
							fmt.Println("Error in translation:", err)
						}
					}
					err = pad.SetText(translatedText)
					if err != nil {
						fmt.Println("Error in pad write:", err)
					}
				}
			}
		}
	})

	err = sc.Connect()
	if err != nil {
		fmt.Println("Failed to connect to the server:", err)
		os.Exit(1)
	}
	defer sc.Close()

	audio := client.CreateAudioChannel()

	err = audio.ListenToAudio()
	if err != nil {
		panic(err)
	}

	oggFile, err := oggwriter.NewWith(sc, 48000, 2)
	if err != nil {
		panic(err)
	}
	defer oggFile.Close()

	audio.OnTrack(func(status *bot.StatusType, track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		// Only handle audio tracks
		if track.Kind() != webrtc.RTPCodecTypeAudio {
			return
		}

		fmt.Println("ID: " + track.ID())
		fmt.Println("Kind: " + track.Kind().String())
		fmt.Println("StreamID: " + track.StreamID())
		fmt.Println("SSRC: " + fmt.Sprint(track.SSRC()))
		fmt.Println("Codec: " + track.Codec().MimeType)
		fmt.Println("Codec PayloadType: " + fmt.Sprint(track.Codec().PayloadType))
		fmt.Println("Codec ClockRate: " + fmt.Sprint(track.Codec().ClockRate))
		fmt.Println("Codec Channels: " + fmt.Sprint(track.Codec().Channels))
		fmt.Println("Codec SDPFmtpLine: " + track.Codec().SDPFmtpLine)

		go func() {
			buffer := make([]byte, 1024)
			for {
				n, _, readErr := track.Read(buffer)

				if *status == bot.DISCONNECTED {
					return
				}

				if readErr != nil {
					fmt.Println("Error during audio track read:", readErr)
					return
				}

				rtpPacket := &rtp.Packet{}
				if err := rtpPacket.Unmarshal(buffer[:n]); err != nil {
					fmt.Println("Error during RTP packet unmarshal:", err)
					return
				}

				if err := oggFile.WriteRTP(rtpPacket); err != nil {
					fmt.Println("Error during OGG file write:", err)
					return
				}
			}
		}()
	})

	for {
		time.Sleep(1 * time.Second)
	}
}

// Map from BBB to Libretranslate language codes
var bbbToLibretranslate = map[string]string{
	"af":     "af",
	"ar":     "ar",
	"az":     "az",
	"bg-BG":  "bg",
	"bn":     "bn",
	"ca":     "ca",
	"cs-CZ":  "cs",
	"da":     "da",
	"de":     "de",
	"dv":     "", // No equivalent in Libretranslate
	"el-GR":  "el",
	"en":     "en",
	"eo":     "eo",
	"es":     "es",
	"es-419": "es", // General Spanish
	"es-ES":  "es",
	"es-MX":  "es",
	"et":     "et",
	"eu":     "", // No equivalent in Libretranslate
	"fa-IR":  "fa",
	"fi":     "fi",
	"fr":     "fr",
	"gl":     "", // No equivalent in Libretranslate
	"he":     "he",
	"hi-IN":  "hi",
	"hr":     "", // No equivalent in Libretranslate
	"hu-HU":  "hu",
	"hy":     "", // No equivalent in Libretranslate
	"id":     "id",
	"it-IT":  "it",
	"ja":     "ja",
	"ka":     "", // No equivalent in Libretranslate
	"km":     "", // No equivalent in Libretranslate
	"kn":     "", // No equivalent in Libretranslate
	"ko-KR":  "ko",
	"lo-LA":  "", // No equivalent in Libretranslate
	"lt-LT":  "lt",
	"lv":     "lv",
	"ml":     "", // No equivalent in Libretranslate
	"mn-MN":  "", // No equivalent in Libretranslate
	"nb-NO":  "nb",
	"nl":     "nl",
	"oc":     "", // No equivalent in Libretranslate
	"pl-PL":  "pl",
	"pt":     "pt",
	"pt-BR":  "pt",
	"ro-RO":  "ro",
	"ru":     "ru",
	"sk-SK":  "sk",
	"sl":     "sl",
	"sr":     "", // No equivalent in Libretranslate
	"sv-SE":  "sv",
	"ta":     "", // No equivalent in Libretranslate
	"te":     "", // No equivalent in Libretranslate
	"th":     "th",
	"tr-TR":  "tr",
	"uk-UA":  "uk",
	"vi-VN":  "", // No equivalent in Libretranslate
	"zh-CN":  "zh",
	"zh-TW":  "zt",
}

// Function to convert BBB language code to Libretranslate language code
func ConvertBBBToLibretranslate(bbbCode string) string {
	if code, exists := bbbToLibretranslate[bbbCode]; exists {
		return code
	}
	return ""
}

// TranslationRequest struct to hold the request payload
type TranslationRequest struct {
	Q      string `json:"q"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// TranslationResponse struct to parse the response
type TranslationResponse struct {
	TranslatedText string `json:"translatedText"`
}

// translate function sends a request to the LibreTranslate API and returns the translated text
func translate(apiURL, text, sourceLang, targetLang string) (string, error) {

	linreTargetLang := ConvertBBBToLibretranslate(targetLang)
	if linreTargetLang == "" {
		return "", fmt.Errorf("unsupported language: %s", targetLang)
	}

	// Create the request payload
	requestPayload := TranslationRequest{
		Q:      text,
		Source: sourceLang,
		Target: linreTargetLang,
	}

	// Convert the payload to JSON
	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("error marshalling request payload: %w", err)
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("error creating HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	// Check if the translation was successful
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get a valid response. Status code: %d, Response: %s", resp.StatusCode, body)
	}

	// Parse the response
	var translationResponse TranslationResponse
	err = json.Unmarshal(body, &translationResponse)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}

	// Return the translated text
	return translationResponse.TranslatedText, nil
}

// Function to get supported languages
func getSupportedLanguages() []string {
	languages := []string{}
	for key := range bbbToLibretranslate {
		if bbbToLibretranslate[key] != "" {
			languages = append(languages, key)
		}
	}
	return languages
}

// Function to remove bot from meeting
func removeBot(client *bot.Client, meetingID, botName string) {
	fmt.Printf("Bot %s leaves %s\n", botName, meetingID)
	if err := client.Leave(); err != nil {
		fmt.Printf("Error leaving meeting: %v\n", err)
	}
}

// Wait for the transcription server to start by making a http request to http://{conf.TranscriptionServer.Host}:{conf.TranscriptionServer.Port}/health
// Retry 10 times with 10 second delay
func waitForServer(conf *config) {
	// Define the URL using the configuration values
	url := fmt.Sprintf("http://%s:%d/health", conf.TranscriptionServer.ExternalHost, conf.TranscriptionServer.HealthCheckPort)

	// Try to connect to the server with retries
	for {
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Waiting for transcription server (%s) to start...\n", url)
			time.Sleep(10 * time.Second)
		} else {
			// Don't forget to close the response body when you're done with it
			resp.Body.Close()

			// If the status code is 200, the server is up
			if resp.StatusCode == http.StatusOK {
				fmt.Println("Transcription-server is up and running.")
				break
			}

			fmt.Println("Transcription-server is not ready yet...")
			time.Sleep(10 * time.Second)
		}
	}

	// Wait for translation server to start
	for {
		resp, err := translate(conf.TranslationServer.URL, "test", "en", "de")
		if err != nil {
			fmt.Printf("Waiting for translation server (%s) to start...\n", conf.TranslationServer.URL)
			time.Sleep(10 * time.Second)
		} else {
			// If the status code is 200, the server is up
			if resp != "" {
				fmt.Println("Translation-server is up and running.")
				break
			}

			fmt.Println("Translation-server is not ready yet...")
			time.Sleep(10 * time.Second)
		}
	}
}

// This part is for loading the config from .env file or the environment vars

type config struct {
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

func validateURL(envVar string, value string) (string, error) {
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value, nil
	}
	return "", fmt.Errorf("%s must be a valid URL", envVar)
}

func validateWS(envVar string, value string) (string, error) {
	if strings.HasPrefix(value, "ws://") || strings.HasPrefix(value, "wss://") {
		return value, nil
	}
	return "", fmt.Errorf("%s must be a valid WebSocket URL", envVar)
}

func validateBoolean(envVar string, value string) (string, error) {
	if value == "true" || value == "false" {
		return value, nil
	}
	return "", fmt.Errorf("%s must be 'true' or 'false'", envVar)
}

func validateInteger(envVar string, value string) (string, error) {
	_, err := strconv.Atoi(value)
	if err != nil {
		return "", fmt.Errorf("%s must be an integer", envVar)
	}
	return value, nil
}

func validateFloat(envVar string, value string) (string, error) {
	_, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return "", fmt.Errorf("%s must be a float", envVar)
	}
	return value, nil
}

func validateString(envVar string, value string) (string, error) {
	if value != "" {
		return value, nil
	}
	return "", fmt.Errorf("%s must be a non-empty string", envVar)
}

func loadConfig() *config {
	godotenv.Load() // Load the .env file

	conf := &config{}
	var hasErrors bool

	get_variable := func(env_var string, default_var string, validate_func func(envVar string, value string) (string, error)) string {
		value, exists := os.LookupEnv(env_var)
		if !exists {
			return default_var
		}
		if validate_func != nil {
			validatedValue, err := validate_func(env_var, value)
			if err != nil {
				hasErrors = true
				fmt.Printf("Error in %s: %s\n", env_var, err)
				return default_var
			}
			return validatedValue
		}
		return value
	}

	var err error

	// Populate config struct using get_variable
	conf.BBB.API.URL = get_variable("BBB_API_URL", "https://example.com/bigbluebutton/api/", validateURL)
	conf.BBB.API.Secret = get_variable("BBB_API_SECRET", "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX", validateString)
	conf.BBB.API.SHA = api.SHA(get_variable("BBB_API_SHA", "SHA256", validateString))

	conf.BBB.Client.URL = get_variable("BBB_CLIENT_URL", "https://example.com/html5client/", validateURL)
	conf.BBB.Client.WS = get_variable("BBB_CLIENT_WS", "wss://example.com/html5client/websocket", validateWS)

	conf.BBB.Pad.URL = get_variable("BBB_PAD_URL", "https://example.com/pad/", validateURL)
	conf.BBB.Pad.WS = get_variable("BBB_PAD_WS", "wss://example.com/pad/", validateWS)

	conf.BBB.WebRTC.WS = get_variable("BBB_WEBRTC_WS", "wss://example.com/bbb-webrtc-sfu", validateWS)

	conf.ChangeSet.External = get_variable("CHANGESET_EXTERNAL", "true", validateBoolean) == "true"
	conf.ChangeSet.Host = get_variable("CHANGESET_HOST", "localhost", validateString)
	conf.ChangeSet.Port, err = strconv.Atoi(get_variable("CHANGESET_PORT", "5051", validateInteger))
	if err != nil {
		fmt.Println("Error in CHANGESET_PORT:", err)
		hasErrors = true
	}

	conf.TranscriptionServer.ExternalHost = get_variable("TRANSCRIPTION_SERVER_EXTERNAL_HOST", "localhost", validateString)
	conf.TranscriptionServer.PortTCP, err = strconv.Atoi(get_variable("TRANSCRIPTION_SERVER_PORT_TCP", "5000", validateInteger))
	if err != nil {
		fmt.Println("Error in TRANSCRIPTION_SERVER_PORT_TCP:", err)
		hasErrors = true
	}
	conf.TranscriptionServer.Secret = get_variable("TRANSCRIPTION_SERVER_SECRET", "your_secret_token", validateString)
	conf.TranscriptionServer.HealthCheckPort, err = strconv.Atoi(get_variable("TRANSCRIPTION_SERVER_HEALTHCHECK_PORT", "8001", validateInteger))
	if err != nil {
		fmt.Println("Error in TRANSCRIPTION_SERVER_HEALTHCHECK_PORT:", err)
		hasErrors = true
	}

	conf.TranslationServer.URL = get_variable("TRANSLATION_SERVER_URL", "localhost", validateString)
	conf.TranslationServer.Secret = get_variable("TRANSLATION_SERVER_SECRET", "your_secret_token", validateString)

	if hasErrors {
		fmt.Println("Configuration errors found. Exiting program.")
		os.Exit(1)
	}

	return conf
}
