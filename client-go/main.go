package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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

		if(msg.Sender != client.InternalUserID) {
			if(msg.Message == "ping") {
				fmt.Println("Sending pong")
				client.SendChatMsg("pong", msg.ChatId)
			}
		}
	})
	if err != nil {
		panic(err)
	}

	enCapture, err := client.CreateCapture("en")
	if err != nil {
		panic(err)
	}




	

	sc := NewStreamClient("172.30.62.194", 5000, true, "your_secret_token")

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


	time.Sleep(200 * time.Second)

	if err := audio.Close(); err != nil {
		panic(err)
	}

	

	fmt.Println("Bot leaves " + newmeeting.MeetingName)
	err = client.Leave()
	if err != nil {
		panic(err)
	}
}










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
	API    configAPI	`json:"api"`
	Client configClient	`json:"client"`
	Pad configPad		`json:"pad"`
	WebRTC configWebRTC `json:"webrtc"`
}

type config struct {
	BBB configBBB `json:"bbb"`
}

func readConfig(file string) config {
	// Try to read from env
	conf := config {
		BBB: configBBB{
			API: configAPI{
				URL: os.Getenv("BBB_API_URL"),
				Secret: os.Getenv("BBB_API_SECRET"),
				SHA: api.SHA(os.Getenv("BBB_API_SHA")),
			},
			Client: configClient{
				URL: os.Getenv("BBB_CLIENT_URL"),
				WS: os.Getenv("BBB_CLIENT_WS"),
			},
			Pad: configPad{
				URL: os.Getenv("BBB_PAD_URL"),
				WS: os.Getenv("BBB_PAD_WS"),
			},
			WebRTC: configWebRTC{
				WS: os.Getenv("BBB_WEBRTC_WS"),
			},
		},
	}

	if (conf.BBB.API.URL != "" && conf.BBB.API.Secret != "" && conf.BBB.API.SHA != "" &&
		conf.BBB.Client.URL != "" && conf.BBB.Client.WS != "" &&
		conf.BBB.Pad.URL != "" && conf.BBB.Pad.WS != ""){
		fmt.Println("Using env variables for config")
		return conf
	}

	// Open our jsonFile
	jsonFile, err := os.Open(file)
	// if we os.Open returns an error then handle it
	if (err != nil) {
		fmt.Println(err)
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()
	// read our opened jsonFile as a byte array.
	byteValue, err := io.ReadAll(jsonFile)
	if(err != nil) {
		panic(err)
	}
	// we unmarshal our byteArray which contains our jsonFile's content into conf
	json.Unmarshal([]byte(byteValue), &conf) 

	return conf
}