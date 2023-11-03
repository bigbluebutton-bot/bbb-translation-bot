package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	api "github.com/ITLab-CC/bigbluebutton-bot/api"

	bot "github.com/ITLab-CC/bigbluebutton-bot"

	bbb "github.com/ITLab-CC/bigbluebutton-bot/bbb"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v4/pkg/media/oggwriter"
)

func main() {

	conf := readConfig("config.json")

	// Wait for the transcription server to start by making a http request to http://{conf.TT.TranscriptionServer.Host}:8001/health
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

	chsetExternal, err := strconv.ParseBool(conf.ChangeSet.External)
	if err != nil {
		panic(err)
	}
	chsetHost := conf.ChangeSet.Host
	chsetPort, err := strconv.Atoi(conf.ChangeSet.Port)
	if err != nil {
		panic(err)
	}

	enCapture, err := client.CreateCapture("en", chsetExternal, chsetHost, chsetPort)
	if err != nil {
		panic(err)
	}

	transcriptionHost := conf.TT.TranscriptionServer.Host
	transcriptionPort, err := strconv.Atoi(conf.TT.TranscriptionServer.Port)
	if err != nil {
		panic(err)
	}
	transcriptionSecret := conf.TT.TranscriptionServer.Secret

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

	// Wait for the transcription server to start by making a http request to http://{conf.TT.TranscriptionServer.Host}:{conf.TT.TranscriptionServer.Port}/health
	// Retry 10 times with 10 second delay
func waitForServer(conf config) {
	// Define the URL using the configuration values
	url := fmt.Sprintf("http://%s:%s/health", conf.TT.TranscriptionServer.Host, "8001")

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

/*
{
    "bbb":{
       "api":{
          "url":"https://example.com/bigbluebutton/api/",
          "secret":"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
          "sha":"SHA256"
       },
       "client":{
          "url":"https://example.com/html5client/",
          "ws":"wss://example.com/html5client/websocket"
       },
       "pad":{
          "url":"https://example.com/pad/",
          "ws":"wss://example.com/pad/"
       },
       "webrtc":{
          "ws":"wss://example.com/bbb-webrtc-sfu"
       }
    },
    "changeset": {
        "external": "false",
        "host": "127.0.0.1",
        "port": "50051"
    },
    "tt":{
       "transcription-server":{
          "host":"172.30.62.203",
          "port":"5000",
          "secret":"your_secret_token"
       },
       "translation-server":{
          "host":"172.30.62.203",
          "secret":""
       }
    }
}
*/

type configAPI struct {
	URL    string  `json:"url"`
	Secret string  `json:"secret"`
	SHA    api.SHA `json:"sha"`
}

type configClient struct {
	URL string `json:"url"`
	WS  string `json:"ws"`
}

type configPad struct {
	URL string `json:"url"`
	WS  string `json:"ws"`
}

type configWebRTC struct {
	WS string `json:"ws"`
}

type configBBB struct {
	API    configAPI    `json:"api"`
	Client configClient `json:"client"`
	Pad    configPad    `json:"pad"`
	WebRTC configWebRTC `json:"webrtc"`
}

type config struct {
	BBB configBBB `json:"bbb"`
	ChangeSet configChangeSet `json:"changeset"`
	TT  configTT  `json:"tt"`
}

type configChangeSet struct {
	External string   `json:"external"`
	Host     string `json:"host"`
	Port     string    `json:"port"`
}

type configTT struct {
	TranscriptionServer configTranscriptionServer `json:"transcription-server"`
	TranslationServer   configTranslationServer   `json:"translation-server"`
}

type configTranscriptionServer struct {
	Host   string `json:"host"`
	Port   string    `json:"port"`
	Secret string `json:"secret"`
}

type configTranslationServer struct {
	Url   string `json:"url"`
	Secret string `json:"secret"`
}

func readConfig(file string) config {
	// Try to read from env
	conf := config{
		BBB: configBBB{
			API: configAPI{
				URL:    os.Getenv("BBB_API_URL"),
				Secret: os.Getenv("BBB_API_SECRET"),
				SHA:    api.SHA(os.Getenv("BBB_API_SHA")),
			},
			Client: configClient{
				URL: os.Getenv("BBB_CLIENT_URL"),
				WS:  os.Getenv("BBB_CLIENT_WS"),
			},
			Pad: configPad{
				URL: os.Getenv("BBB_PAD_URL"),
				WS:  os.Getenv("BBB_PAD_WS"),
			},
			WebRTC: configWebRTC{
				WS: os.Getenv("BBB_WEBRTC_WS"),
			},
		},
		ChangeSet: configChangeSet{
			External: os.Getenv("CHANGESET_EXTERNAL"),
			Host:     os.Getenv("CHANGESET_HOST"),
			Port:     os.Getenv("CHANGESET_PORT"),
		},
		TT: configTT{
			TranscriptionServer: configTranscriptionServer{
				Host:   os.Getenv("TRANSCRIPTION_SERVER_HOST"),
				Port:   os.Getenv("TRANSCRIPTION_SERVER_PORT"),
				Secret: os.Getenv("TRANSCRIPTION_SERVER_SECRET"),
			},
			TranslationServer: configTranslationServer{
				Url:   os.Getenv("TRANSLATION_SERVER_URL"),
				Secret: os.Getenv("TRANSLATION_SERVER_SECRET"),
			},
		},
	}

	if conf.BBB.API.URL != "" && conf.BBB.API.Secret != "" && conf.BBB.API.SHA != "" &&
		conf.BBB.Client.URL != "" && conf.BBB.Client.WS != "" &&
		conf.BBB.Pad.URL != "" && conf.BBB.Pad.WS != "" && 
		conf.BBB.WebRTC.WS != "" && 
		conf.ChangeSet.Host != "" && conf.ChangeSet.Port != "" &&
		conf.TT.TranscriptionServer.Host  != "" && conf.TT.TranscriptionServer.Port != "" && conf.TT.TranscriptionServer.Secret != "" && 
		conf.TT.TranslationServer.Url != "" && conf.TT.TranslationServer.Secret != "" {
		fmt.Println("Using env variables for config")
		return conf
	}

	// Open our jsonFile
	jsonFile, err := os.Open(file)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()
	// read our opened jsonFile as a byte array.
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		panic(err)
	}
	// we unmarshal our byteArray which contains our jsonFile's content into conf
	json.Unmarshal([]byte(byteValue), &conf)

	return conf
}
