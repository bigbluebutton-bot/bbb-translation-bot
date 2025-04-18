package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	bbbbot "github.com/bigbluebutton-bot/bigbluebutton-bot"
	bbbapi "github.com/bigbluebutton-bot/bigbluebutton-bot/api"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/go-chi/chi/v5"
)

// -----------------------------------------------------------------------------
// Globals
// -----------------------------------------------------------------------------

var (
	conf    *Settings
	BM      *BotManager
	bbb_api *bbbapi.ApiRequest
)

// -----------------------------------------------------------------------------
// CLI options
// -----------------------------------------------------------------------------

type Options struct {
	Port int `help:"Port to listen on" short:"p" default:"8080"`
}

// -----------------------------------------------------------------------------
// Typed I/O structs
// -----------------------------------------------------------------------------

type statusResponse struct {
	BotsCount int `json:"bots_count"`
	MaxBots   int `json:"max_bots"`
}

type StatusOutput struct{ Body statusResponse }
type MeetingsOutput struct{ Body []bbbapi.Meeting }
type MeetingOutput struct{ Body bbbapi.Meeting }
type LanguagesOutput struct{ Body map[string]string }
type BotsOutput struct{ Body map[string]*Bot }
type BotOutput struct{ Body *Bot }

// -----------------------------------------------------------------------------
// Huma route registration
// -----------------------------------------------------------------------------

