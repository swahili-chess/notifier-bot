package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"

	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
	"github.com/swahili-chess/notifier-bot/config"
	"github.com/swahili-chess/notifier-bot/internal/lichess"
	polling "github.com/swahili-chess/notifier-bot/internal/polling"
	"github.com/swahili-chess/notifier-bot/internal/req"
)

const (
	start_txt       = "Use this bot to get link of games of Chesswahili team members that are actively playing on Lichess. Type /stop to stop receiving notifications`"
	stop_txt        = "Sorry to see you leave You wont be receiving notifications. Type /start to receive"
	unknown_cmd     = "I don't know that command"
	maintenance_txt = "We are having Bot maintenance. Service will resume shortly"
)

func init() {

	var programLevel = new(slog.LevelVar)
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: programLevel})
	slog.SetDefault(slog.New(h))

}

// telegram bot user struct
type TgBotUser struct {
	ID       int64 `json:"id"`
	Isactive bool  `json:"isactive"`
}

func main() {

	var is_maintenance_txt = false

	flag.StringVar(&config.Cfg.Url, "api-url", os.Getenv("API_URL"), "API URL")
	flag.StringVar(&config.Cfg.BotToken, "bot-token", os.Getenv("TG_BOT_TOKEN"), "Bot Token")
	flag.StringVar(&config.Cfg.BasicAuth.USERNAME, "basicauth-username", os.Getenv("BASICAUTH_USERNAME"), "basicauth-username")
	flag.StringVar(&config.Cfg.BasicAuth.PASSWORD, "basicauth-password", os.Getenv("BASICAUTH_PASSWORD"), "basicauth-password")

	flag.Parse()

	if config.Cfg.BotToken == "" || config.Cfg.Url == "" {
		slog.Error("main: Bot token or API url not provided")
		return
	}

	bot, err := tgbotapi.NewBotAPI(config.Cfg.BotToken)
	if err != nil {
		slog.Error("main: Failed to create bot api instance", "error", err)
		return
	}

	links := make(map[string]time.Time)

	swbot := &polling.SWbot{
		Bot:   bot,
		Links: &links,
	}

	u := tgbotapi.NewUpdate(0)

	u.Timeout = 60

	membersIdsChan := make(chan []lichess.MemberDB)

	updates := bot.GetUpdatesChan(u)

	//Fetch from the team for the first time
	memberIds := lichess.FetchTeamMembers()
	if len(memberIds) == 0 {
		slog.Error("main: Length of player ids shouldn't be 0")
	}
	swbot.AddNewLichessTeamMembers(memberIds)

	go swbot.PollAndUpdateTeamMembers(membersIdsChan)

	go swbot.PollAndUpdateMemberStatus(membersIdsChan, &memberIds)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if !update.Message.IsCommand() {
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		switch update.Message.Command() {
		case "start":
			msg.Text = start_txt
			botUser := TgBotUser{
				ID:       update.Message.From.ID,
				Isactive: true,
			}
			var errResponse req.ErrorResponse

			statusCode, err := req.PostOrPutRequest(http.MethodPost, fmt.Sprintf("%s/telegram/bot/users", config.Cfg.Url), botUser, &errResponse)
			if statusCode == http.StatusInternalServerError {
				switch {
				case errResponse.Error == `pq: duplicate key value violates unique constraint "tgbot_users_pkey"`:
					args := TgBotUser{
						ID:       botUser.ID,
						Isactive: botUser.Isactive,
					}

					statusCode, err := req.PostOrPutRequest(http.MethodPut, fmt.Sprintf("%s/telegram/bot/users", config.Cfg.Url), args, &errResponse)
					if statusCode == http.StatusInternalServerError {
						slog.Error("main: Failed to update bot user", "error", errResponse.Error)
					} else if err != nil {
						slog.Error("main : Failed to update bot user", "error", err)
					}

				default:
					slog.Error("main: Failed to insert bot user", "error", err)
				}
			} else if err != nil {
				slog.Error("main: Failed to insert bot user", "error", err, "statuscode", statusCode)
			}

		case "stop":
			botUser := TgBotUser{
				ID:       update.Message.From.ID,
				Isactive: false,
			}
			var errResponse req.ErrorResponse

			statusCode, err := req.PostOrPutRequest(http.MethodPut, fmt.Sprintf("%s/telegram/bot/users", config.Cfg.Url), botUser, &errResponse)
			if statusCode == http.StatusInternalServerError {
				slog.Error("main: Failed to update bot user", "error", errResponse.Error)
			} else if err != nil {
				slog.Error("main: Failed to update bot user", "error", err)
			}

			msg.Text = stop_txt

		case "subs":
			var res []int64
			var errResponse req.ErrorResponse
			statusCode, err := req.GetRequest(fmt.Sprintf("%s/telegram/bot/users/active", config.Cfg.Url), &res, &errResponse)
			if statusCode == http.StatusInternalServerError {
				slog.Error("main: Failed to get telegram bot users", "error", errResponse.Error)

			} else if statusCode != http.StatusOK || err != nil {
				slog.Error("main: Failed to get telegram bot users", "error", err, "statusCode", statusCode)
			}

			msg.Text = fmt.Sprintf("There are %d subscribers in chesswahiliBot", len(res))

		case "ml":
			msg.Text = fmt.Sprintf("There are %d in a map so far.", len(*swbot.Links))

		case "sm":
			if polling.Master_ID == update.Message.From.ID {
				is_maintenance_txt = true
			}

		case "help":
			msg.Text = `
			Commands for this @chesswahiliBot bot are:
			
			/start  start the bot (i.e., enable receiving of the game links)
			/stop   stop the bot (i.e., disable receiving of the game links)
			/subs   subscribers for the bot
			/ml     current map length
			/help   this help text
			/sm     send maintenace message for @Hopertz only.
			`

		default:
			msg.Text = unknown_cmd
		}

		if is_maintenance_txt {
			swbot.NotifyUsersOfMaintenance(maintenance_txt)
			is_maintenance_txt = false

		} else {
			if _, err := swbot.Bot.Send(msg); err != nil {
				slog.Error("main: Failed to send msg", "error", err, "msg", msg)
			}
		}

	}
}
