package lichess

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const team_members_url = "https://lichess.org/api/team/nyumbani-mates/users"

type Member struct {
	ID       string `json:"id"`
	Username string `json:"name"`
}

// struct with  json tags as  db columns
type MemberDB struct {
	LichessID string `json:"lichess_id"`
	Username  string `json:"username"`
}

// fetches all team membere of nyumbani mates
func FetchTeamMembers() []MemberDB {

	var dt []MemberDB
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", team_members_url, nil)

	if err != nil {
		slog.Error("FetchTeamMembers: Failed to create request", "error", err)
		return dt
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("LICHESS_TOKEN")))

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("FetchTeamMembers: Failed to fetch team members", "error", err)
		return dt
	}

	defer resp.Body.Close()

	results := json.NewDecoder(resp.Body)

	for {

		var member Member

		err := results.Decode(&member)

		if err != nil {
			if err != io.EOF {
				slog.Error("FetchTeamMembers: We got an error while reading", "error", err)
			}

			break
		}

		dt = append(dt, MemberDB{
			LichessID: member.ID,
			Username:  member.Username,
		})

	}

	// add by force as team member (refused to join) username & Id /;
	dt = append(dt, MemberDB{
		LichessID: "herald18",
		Username:  "herald18",
	})

	return dt

}
