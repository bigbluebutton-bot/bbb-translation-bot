package main

import (
	"fmt"
	"strings"
	"sync"

	bbbbot "github.com/bigbluebutton-bot/bigbluebutton-bot"
	"github.com/bigbluebutton-bot/bigbluebutton-bot/pad"
	"github.com/google/uuid"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v4/pkg/media/oggwriter"
)

type BotManager struct {
	Max_bots int

	lock    sync.Mutex
	bots map[string]*Bot

	bbb_client_url         string
	bbb_client_ws          string
	bbb_pad_url            string
	bbb_pad_ws             string
	bbb_api_url            string
	bbb_api_secret         string
	bbb_webrtc_ws          string
	transcription_host     string
	transcription_port     int
	transcription_secret   string
	translation_server_url string
	changeset_external     bool
	changeset_port         int
	changeset_host         string
}

func NewBotManager(
	max_bots int,
	bbb_client_url string,
	bbb_client_ws string,
	bbb_pad_url string,
	bbb_pad_ws string,
	bbb_api_url string,
	bbb_api_secret string,
	bbb_webrtc_ws string,
	transcription_host string,
	transcription_port int,
	transcription_secret string,
	translation_server_url string,
	changeset_external bool,
	changeset_port int,
	changeset_host string,
) *BotManager {
	return &BotManager{
		Max_bots:              max_bots,
		bots:                   make(map[string]*Bot),
		bbb_client_url:         bbb_client_url,
		bbb_client_ws:          bbb_client_ws,
		bbb_pad_url:            bbb_pad_url,
		bbb_pad_ws:             bbb_pad_ws,
		bbb_api_url:            bbb_api_url,
		bbb_api_secret:         bbb_api_secret,
		bbb_webrtc_ws:          bbb_webrtc_ws,
		transcription_host:     transcription_host,
		transcription_port:     transcription_port,
		transcription_secret:   transcription_secret,
		translation_server_url: translation_server_url,
		changeset_external:     changeset_external,
		changeset_port:         changeset_port,
		changeset_host:         changeset_host,
	}
}

func (bm *BotManager) AddBot() (*Bot, error) {
	if len(bm.bots) >= bm.Max_bots {
		return nil, fmt.Errorf("max bots reached")
	}

	// transcription_host string,
	// transcription_port int,
	// transcription_secret string,
	// translation_server_url string,
	// changeset_external bool,
	// changeset_port int,
	// changeset_host string,
	new_bot := NewBot(
		bm.bbb_client_url,
		bm.bbb_client_ws,
		bm.bbb_pad_url,
		bm.bbb_pad_ws,
		bm.bbb_api_url,
		bm.bbb_api_secret,
		bm.bbb_webrtc_ws,
		bm.transcription_host,
		bm.transcription_port,
		bm.transcription_secret,
		bm.translation_server_url,
		bm.changeset_external,
		bm.changeset_port,
		bm.changeset_host,
		Transcribe,
	)
	bm.lock.Lock()
	defer bm.lock.Unlock()

	bm.bots[new_bot.id] = new_bot

	return new_bot, nil
}

func (bm *BotManager) RemoveBot(botID string) {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	if bot, ok := bm.bots[botID]; ok {
		bot.Disconnect()
		delete(bm.bots, botID)
	}
}

func (bm *BotManager) GetBot(botID string) (*Bot, bool) {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	if bot, ok := bm.bots[botID]; ok {
		return bot, true
	}
	return nil, false
}

func (bm *BotManager) GetBots() map[string]*Bot {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	return bm.bots
}

// enum task [transcribe, translate]
type Task int

const (
	Transcribe Task = iota
	Translate
)

type StatusType int

const (
	Connected StatusType = iota
	Connecting
	Disconnected
)

type Bot struct {
	id           string
	status       StatusType
	client       *bbbbot.Client
	clients      []*bbbbot.Client
	clientsMutex sync.Mutex
	streamclient *StreamClient
	audioclient  *bbbbot.AudioClient
	oggFile      *oggwriter.OggWriter
	en_caption   *pad.Pad

	bbb_client_url         string
	bbb_client_ws          string
	bbb_pad_url            string
	bbb_pad_ws             string
	bbb_api_url            string
	bbb_api_secret         string
	bbb_webrtc_ws          string
	transcription_host     string
	transcription_port     int
	transcription_secret   string
	translation_server_url string
	changeset_external     bool
	changeset_port         int
	changeset_host         string
	task                   Task

	meetingID string
	userName  string
	moderator bool
}

func NewBot(
	bbb_client_url string,
	bbb_client_ws string,
	bbb_pad_url string,
	bbb_pad_ws string,
	bbb_api_url string,
	bbb_api_secret string,
	bbb_webrtc_ws string,
	transcription_host string,
	transcription_port int,
	transcription_secret string,
	translation_server_url string,
	changeset_external bool,
	changeset_port int,
	changeset_host string,
	task Task,
) *Bot {
	client, err := bbbbot.NewClient(
		bbb_client_url,
		bbb_client_ws,
		bbb_pad_url,
		bbb_pad_ws,
		bbb_api_url,
		bbb_api_secret,
		bbb_webrtc_ws,
	)
	if err != nil {
		panic(err)
	}

	streamclient := NewStreamClient(transcription_host, transcription_port, true, transcription_secret)

	// Create obj
	return &Bot{
		id:           uuid.New().String(),
		status:       Disconnected,
		task:         task,
		client:       client,
		clients:      make([]*bbbbot.Client, 0),
		streamclient: streamclient,
		audioclient:  client.CreateAudioChannel(),
		oggFile:      nil,
		en_caption:   nil,

		bbb_client_url:         bbb_client_url,
		bbb_client_ws:          bbb_client_ws,
		bbb_pad_url:            bbb_pad_url,
		bbb_pad_ws:             bbb_pad_ws,
		bbb_api_url:            bbb_api_url,
		bbb_api_secret:         bbb_api_secret,
		bbb_webrtc_ws:          bbb_webrtc_ws,
		translation_server_url: translation_server_url,
		changeset_port:         changeset_port,
		changeset_host:         changeset_host,
		changeset_external:     changeset_external,

		meetingID: "",
		userName:  "",
		moderator: true,
	}
}

