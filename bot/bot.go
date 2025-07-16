package main

import (
	"encoding/json"
	"fmt"
	"log"
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

	lock sync.Mutex
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
		Max_bots:               max_bots,
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
		log.Printf("[ERROR] Max bots reached: %d", bm.Max_bots)
		return nil, fmt.Errorf("max bots reached: %d", bm.Max_bots)
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
		TaskTranscribe,
	)
	bm.lock.Lock()
	defer bm.lock.Unlock()

	bm.bots[new_bot.ID] = new_bot

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

func (bm *BotManager) Bot(botID string) (*Bot, bool) {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	if bot, ok := bm.bots[botID]; ok {
		return bot, true
	}
	return nil, false
}

func (bm *BotManager) Bots() map[string]*Bot {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	return bm.bots
}

// enum task [transcribe, translate]
type Task int

const (
	TaskTranscribe Task = iota
	TaskTranslate
)

type StatusType int

const (
	Connected StatusType = iota
	Connecting
	Disconnected
)

type Bot struct {
	ID           string     `json:"id"`
	Status       StatusType `json:"status"`
	client       *bbbbot.Client
	clients      map[string]*bbbbot.Client // map of language and client
	captures     map[string]*pad.Pad       // map of language and capture
	Sub_bots     int                       `json:"sub_bots"`
	Languages    []string                  `json:"languages"`
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
	Task                   Task `json:"task"`

	MeetingID string `json:"meeting_id"`
	UserName  string `json:"user_name"`
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
	return_bot := &Bot{
		ID:           uuid.New().String(),
		Status:       Disconnected,
		Task:         task,
		client:       client,
		clients:      make(map[string]*bbbbot.Client),
		captures:     make(map[string]*pad.Pad),
		Languages:    make([]string, 0),
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

		MeetingID: "",
		UserName:  "",
		moderator: true,
	}
	return_bot.Languages = append(return_bot.Languages, "en")
	return return_bot
}