func addRoutes(api huma.API) {
	// -------------------------------------------------------------------------
	// System
	// -------------------------------------------------------------------------
	huma.Register(api, huma.Operation{
		OperationID: "get-status",
		Method:      http.MethodGet,
		Path:        "/api/v1/status",
		Summary:     "Get server status",
		Tags:        []string{"System"},
	}, func(_ context.Context, _ *struct{}) (*StatusOutput, error) {
		return &StatusOutput{
			Body: statusResponse{
				BotsCount: len(BM.GetBots()),
				MaxBots:   BM.Max_bots,
			},
		}, nil
	})

	// -------------------------------------------------------------------------
	// BBB meetings
	// -------------------------------------------------------------------------
	huma.Register(api, huma.Operation{
		OperationID: "get-meetings",
		Method:      http.MethodGet,
		Path:        "/api/v1/bbb/meetings",
		Summary:     "List all BBB meetings",
		Tags:        []string{"BBB"},
	}, func(_ context.Context, _ *struct{}) (*MeetingsOutput, error) {
		meetings, err := bbb_api.GetMeetings()
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "Failed to fetch meetings")
		}
		list := make([]bbbapi.Meeting, 0, len(meetings))
		for _, m := range meetings {
			list = append(list, m)
		}
		return &MeetingsOutput{Body: list}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-meeting",
		Method:      http.MethodGet,
		Path:        "/api/v1/bbb/meeting/{meeting_id}",
		Summary:     "Get a single meeting",
		Tags:        []string{"BBB"},
	}, func(_ context.Context, input *struct {
		MeetingID string `path:"meeting_id" doc:"Meeting ID"`
	}) (*MeetingOutput, error) {
		meetings, err := bbb_api.GetMeetings()
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "Failed to fetch meeting")
		}
		meeting, ok := meetings[input.MeetingID]
		if !ok {
			return nil, huma.NewError(http.StatusNotFound, "Meeting not found")
		}
		return &MeetingOutput{Body: meeting}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "delete-meeting",
		Method:        http.MethodDelete,
		Path:          "/api/v1/bbb/meeting/{meeting_id}",
		Summary:       "End a meeting",
		Tags:          []string{"BBB"},
		DefaultStatus: http.StatusNoContent,
	}, func(_ context.Context, input *struct {
		MeetingID string `path:"meeting_id" doc:"Meeting ID"`
	}) (*struct{}, error) {
		if _, err := bbb_api.EndMeeting(input.MeetingID); err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "Failed to end meeting")
		}
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-languages",
		Method:      http.MethodGet,
		Path:        "/api/v1/bbb/languages",
		Summary:     "List supported languages",
		Tags:        []string{"BBB"},
	}, func(_ context.Context, _ *struct{}) (*LanguagesOutput, error) {
		codes := bbbbot.AllLanguages()
		m := make(map[string]string, len(codes))
		for _, c := range codes {
			m[string(c)] = bbbbot.LanguageShortToName(c)
		}
		return &LanguagesOutput{Body: m}, nil
	})

	// -------------------------------------------------------------------------
	// Bots
	// -------------------------------------------------------------------------
	huma.Register(api, huma.Operation{
		OperationID: "get-bots",
		Method:      http.MethodGet,
		Path:        "/api/v1/bots",
		Summary:     "List all bots",
		Tags:        []string{"Bots"},
	}, func(_ context.Context, _ *struct{}) (*BotsOutput, error) {
		return &BotsOutput{Body: BM.GetBots()}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-bot",
		Method:      http.MethodGet,
		Path:        "/api/v1/bot/{bot_id}",
		Summary:     "Get bot details",
		Tags:        []string{"Bots"},
	}, func(_ context.Context, input *struct {
		BotID string `path:"bot_id" doc:"Bot ID"`
	}) (*BotOutput, error) {
		bot, ok := BM.GetBot(input.BotID)
		if !ok {
			return nil, huma.NewError(http.StatusNotFound, "Bot not found")
		}
		return &BotOutput{Body: bot}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "bot-join",
		Method:        http.MethodPost,
		Path:          "/api/v1/bot/join/{meeting_id}",
		Summary:       "Create a bot and join a meeting",
		Tags:          []string{"Bots"},
		DefaultStatus: http.StatusOK,
	}, func(_ context.Context, input *struct {
		MeetingID string `path:"meeting_id" doc:"Meeting ID"`
	}) (*BotOutput, error) {
		// check if there is already a bot in this meeting
		for _, bot := range BM.GetBots() {
			if bot.MeetingID == input.MeetingID {
				return nil, huma.NewError(http.StatusConflict, "Bot already in meeting")
			}
		}

		if len(BM.GetBots()) >= BM.Max_bots {
			return nil, huma.NewError(http.StatusTooManyRequests, "Max bots limit reached")
		}

		if meetings, err := bbb_api.GetMeetings(); err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "Failed to fetch meetings")
		} else if _, ok := meetings[input.MeetingID]; !ok {
			return nil, huma.NewError(http.StatusNotFound, "Meeting not found")
		}

		bot, err := BM.AddBot()
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "Failed to create bot")
		} else if err := bot.Join(input.MeetingID, "Bot"); err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "Failed to join meeting")
		}
		return &BotOutput{Body: bot}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "bot-leave",
		Method:        http.MethodPost,
		Path:          "/api/v1/bot/{bot_id}/leave",
		Summary:       "Bot leaves its meeting",
		Tags:          []string{"Bots"},
		DefaultStatus: http.StatusOK,
	}, func(_ context.Context, input *struct {
		BotID string `path:"bot_id" doc:"Bot ID"`
	}) (*struct{}, error) {
		_, ok := BM.GetBot(input.BotID)
		if !ok {
			return nil, huma.NewError(http.StatusNotFound, "Bot not found")
		}
		BM.RemoveBot(input.BotID)

		return nil, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "bot-set-task",
		Method:        http.MethodPut,
		Path:          "/api/v1/bot/{bot_id}/task/{task}",
		Summary:       "Set bot task (transcribe/translate)",
		Tags:          []string{"Bots"},
		DefaultStatus: http.StatusOK,
	}, func(_ context.Context, input *struct {
		BotID string `path:"bot_id"  doc:"Bot ID"`
		Task  string `path:"task"    enum:"transcribe,translate" doc:"Task type"`
	}) (*struct{}, error) {
		bot, ok := BM.GetBot(input.BotID)
		if !ok {
			return nil, huma.NewError(http.StatusNotFound, "Bot not found")
		}
		var task Task
		switch input.Task {
		case "transcribe":
			task = Transcribe
		case "translate":
			task = Translate
		default:
			return nil, huma.NewError(http.StatusBadRequest, "Invalid task type")
		}
		bot.SetTask(task)
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "bot-translate-start",
		Method:        http.MethodPut,
		Path:          "/api/v1/bot/{bot_id}/translate/{lang}",
		Summary:       "Start translation",
		Tags:          []string{"Bots"},
		DefaultStatus: http.StatusOK,
	}, func(_ context.Context, input *struct {
		BotID string `path:"bot_id" doc:"Bot ID"`
		Lang  string `path:"lang"  doc:"Language code"`
	}) (*struct{}, error) {
		bot, ok := BM.GetBot(input.BotID)
		if !ok {
			return nil, huma.NewError(http.StatusNotFound, "Bot not found")
		}
		if !isValidLanguage(input.Lang) {
			return nil, huma.NewError(http.StatusBadRequest, "Invalid language code")
		}

		//check if the bot is already translating this language
		all_translations := bot.GetAllActiveTranslations()
		for _, t := range all_translations {
			if t == input.Lang {
				return nil, huma.NewError(http.StatusConflict, "Bot already translating this language")
			}
		}

		// check the task
		if bot.GetTask() != Translate {
			return nil, huma.NewError(http.StatusBadRequest, "Bot is not in translate mode")
		}

		if err := bot.Translate(input.Lang); err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "Failed to start translation")
		}
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "bot-translate-stop",
		Method:        http.MethodDelete,
		Path:          "/api/v1/bot/{bot_id}/translate/{lang}",
		Summary:       "Stop translation",
		Tags:          []string{"Bots"},
		DefaultStatus: http.StatusOK,
	}, func(_ context.Context, input *struct {
		BotID string `path:"bot_id" doc:"Bot ID"`
		Lang  string `path:"lang"  doc:"Language code"`
	}) (*struct{}, error) {
		bot, ok := BM.GetBot(input.BotID)
		if !ok {
			return nil, huma.NewError(http.StatusNotFound, "Bot not found")
		}
		if !isValidLanguage(input.Lang) {
			return nil, huma.NewError(http.StatusBadRequest, "Invalid language code")
		}
		// check if this language is being translated
		all_languages := bot.GetAllActiveTranslations()
		found := false
		for _, t := range all_languages {
			if t == input.Lang {
				found = true
				break
			}
		}
		if !found {
			return nil, huma.NewError(http.StatusNotFound, "Language ist not actively being translated")
		}

		if err := bot.StopTranslate(input.Lang); err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "Failed to stop translation")
		}
		return nil, nil
	})
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func isValidLanguage(lang string) bool {
	for _, c := range bbbbot.AllLanguages() {
		if string(c) == lang {
			return true
		}
	}
	return false
}

