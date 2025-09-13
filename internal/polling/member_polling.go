package poll

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/swahili-chess/notifier-bot/config"
	"github.com/swahili-chess/notifier-bot/internal/lichess"
	"github.com/swahili-chess/notifier-bot/internal/req"
)

const (
	withGameIds      = "&withGameIds=true"
	urlStatus        = "https://lichess.org/api/users/status?ids="
	base_url         = "https://lichess.org/"
	minLinkStayInMap = 1 * time.Hour
	cleanUpTime      = 30 * time.Minute
	Master_ID        = 731217828
)

type Member struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Title     string `json:"title,omitempty"`
	Online    bool   `json:"online"`
	Playing   bool   `json:"playing"`
	Streaming bool   `json:"streaming"`
	Patron    bool   `json:"patron"`
	PlayingId string `json:"playingId"`
}

// pollAndUpdateMemberStatus periodically polls for member data, updates the member list,
// and fetches member status when available. It also handles removing expired game links in a separate goroutine.
// It either receives from chan and update team members or ids or it fetch status of current member ids
func (sw *SWbot) PollAndUpdateMemberStatus(membersIdsChan <-chan []lichess.MemberDB, membersIds *[]lichess.MemberDB) {

	ticker := time.NewTicker(time.Second * 6)
	defer ticker.Stop()

	go sw.removeExpiredGameLinks(sw.Links)

	for range ticker.C {
		select {

		case memberIdsLists := <-membersIdsChan:
			if len(memberIdsLists) != 0 {
				*membersIds = memberIdsLists
			}

		default:
			url := buildMemberStatusesURL(*membersIds, urlStatus)
			if url != "" {
				sw.fetchAndUpdateMemberStatuses(url, sw.Links)
			}
		}

	}
}

// fetchAndUpdateMemberStatuses fetches the statuses of team members from the provided URL,
// checks whether they are currently playing, and updates the `links` map with the playing member IDs.
// If a new playing member is found, it sends a notification to active Telegram bot users.
func (sw *SWbot) fetchAndUpdateMemberStatuses(url string, links *map[string]time.Time) {
	var members []Member

	var client = &http.Client{
		Timeout: time.Second * 10,
	}

	resp, err := client.Get(url)
	if err != nil {
		slog.Error("Error while fetching Member status details", "err", err)
		return
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&members)
	if err != nil {
		slog.Error("Error decoding the json body", "err", err)
		return
	}

	for _, member := range members {

		// Sometimes the playingId is empty string
		if len(member.PlayingId) != 0 {

			sw.mu.RLock()
			_, idExists := (*links)[member.PlayingId]
			sw.mu.RUnlock()

			if !idExists {
				sw.mu.Lock()
				(*links)[member.PlayingId] = time.Now()
				sw.mu.Unlock()

				sw.sendLinkToActiveTelegramIds(member.PlayingId)
			}

		}
	}

}

// prepare the url to fetch the status of the members
func buildMemberStatusesURL(members []lichess.MemberDB, urlStatus string) string {

	if len(members) == 0 {
		return ""
	}
	membersIds := []string{}
	for _, member := range members {
		membersIds = append(membersIds, member.LichessID)
	}

	var urlBuilder strings.Builder
	urlBuilder.WriteString(urlStatus)
	urlBuilder.WriteString(strings.Join(membersIds, ","))
	urlBuilder.WriteString(withGameIds)

	return urlBuilder.String()

}

// delete game links that have stayed in the map for more than 1 hour
func (sw *SWbot) removeExpiredGameLinks(links *map[string]time.Time) {

	// Run the clean up every 30 minutes
	ticker := time.NewTicker(cleanUpTime)

	defer ticker.Stop()

	for range ticker.C {
		for lichessId, timeAtStart := range *links {
			if time.Since(timeAtStart) > minLinkStayInMap {
				sw.mu.Lock()
				delete(*links, lichessId)
				sw.mu.Unlock()

			}
		}
	}
}

// sends link to telegram Ids (users) that are actively using the bot
func (sw *SWbot) sendLinkToActiveTelegramIds(linkId string) {

	var ids []int64
	var errResponse req.ErrorResponse

	statusCode, err := req.GetRequest(fmt.Sprintf("%s/telegram/bot/users/active", config.Cfg.Url), &ids, &errResponse)
	if statusCode == http.StatusInternalServerError {
		slog.Error("failed to get telegram bot users", "err", errResponse.Error)

	} else if statusCode != http.StatusOK || err != nil {
		slog.Error("failed to get telegram bot users", "err", err, "statusCode", statusCode)
	}
	for _, id := range ids {
		msg := tgbotapi.NewMessage(id, fmt.Sprintf("%s%s", base_url, linkId))

		sw.Bot.Send(msg)
	}
}

// Sends a message to all active users when the bot is going for maintanance (doesn't send to author Hopertz)
func (sw *SWbot) NotifyUsersOfMaintenance(msg string) {

	var ids []int64
	var errResponse req.ErrorResponse

	statusCode, err := req.GetRequest(fmt.Sprintf("%s/telegram/bot/users/active", config.Cfg.Url), &ids, &errResponse)
	if statusCode == http.StatusInternalServerError {
		slog.Error("failed to get telegram bot users", "err", errResponse.Error)

	} else if statusCode != http.StatusOK || err != nil {
		slog.Error("failed to get telegram bot users", "err", err, "statusCode", statusCode)

	}
	for _, id := range ids {
		if id != Master_ID {
			msg := tgbotapi.NewMessage(id, msg)
			sw.Bot.Send(msg)
		}

	}
}
