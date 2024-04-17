package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
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

	fmt.Println("-----------------------------------------------")

	fmt.Println("All meetings:")
	meetings, err := bbbapi.GetMeetings()
	if err != nil {
		panic(err)
	}
	for _, meeting := range meetings {
		fmt.Print(meeting.MeetingName + ": ")
		fmt.Println(bbbapi.IsMeetingRunning(meeting.MeetingID))
	}

	fmt.Println("-----------------------------------------------")

	url, err := bbbapi.JoinGetURL(newmeeting.MeetingID, "TestUser", true)
	if err != nil {
		panic(err)
	}
	fmt.Println("Moderator join url: " + url)

	time.Sleep(1 * time.Second)

	fmt.Println("-----------------------------------------------")

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

	err = client.OnGroupChatMsg(func(msg bbb.Message) {

		fmt.Println("[" + msg.SenderName + "]: " + msg.Message)

		if msg.Sender != client.InternalUserID {
			if msg.Message == "ping" {
				fmt.Println("Sending pong")
				client.SendChatMsg("pong", msg.ChatId)
			}
		}
	})
	if err != nil {
		panic(err)
	}

	chsetExternal := conf.ChangeSet.External
	chsetHost := conf.ChangeSet.Host
	chsetPort := conf.ChangeSet.Port

	enCapture, err := client.CreateCapture("en", chsetExternal, chsetHost, chsetPort)
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

		err = enCapture.SetText(validtext)
		if err != nil {
			panic(err)
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

	// if err := audio.Close(); err != nil {
	// 	panic(err)
	// }

	// fmt.Println("Bot leaves " + newmeeting.MeetingName)
	// err = client.Leave()
	// if err != nil {
	// 	panic(err)
	// }
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
			fmt.Println("Waiting for transcription server to start...")
			time.Sleep(10 * time.Second)
		} else {
			// Don't forget to close the response body when you're done with it
			resp.Body.Close()

			// If the status code is 200, the server is up
			if resp.StatusCode == http.StatusOK {
				fmt.Println("Transcription server is up and running.")
				return
			}

			fmt.Println("Server is not ready yet...")
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