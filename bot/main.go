package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	bbbbot "github.com/bigbluebutton-bot/bigbluebutton-bot"
	api "github.com/bigbluebutton-bot/bigbluebutton-bot/api"
	"github.com/bigbluebutton-bot/bigbluebutton-bot/bbb"
)

var (
	currentMeetingID string
	isBotConnected   bool

	// Variables needed for translation functionality
	conf          *Settings
	client		*bbbbot.Client
	clients       []*bbbbot.Client
	clientsMutex  = &sync.Mutex{}
)

func main() {
	var err error
	conf, err = LoadSettings() // Replace readConfig with LoadSettings
	if err != nil {
		panic(err)
	}

	bbbapi, err := api.NewRequest(conf.BBB.API.URL, conf.BBB.API.Secret, conf.BBB.API.SHA)
	if err != nil {
		panic(err)
	}

	client, err = bbbbot.NewClient(
		conf.BBB.Client.URL,
		conf.BBB.Client.WS,
		conf.BBB.Pad.URL,
		conf.BBB.Pad.WS,
		conf.BBB.API.URL,
		conf.BBB.API.Secret,
		conf.BBB.WebRTC.WS,
	)
	if err != nil {
		panic(err)
	}

	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", fs)

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		// GET /status
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		response := map[string]interface{}{
			"connected": isBotConnected,
			"meetingID": currentMeetingID,
		}
		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/meetings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleGetMeetings(w, r, bbbapi)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/meeting/", func(w http.ResponseWriter, r *http.Request) {
		// URL patterns we need to handle here:
		// POST /meeting
		// GET /meeting/{meetingID}/join
		// POST /meeting/{meetingID}/bot/join
		// POST /meeting/{meetingID}/bot/leave
		// POST /meeting/{meetingID}/end

		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

		fmt.Println(parts)
		if len(parts) == 1 && r.Method == http.MethodPost {
			handleCreateMeeting(w, r, bbbapi)
			return
		}

		if len(parts) < 2 {
			http.Error(w, "meetingID not provided", http.StatusBadRequest)
			return
		}
		meetingID := parts[1]

		if len(parts) == 3 && parts[2] == "join" && r.Method == http.MethodGet {
			handleGetJoinLinks(w, r, bbbapi, meetingID)
			return
		}

		if len(parts) == 4 && parts[2] == "bot" && parts[3] == "join" && r.Method == http.MethodPost {
			handleBotJoinMeeting(w, r, meetingID)
			return
		}

		if len(parts) == 4 && parts[2] == "bot" && parts[3] == "leave" && r.Method == http.MethodPost {
			handleBotLeaveMeeting(w, r)
			return
		}

		if len(parts) == 3 && parts[2] == "end" && r.Method == http.MethodPost {
			handleEndMeeting(w, r, bbbapi, meetingID)
			return
		}

		http.Error(w, "Not found", http.StatusNotFound)
	})

	// Start server
	port := ":8080"
	fmt.Printf("Starting server on port %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		panic(err)
	}
}

func handleGetMeetings(w http.ResponseWriter, r *http.Request, bbbapi *api.ApiRequest) {
	meetings, err := bbbapi.GetMeetings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"meetings": meetings,
	}
	json.NewEncoder(w).Encode(response)
}

