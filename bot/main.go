package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	api "github.com/bigbluebutton-bot/bigbluebutton-bot/api"
)

var (
	conf *Settings
	BM   *BotManager
)

func main() {
	var err error
	conf, err = LoadSettings() // Replace readConfig with LoadSettings
	if err != nil {
		panic(err)
	}

	// Check if all external services are up and running
	healthCheck(conf)

	BM = NewBotManager(
		1,
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

	bbbapi, err := api.NewRequest(conf.BBB.API.URL, conf.BBB.API.Secret, conf.BBB.API.SHA)
	if err != nil {
		panic(err)
	}

	// Get a list of all meetings
	meetings, err := bbbapi.GetMeetings()
	if err != nil {
		panic(err)
	}

	// Join all meetings
	for _, meeting := range meetings {
		// Create a bot
		bot, err := BM.AddBot()
		if err != nil {
			panic(err)
		}
		bot.Join(
			meeting.MeetingID,
			"Bot",
		)
		bot.Translate("de")
	}

	time.Sleep(50 * time.Second)

	fmt.Println("THE END")
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
