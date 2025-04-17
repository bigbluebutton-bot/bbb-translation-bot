package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	bbbbot "github.com/bigbluebutton-bot/bigbluebutton-bot"
	bbbapi "github.com/bigbluebutton-bot/bigbluebutton-bot/api"
	"github.com/gorilla/mux"
)

var (
	conf *Settings
	BM   *BotManager
	bbb_api *bbbapi.ApiRequest
)

// Define the API handlers
type statusResponse struct {
	Bots_count int `json:"bots_count"`
	Max_bots int `json:"max_bots"`
}
func statusHandler(w http.ResponseWriter, r *http.Request) {
	// Check the status of the bot manager
	status := statusResponse{
		Bots_count: len(BM.GetBots()),
		Max_bots: BM.Max_bots,
	}

	// Respond with the status
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// marshall status to JSON
	responseData, err := json.Marshal(status)
	if err != nil {
		http.Error(w, "Failed to marshal status data", http.StatusInternalServerError)
		return
	}
	w.Write(responseData)
}

func getAllMeetingsHandler(w http.ResponseWriter, r *http.Request) {
	meetings, err := bbb_api.GetMeetings()
	if err != nil {
		http.Error(w, "Failed to fetch meetings", http.StatusInternalServerError)
		return
	}

	// Respond with the list of meetings
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// retrun meetings as JSON
	meetings_list := make([]bbbapi.Meeting, len(meetings))
	count := 0
	for _, meeting := range meetings {
		meetings_list[count] = meeting
		count++
	}
	
	// marshall meetings_list to JSON
	responseData, err := json.Marshal(meetings_list)
	if err != nil {
		http.Error(w, "Failed to marshal meetings data", http.StatusInternalServerError)
		return
	}
	w.Write(responseData)
}

func getMeetingHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	meetingID := vars["meeting_id"]

	meetings, err := bbb_api.GetMeetings()
	if err != nil {
		http.Error(w, "Failed to fetch meeting", http.StatusInternalServerError)
		return
	}
	meeting, ok := meetings[meetingID]
	if !ok {
		http.Error(w, "Meeting not found", http.StatusNotFound)
		return
	}

	// Respond with the meeting data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// marshall meeting to JSON
	responseData, err := json.Marshal(meeting)
	if err != nil {
		http.Error(w, "Failed to marshal meeting data", http.StatusInternalServerError)
		return
	}
	w.Write(responseData)
}

func deleteMeetingHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	meetingID := vars["meeting_id"]

	_, err := bbb_api.EndMeeting(meetingID)
	if err != nil {
		http.Error(w, "Failed to delete meeting", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getLanguagesHandler(w http.ResponseWriter, r *http.Request) {
	language_codes := bbbbot.AllLanguages()
	// Create a map with the language codes and its name
	language_map := make(map[string]string)
	for _, code := range language_codes {
		language_map[string(code)] = bbbbot.LanguageShortToName(code)
	}

	// Respond with the list of languages
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// marshall language_map to JSON
	responseData, err := json.Marshal(language_map)
	if err != nil {
		http.Error(w, "Failed to marshal languages data", http.StatusInternalServerError)
		return
	}
	w.Write(responseData)
}

// Define bot-related handlers
func getAllBotsHandler(w http.ResponseWriter, r *http.Request) {
	bots := BM.GetBots()

	// Respond with all bots information
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// marshall bots to JSON
	responseData, err := json.Marshal(bots)
	if err != nil {
		http.Error(w, "Failed to marshal bots data", http.StatusInternalServerError)
		return
	}
	w.Write(responseData)
}

func getBotHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	botID := vars["bot_id"]

	bot, found := BM.GetBot(botID)
	if !found {
		http.Error(w, "Bot not found", http.StatusNotFound)
		return
	}

	// Respond with bot details
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// marshall bot to JSON
	responseData, err := json.Marshal(bot)
	if err != nil {
		http.Error(w, "Failed to marshal bot data", http.StatusInternalServerError)
		return
	}
	w.Write(responseData)
}

func botJoinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	meetingID := vars["meeting_id"]

	// check if max bots limit is reached
	if len(BM.GetBots()) >= BM.Max_bots {
		http.Error(w, "Max bots limit reached", http.StatusTooManyRequests)
		return
	}

	// check if meeting exists
	meetings, err := bbb_api.GetMeetings()
	if err != nil {
		http.Error(w, "Failed to fetch meetings", http.StatusInternalServerError)
		return
	}

	_, ok := meetings[meetingID]
	if !ok {
		http.Error(w, "Meeting not found", http.StatusNotFound)
		return
	}

	// Create a new bot
	bot, err := BM.AddBot()
	if err != nil {
		http.Error(w, "Failed to create bot", http.StatusInternalServerError)
		return
	}

	// Decode request body to extract meeting ID
	err = bot.Join(meetingID, "Bot")
	if err != nil {
		http.Error(w, "Failed to join meeting", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func botLeaveHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	botID := vars["bot_id"]

	bot, found := BM.GetBot(botID)
	if !found {
		http.Error(w, "Bot not found", http.StatusNotFound)
		return
	}

	// Leave the meeting
	bot.Disconnect()

	w.WriteHeader(http.StatusOK)
}

func botSetTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	botID := vars["bot_id"]
	task_string := vars["task"]

	bot, found := BM.GetBot(botID)
	if !found {
		http.Error(w, "Bot not found", http.StatusNotFound)
		return
	}

	// task string to Task type
	var task Task
	switch task_string {
		case "transcribe":
			task = Transcribe
		case "translate":
			task = Translate
		default:
			http.Error(w, "Invalid task type", http.StatusBadRequest)
			return
	}

	bot.SetTask(task)

	w.WriteHeader(http.StatusOK)
}

func botTranslateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	botID := vars["bot_id"]
	lang := vars["lang"]

	bot, found := BM.GetBot(botID)
	if !found {
		http.Error(w, "Bot not found", http.StatusNotFound)
		return
	}

	// check if lang is valid
	language_codes := bbbbot.AllLanguages()
	found = false
	for _, code := range language_codes {
		if code == bbbbot.Language(lang) {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "Invalid language code", http.StatusBadRequest)
		return
	}

	// Start translation
	err := bot.Translate(lang)
	if err != nil {
		http.Error(w, "Failed to start translation", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func botStopTranslateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	botID := vars["bot_id"]
	lang := vars["lang"]

	bot, found := BM.GetBot(botID)
	if !found {
		http.Error(w, "Bot not found", http.StatusNotFound)
		return
	}

	// check if lang is valid
	language_codes := bbbbot.AllLanguages()
	found = false
	for _, code := range language_codes {
		if code == bbbbot.Language(lang) {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "Invalid language code", http.StatusBadRequest)
		return
	}

	// Stop translation
	err := bot.StopTranslate(lang)
	if err != nil {
		http.Error(w, "Failed to stop translation", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}


// Initialize the HTTP server and routes
func main() {
	var err error
	conf, err = LoadSettings() // Replace readConfig with LoadSettings
	if err != nil {
		panic(err)
	}

	// Check if all external services are up and running
	healthCheck(conf)

	bbb_api, err = bbbapi.NewRequest(conf.BBB.API.URL, conf.BBB.API.Secret, conf.BBB.API.SHA)
	if err != nil {
		panic(err)
	}

	// Create the BotManager
	BM = NewBotManager(
		conf.Bot.Limit,
		conf.BBB.Client.URL,
		conf.BBB.Client.WS,
		conf.BBB.Pad.URL,
		conf.BBB.Pad.WS,
		conf.BBB.API.URL,
		conf.BBB.API.Secret,
		conf.BBB.WebRTC.WS,
		conf.TranscriptionServer.ExternalHost,
		conf.TranscriptionServer.PortTCP,
		conf.TranscriptionServer.Secret,
		conf.TranslationServer.URL,
		conf.ChangeSet.External,
		conf.ChangeSet.Port,
		conf.ChangeSet.Host,
	)

	// Initialize the router
	r := mux.NewRouter()

	// API routes
	r.HandleFunc("/api/v1/status", statusHandler).Methods("GET")

	r.HandleFunc("/api/v1/bbb/meetings", getAllMeetingsHandler).Methods("GET")
	r.HandleFunc("/api/v1/bbb/meeting/{meeting_id}", getMeetingHandler).Methods("GET")
	r.HandleFunc("/api/v1/bbb/meeting/{meeting_id}", deleteMeetingHandler).Methods("DELETE")
	r.HandleFunc("/api/v1/bbb/languages", getLanguagesHandler).Methods("GET")

	r.HandleFunc("/api/v1/bots", getAllBotsHandler).Methods("GET")
	r.HandleFunc("/api/v1/bot/{bot_id}", getBotHandler).Methods("GET")
	r.HandleFunc("/api/v1/bot/join/{meeting_id}", botJoinHandler).Methods("POST")
	r.HandleFunc("/api/v1/bot/{bot_id}/leave", botLeaveHandler).Methods("POST")
	r.HandleFunc("/api/v1/bot/{bot_id}/task/{task}", botSetTaskHandler).Methods("PUT")
	r.HandleFunc("/api/v1/bot/{bot_id}/translate/{lang}", botTranslateHandler).Methods("PUT")
	r.HandleFunc("/api/v1/bot/{bot_id}/translate/{lang}", botStopTranslateHandler).Methods("DELETE")

	// Serve static files in public to /
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./public"))))

	// Start the server
	log.Println("Server starting on :8080...")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func healthCheck(conf *Settings) {
	// Checks if all external services are up and running

	// Transcription server
	// Translation server
	// ChangeSet server

	transcription_srv := conf.TranscriptionServer.ExternalHost
	transcription_health_port := conf.TranscriptionServer.HealthCheckPort

	// Check if the transcription server is reachable
	for {
		if _, err := http.Get("http://" + transcription_srv + ":" + strconv.Itoa(transcription_health_port) + "/health"); err != nil {
			fmt.Println("Transcription server is down (" + "http://" + transcription_srv + ":" + strconv.Itoa(transcription_health_port) + "/health" + "). Retrying in 5seconds...")
			time.Sleep(5 * time.Second)
		} else {
			fmt.Println("Transcription server is up")
			break
		}
		time.Sleep(5 * time.Second)
	}
}