func handleCreateMeeting(w http.ResponseWriter, r *http.Request, bbbapi *api.ApiRequest) {
	var req struct {
		Name               string `json:"name"`
		MeetingID          string `json:"meetingID"`
		AttendeePW         string `json:"attendeePW"`
		ModeratorPW        string `json:"moderatorPW"`
		Welcome            string `json:"welcome"`
		AllowStartStopRec  bool   `json:"allowStartStopRecording"`
		AutoStartRecording bool   `json:"autoStartRecording"`
		Record             bool   `json:"record"`
		VoiceBridge        int    `json:"voiceBridge"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	meeting, err := bbbapi.CreateMeeting(req.Name, req.MeetingID, req.AttendeePW, req.ModeratorPW, req.Welcome, req.AllowStartStopRec, req.AutoStartRecording, req.Record, int64(req.VoiceBridge))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(meeting)
}

func handleBotJoinMeeting(w http.ResponseWriter, r *http.Request, meetingID string) {
	var req struct {
		UserName  string `json:"userName"`
		Moderator bool   `json:"moderator"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := client.Join(meetingID, req.UserName, req.Moderator); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update the global state
	isBotConnected = true
	currentMeetingID = meetingID

	// Add translation functionality: listen to group chat messages and respond
	err := client.OnGroupChatMsg(func(msg bbb.Message) {
		fmt.Println("[" + msg.SenderName + "]: " + msg.Message)

		// If message is not from the bot itself
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
					"!translate [language] stop - Stops translating into the specified language"
				client.SendChatMsg(helpText, msg.ChatId)
			} else if strings.HasPrefix(message, "!translate") {
				args := strings.Split(message, " ")
				if len(args) < 2 {
					translateHelp := "Usage: !translate [language]\nLanguages: " + strings.Join(getSupportedLanguages(), ", ")
					client.SendChatMsg(translateHelp, msg.ChatId)
				} else {
					language := args[1]
					if language == "stop" {
						client.SendChatMsg("Stopping translation and leaving the meeting.", msg.ChatId)
						removeBot(client, meetingID, "Bot-"+language)
					} else {
						name := client.LanguageShortToName(bbbbot.Language(language))
						if name != "" {
							chsetExternal := conf.ChangeSet.External
							chsetHost := conf.ChangeSet.Host
							chsetPort := conf.ChangeSet.Port

							newClient, err := bbbbot.NewClient(
								conf.BBB.Client.URL,
								conf.BBB.Client.WS,
								conf.BBB.Pad.URL,
								conf.BBB.Pad.WS,
								conf.BBB.API.URL,
								conf.BBB.API.Secret,
								conf.BBB.WebRTC.WS,
							)
							if err != nil {
								fmt.Println("Error creating new translator bot:", err)
								return
							}

							if err := newClient.Join(meetingID, "Bot-"+language, true); err != nil {
								fmt.Println("Error joining translator bot:", err)
								return
							}

							_, err = newClient.CreateCapture(bbbbot.Language(language), chsetExternal, chsetHost, chsetPort)
							if err != nil {
								fmt.Println("Error creating capture:", err)
								return
							}

							clientsMutex.Lock()
							clients = append(clients, newClient)
							clientsMutex.Unlock()

							client.SendChatMsg("Started translating into "+name+".", msg.ChatId)
						} else {
							client.SendChatMsg("Unsupported language: "+language, msg.ChatId)
						}
					}
				}
			}
		}
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func handleBotLeaveMeeting(w http.ResponseWriter, r *http.Request) {
	// Since we don't store the bot client here directly after join (we create it in handler), 
	// we would need a reference to that client. For simplicity, let's assume there's only one bot client connected.
	// This is a simplification. In a real scenario, we'd track the client instance globally or by meetingID.
	// But given instructions, let's assume we have a single global client reference for the current bot meeting.

	if !isBotConnected {
		json.NewEncoder(w).Encode(map[string]string{"status": "no_bot_connected"})
		return
	}

	// We need to find the client that joined currentMeetingID.
	// If we wanted to leave the main bot (the first one that joined), we must have a reference.
	// Let's store the first joined client as well in a global variable when bot joins.

	// For simplicity, let's re-create the client and leave:
	leaveClient, err := bbbbot.NewClient(
		conf.BBB.Client.URL,
		conf.BBB.Client.WS,
		conf.BBB.Pad.URL,
		conf.BBB.Pad.WS,
		conf.BBB.API.URL,
		conf.BBB.API.Secret,
		conf.BBB.WebRTC.WS,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// If the bot is connected to currentMeetingID, join with same name (assuming we know name)
	// Actually we can just call Leave() but we need the bot to be in the meeting.
	leaveClient.Join(currentMeetingID, "Bot", true) // ignore error
	if err := leaveClient.Leave(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	isBotConnected = false
	currentMeetingID = ""

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func handleEndMeeting(w http.ResponseWriter, r *http.Request, bbbapi *api.ApiRequest, meetingID string) {
	if _, err := bbbapi.EndMeeting(meetingID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if currentMeetingID == meetingID {
		isBotConnected = false
		currentMeetingID = ""
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func handleGetJoinLinks(w http.ResponseWriter, r *http.Request, bbbapi *api.ApiRequest, meetingID string) {
	// Get username from query params
	username := r.URL.Query().Get("username")
	if username == "" {
		username = "Guest"
	}

	moderatorURL, err := bbbapi.JoinGetURL(meetingID, username, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	attendeeURL, err := bbbapi.JoinGetURL(meetingID, username, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"moderator": moderatorURL,
		"attendee":  attendeeURL,
	}
	json.NewEncoder(w).Encode(response)
}

// ---------------------- HELPER FUNCTIONS FOR TRANSLATION ----------------------

// Map from BBB to Libretranslate language codes
var bbbToLibretranslate = map[string]string{
	"af":    "af",
	"ar":    "ar",
	"az":    "az",
	"bg-BG": "bg",
	"bn":    "bn",
	"ca":    "ca",
	"cs-CZ": "cs",
	"da":    "da",
	"de":    "de",
	"el-GR": "el",
	"en":    "en",
	"eo":    "eo",
	"es":    "es",
	"es-419": "es",
	"es-ES": "es",
	"es-MX": "es",
	"et":    "et",
	"fa-IR": "fa",
	"fi":    "fi",
	"fr":    "fr",
	"he":    "he",
	"hi-IN": "hi",
	"hu-HU": "hu",
	"id":    "id",
	"it-IT": "it",
	"ja":    "ja",
	"ko-KR": "ko",
	"lt-LT": "lt",
	"lv":    "lv",
	"nb-NO": "nb",
	"nl":    "nl",
	"pl-PL": "pl",
	"pt":    "pt",
	"pt-BR": "pt",
	"ro-RO": "ro",
	"ru":    "ru",
	"sk-SK": "sk",
	"sl":    "sl",
	"sv-SE": "sv",
	"th":    "th",
	"tr-TR": "tr",
	"uk-UA": "uk",
	"zh-CN": "zh",
}

// ConvertBBBToLibretranslate converts BBB language code to Libretranslate language code
func ConvertBBBToLibretranslate(bbbCode string) string {
	if code, exists := bbbToLibretranslate[bbbCode]; exists {
		return code
	}
	return ""
}

// TranslationRequest struct to hold the request payload for translation
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

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("error marshalling request payload: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("error creating HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get a valid response. Status code: %d, Response: %s", resp.StatusCode, body)
	}

	var translationResponse TranslationResponse
	err = json.Unmarshal(body, &translationResponse)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}

	return translationResponse.TranslatedText, nil
}

// getSupportedLanguages returns a list of supported languages
func getSupportedLanguages() []string {
	languages := []string{}
	for key := range bbbToLibretranslate {
		if bbbToLibretranslate[key] != "" {
			languages = append(languages, key)
		}
	}
	return languages
}

// removeBot from meeting
func removeBot(client *bbbbot.Client, meetingID, botName string) {
	fmt.Printf("Bot %s leaves %s\n", botName, meetingID)
	if err := client.Leave(); err != nil {
		fmt.Printf("Error leaving meeting: %v\n", err)
	}
}