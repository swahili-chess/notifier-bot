package poll

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/swahili-chess/notifier-bot/config"
	lichess "github.com/swahili-chess/notifier-bot/internal/lichess"
	"github.com/swahili-chess/notifier-bot/internal/req"
)

type SWbot struct {
	Bot   *tgbotapi.BotAPI
	Links *map[string]time.Time
	mu    sync.RWMutex
}

// periodically fetches the list of team members and sends the result to a channel for processing.
// It also calls a function to add new members to the Lichess team member database if they are not already present.
func (sw *SWbot) PollAndUpdateTeamMembers(members chan<- []lichess.MemberDB) {

	ticker := time.NewTicker(time.Minute * 5)

	defer ticker.Stop()

	for range ticker.C {
		res := lichess.FetchTeamMembers()
		members <- res
		sw.AddNewLichessTeamMembers(res)

	}
}

// retrieves the current list of members, compares it with the provided old list of members,
// and inserts any new members who are not yet in the database.
func (sw *SWbot) AddNewLichessTeamMembers(allMembers []lichess.MemberDB) {

	var oldMembers []string
	var errResponse req.ErrorResponse

	statusCode, err := req.GetRequest(fmt.Sprintf("%s/lichess/members", config.Cfg.Url), &oldMembers, &errResponse)

	if statusCode == http.StatusOK && err == nil {
		newMembers := filterNewMembers(oldMembers, allMembers)
		for _, player := range newMembers {
			statusCode, err := req.PostOrPutRequest(http.MethodPost, fmt.Sprintf("%s/lichess/members", config.Cfg.Url), player, &errResponse)
			if statusCode == http.StatusInternalServerError {
				slog.Error("AddNewLichessTeamMembers: Failed to insert member", "error", errResponse.Error, "url", fmt.Sprintf("%s/lichess/members", config.Cfg.Url))
			} else if err != nil {
				slog.Error("AddNewLichessTeamMembers: Failed to insert member", "error", err, "url", fmt.Sprintf("%s/lichess/members", config.Cfg.Url))
			}
		}
	} else if statusCode == http.StatusInternalServerError {
		slog.Error("AddNewLichessTeamMembers: Failed to get members", "error", errResponse.Error, "url", fmt.Sprintf("%s/lichess/members", config.Cfg.Url))

	} else {
		slog.Error("AddNewLichessTeamMembers: Failed to get members", "error", err, "url", fmt.Sprintf("%s/lichess/members", config.Cfg.Url))
	}

}

// compares the list of old member IDs with a list of all new members ids and returns only the new members
func filterNewMembers(oldMembers []string, allMembers []lichess.MemberDB) []lichess.MemberDB {
	newMembers := []lichess.MemberDB{}
	oldMembersSet := make(map[string]bool)

	for _, member := range oldMembers {
		oldMembersSet[member] = true
	}

	for _, member := range allMembers {
		if _, found := oldMembersSet[member.LichessID]; !found {
			newMembers = append(newMembers, member)
		} else {
			delete(oldMembersSet, member.LichessID)
		}
	}

	return newMembers
}