func healthCheck(c *Settings) {
	url := "http://" + c.TranscriptionServer.ExternalHost + ":" +
		strconv.Itoa(c.TranscriptionServer.HealthCheckPort) + "/health"

	for {
		if _, err := http.Get(url); err != nil {
			fmt.Printf("Transcription server is down (%s). Retrying in 5 seconds...\n", url)
			time.Sleep(5 * time.Second)
		} else {
			fmt.Println("Transcription server is up")
			break
		}
	}
}

// -----------------------------------------------------------------------------
// main
// -----------------------------------------------------------------------------

func main() {
	cli := humacli.New(func(hooks humacli.Hooks, opt *Options) {
		// ---------------------------------------------------------------------
		// Initialise settings, external services & state
		// ---------------------------------------------------------------------
		var err error
		conf, err = LoadSettings()
		if err != nil {
			panic(err)
		}
		healthCheck(conf)

		bbb_api, err = bbbapi.NewRequest(conf.BBB.API.URL, conf.BBB.API.Secret, conf.BBB.API.SHA)
		if err != nil {
			panic(err)
		}

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

		// ---------------------------------------------------------------------
		// Router & API
		// ---------------------------------------------------------------------
		router := chi.NewMux()
		api := humachi.New(router, huma.DefaultConfig("BBB Bot API", "1.0.0"))
		addRoutes(api)

		// Serve static assets from ./public
		router.Mount("/", http.StripPrefix("/", http.FileServer(http.Dir("./public"))))

		// ---------------------------------------------------------------------
		// Start server
		// ---------------------------------------------------------------------
		hooks.OnStart(func() {
			addr := fmt.Sprintf(":%d", opt.Port)
			log.Printf("Server starting on %s ...", addr)
			log.Fatal(http.ListenAndServe(addr, router))
		})
	})

	cli.Run()
}