func (b *Bot) Join(
	meetingID string,
	UserName string,
) error {
	if b.status == Connecting {
		// return error connecting
		return fmt.Errorf("already connecting")
	}

	if b.status == Connected {
		b.Disconnect()
	}
	b.status = Connecting
	defer func() {
		b.status = Connected
	}()

	b.meetingID = meetingID
	b.userName = UserName

	err := b.client.Join(b.meetingID, b.userName, b.moderator)
	if err != nil {
		return err
	}

	b.en_caption, err = b.client.CreateCapture(bbbbot.Language("en"), b.changeset_external, b.changeset_host, b.changeset_port)
	if err != nil {
		return err
	}

	b.streamclient.OnConnected(func(message string) {
		fmt.Println("Connected to server.")
	})

	b.streamclient.OnDisconnected(func(message string) {
		fmt.Println("Disconnected from server.")
		b.client.Leave()
	})

	b.streamclient.OnTimeout(func(message string) {
		fmt.Println("Connection to server timed out.")
		b.client.Leave()
	})

	b.streamclient.OnTCPMessage(func(text string) {
		fmt.Println("TCP message event:", text)
		validtext := strings.ToValidUTF8(text, "")

		if b.task == Transcribe {
			// use the english capture
			captures := b.client.GetCaptures()
			for _, capture := range captures {
				if capture.ShortLanguageName == "en" {
					err := capture.SetText(validtext)
					if err != nil {
						fmt.Println("Error in pad write:", err)
					}
				}
			}
		} else if b.task == Translate {
			// use the english capture to set the text
			captures := b.client.GetCaptures()
			for _, capture := range captures {
				if capture.ShortLanguageName == "en" {
					err := capture.SetText(validtext)
					if err != nil {
						fmt.Println("Error in pad write:", err)
					}
				}
			}
			// use the other captures to set the text
			captures = b.client.GetCaptures()
			for _, capture := range captures {
				if capture.ShortLanguageName != "en" {
					translatedText, err := translate(b.translation_server_url, validtext, "en", capture.ShortLanguageName)
					if err != nil {
						fmt.Println("Error in translation:", err)
					}
					err = capture.SetText(translatedText)
					if err != nil {
						fmt.Println("Error in pad write:", err)
					}
				}
			}
		}

		b.clientsMutex.Lock()
		clientsTemp := b.clients
		b.clientsMutex.Unlock()
		if clientsTemp != nil {
			for _, cl := range clientsTemp {
				pads := cl.GetCaptures()
				for _, pad := range pads {
					translatedText := validtext
					var err error
					if pad.ShortLanguageName != "en" {
						translatedText, err = translate(b.translation_server_url, validtext, "en", pad.ShortLanguageName)
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

	err = b.streamclient.Connect()
	if err != nil {
		return err
	}

	b.audioclient = b.client.CreateAudioChannel()

	err = b.audioclient.ListenToAudio()
	if err != nil {
		panic(err)
	}

	b.oggFile, err = oggwriter.NewWith(b.streamclient, 48000, 2)
	if err != nil {
		panic(err)
	}

	b.audioclient.OnTrack(func(status *bbbbot.StatusType, track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
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

				if *status == bbbbot.DISCONNECTED {
					b.oggFile.Close()
					b.streamclient.Close()
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

				if err := b.oggFile.WriteRTP(rtpPacket); err != nil {
					fmt.Println("Error during OGG file write:", err)
					return
				}
			}
		}()
	})

	return nil
}

func (b *Bot) Disconnect() {
	if b.streamclient != nil {
		b.streamclient.Close()
	}
	if b.audioclient != nil {
		b.client.Leave()
	}
	if b.en_caption != nil {
		b.audioclient.Close()
	}
	if b.client != nil {
		b.oggFile.Close()
	}

	// diconnect all clients
	b.clientsMutex.Lock()
	for _, cl := range b.clients {
		cl.Leave()
	}
	b.clients = b.clients[:0]
	b.clientsMutex.Unlock()
}

func (b *Bot) Translate(
	targetLang string,
) error {
	if b.status == Connecting {
		// return error connecting
		return fmt.Errorf("bot is connecting")
	}

	if b.status == Disconnected {
		return fmt.Errorf("bot is disconnected")
	}

	// create a new client, join meeting and create capture
	new_client, err := bbbbot.NewClient(
		b.bbb_client_url,
		b.bbb_client_ws,
		b.bbb_pad_url,
		b.bbb_pad_ws,
		b.bbb_api_url,
		b.bbb_api_secret,
		b.bbb_webrtc_ws,
	)
	if err != nil {
		return err
	}

	// join the meeting
	new_client.Join(b.meetingID, b.userName+"-"+targetLang, b.moderator)

	// create a new capture
	_, err = new_client.CreateCapture(bbbbot.Language(targetLang), b.changeset_external, b.changeset_host, b.changeset_port)
	if err != nil {
		return err
	}

	// add new client to the list of clients
	b.clientsMutex.Lock()
	b.clients = append(b.clients, new_client)
	b.clientsMutex.Unlock()

	return nil
}