func (b *Bot) Join(
	meetingID string,
	UserName string,
) error {
	if b.Status == Connecting {
		// return error connecting
		return fmt.Errorf("already connecting")
	}

	if b.Status == Connected {
		b.Disconnect()
	}
	b.Status = Connecting
	defer func() {
		b.Status = Connected
	}()

	b.MeetingID = meetingID
	b.UserName = UserName

	err := b.client.Join(b.MeetingID, b.UserName, b.moderator)
	if err != nil {
		return err
	}

	b.en_caption, err = b.client.CreateCapture(bbbbot.Language("en"), b.changeset_external, b.changeset_host, b.changeset_port)
	if err != nil {
		return err
	}

	b.en_caption.OnDisconnect(func() {
		log.Println("En caption disconnected")
		b.Disconnect()
	})

	b.streamclient.OnConnected(func(message string) {
		log.Println("Connected to server.")
	})

	b.streamclient.OnDisconnected(func(message string) {
		log.Println("Disconnected from server.")
		b.client.Leave()
	})

	b.streamclient.OnTimeout(func(message string) {
		log.Println("Connection to server timed out.")
		b.client.Leave()
	})

	b.streamclient.OnTCPMessage(func(text string) {
		log.Println("TCP message event:", text)
		validtext := strings.ToValidUTF8(text, "")

		if b.Task == TaskTranscribe {
			// use the english capture
			captures := b.client.GetCaptures()
			for _, capture := range captures {
				if capture.ShortLanguageName == "en" {
					err := capture.SetText(validtext)
					if err != nil {
						log.Println("Error in pad write:", err)
					}
				}
			}
		} else if b.Task == TaskTranslate {
			// use the english capture to set the text
			captures := b.client.GetCaptures()
			for _, capture := range captures {
				if capture.ShortLanguageName == "en" {
					err := capture.SetText(validtext)
					if err != nil {
						log.Println("Error in pad write:", err)
					}
				}
			}
			// use the other captures to set the text
			clients := b.clients
			for _, client := range clients {
				captures = client.GetCaptures()
				for _, capture := range captures {
					if capture.ShortLanguageName != "en" {
						translatedText, err := translate(b.translation_server_url, validtext, "en", capture.ShortLanguageName)
						if err != nil {
							log.Println("Error in translation:", err)
						}
						err = capture.SetText(translatedText)
						if err != nil {
							log.Println("Error in pad write:", err)
						}
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

		log.Println("ID: " + track.ID())
		log.Println("Kind: " + track.Kind().String())
		log.Println("StreamID: " + track.StreamID())
		log.Println("SSRC: " + fmt.Sprint(track.SSRC()))
		log.Println("Codec: " + track.Codec().MimeType)
		log.Println("Codec PayloadType: " + fmt.Sprint(track.Codec().PayloadType))
		log.Println("Codec ClockRate: " + fmt.Sprint(track.Codec().ClockRate))
		log.Println("Codec Channels: " + fmt.Sprint(track.Codec().Channels))
		log.Println("Codec MimeType: " + track.Codec().MimeType)

		go func() {
			buffer := make([]byte, 1024)
			defer b.oggFile.Close()
			defer b.streamclient.Close()
			for {
				n, _, readErr := track.Read(buffer)

				if *status == bbbbot.DISCONNECTED {
					return
				}

				if readErr != nil {
					log.Println("Error during audio track read:", readErr)
					return
				}

				rtpPacket := &rtp.Packet{}
				if err := rtpPacket.Unmarshal(buffer[:n]); err != nil {
					log.Println("Error during RTP packet unmarshal:", err)
					return
				}

				if err := b.oggFile.WriteRTP(rtpPacket); err != nil {
					if *status == bbbbot.DISCONNECTED {
						return
					}

					log.Println("Error during OGG file write:", err)
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

	b.clientsMutex.Lock()
	for lang, cl := range b.clients {
		cl.Leave()
		if cap, ok := b.captures[lang]; ok {
			cap.Disconnect()
			delete(b.captures, lang)
		}
	}
	for k := range b.clients {
		delete(b.clients, k)
	}
	b.clientsMutex.Unlock()
}

type taskRequest struct {
	Task string `json:"task"`
}

func (b *Bot) Translate(
	targetLang string,
) error {
	if b.Status == Connecting {
		// return error connecting
		return fmt.Errorf("bot is connecting")
	}

	if b.Status == Disconnected {
		return fmt.Errorf("bot is disconnected")
	}

	if b.Task != TaskTranslate {
		return fmt.Errorf("bot is not in translate mode")
	}

	// check if language is already in use
	if _, ok := b.clients[targetLang]; ok {
		b.clients[targetLang].Leave()
		delete(b.clients, targetLang)
		if capture, ok := b.captures[targetLang]; ok {
			capture.Disconnect()
			delete(b.captures, targetLang)
		}
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
	new_client.Join(b.MeetingID, b.UserName+"-"+targetLang, b.moderator)

	// create a new capture
	new_capture, err := new_client.CreateCapture(bbbbot.Language(targetLang), b.changeset_external, b.changeset_host, b.changeset_port)
	if err != nil {
		return err
	}

	new_capture.OnDisconnect(func() {
		log.Printf("New capture %s disconnected", targetLang)
		b.StopTranslate(targetLang)
	})

	// add new client to the list of clients
	b.clientsMutex.Lock()
	b.clients[targetLang] = new_client
	b.captures[targetLang] = new_capture

	// if language code in the list, remove it
	skip := false
	for _, lang := range b.Languages {
		if lang == targetLang {
			// remove it from the list
			skip = true
		}
	}
	if !skip {
		b.Languages = append(b.Languages, targetLang)
	}
	b.clientsMutex.Unlock()

	return nil
}

func (b *Bot) GetAllActiveTranslations() []string {
	return b.Languages
}

func (b *Bot) StopTranslate(
	targetLang string,
) error {
	if targetLang == "en" {
		// switch to transcription mode
		b.SetTask(TaskTranscribe)
		return nil
	}

	b.clientsMutex.Lock()
	defer b.clientsMutex.Unlock()

	if client, ok := b.clients[targetLang]; ok {
		// check if client is connected
		client.Leave()
		delete(b.clients, targetLang)
		if capture, ok := b.captures[targetLang]; ok {
			capture.Disconnect()
			delete(b.captures, targetLang)
		}
		// remove language from list
		for i, lang := range b.Languages {
			if lang == targetLang {
				b.Languages = append(b.Languages[:i], b.Languages[i+1:]...)
				break
			}
		}
		// delete client from clients map
		for k := range b.clients {
			if k == targetLang {
				delete(b.clients, k)
			}
		}

		return nil
	}
	return fmt.Errorf("client not found")
}

func (b *Bot) GetTask() Task {
	return b.Task
}

func (b *Bot) SetTask(task Task) {
	if b.Task == TaskTranslate && task == TaskTranscribe {
		// stop all clients
		for _, cl := range b.clients {
			cl.Leave()
		}

		// send task to transcription server
		task_req := taskRequest{
			Task: "transcribe",
		}
		task_req_json, err := json.Marshal(task_req)
		if err != nil {
			return
		}
		err = b.streamclient.SendTCPMessage(string(task_req_json))
		if err != nil {
			return
		}
	}

	if b.Task == TaskTranscribe && task == TaskTranslate {
		// send task to transcription server
		task_req := taskRequest{
			Task: "translate",
		}
		task_req_json, err := json.Marshal(task_req)
		if err != nil {
			log.Println("Error in task request json:", err)
			return
		}
		err = b.streamclient.SendTCPMessage(string(task_req_json))
		if err != nil {
			log.Println("Error in task request send:", err)
			return
		}

		all_languages := b.GetAllActiveTranslations()

		// stop all clients
		for k := range b.clients {
			delete(b.clients, k)
		}

		b.Task = task

		// start all clients
		for _, lang := range all_languages {
			// skip en
			if lang == "en" {
				continue
			}

			log.Println("Starting translation for language:", lang)
			err := b.Translate(lang)
			if err != nil {
				log.Println("Error in translate:", err)
			}
		}
	}

	b.Task = task
}
