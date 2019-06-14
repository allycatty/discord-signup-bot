package commands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gsmcwhirter/discord-bot-lib/v8/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v8/etfapi"
	"github.com/gsmcwhirter/go-util/v3/errors"
	log "github.com/gsmcwhirter/go-util/v3/logging"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

var ErrUnknownRole = errors.New("unknown role")

var isAdminAuthorized = msghandler.IsAdminAuthorized
var isAdminChannel = msghandler.IsAdminChannel

func isSignupChannel(logger log.Logger, msg cmdhandler.Message, signupChannel, adminChannel, adminRole string, session *etfapi.Session) bool {
	if msghandler.IsSignupChannel(msg, signupChannel, session) {
		return true
	}

	if !isAdminChannel(logger, msg, adminChannel, session) {
		return false
	}

	return isAdminAuthorized(logger, msg, adminRole, session)
}

func signupsForRole(role string, signups []storage.TrialSignup, sorted bool) []string {
	roleLower := strings.ToLower(role)
	t := make([]string, 0, len(signups))
	for _, s := range signups {
		if strings.ToLower(s.GetRole()) == roleLower {
			t = append(t, s.GetName())
		}
	}

	if sorted {
		sort.Strings(t)
	}

	return t
}

func roleCountByName(role string, roleCounts []storage.RoleCount) (storage.RoleCount, bool) {
	roleLower := strings.ToLower(role)
	for _, rc := range roleCounts {
		if strings.ToLower(rc.GetRole()) == roleLower {
			return rc, true
		}
	}

	return nil, false
}

func parseSettingDescriptionArgs(args []string) (map[string]string, error) {
	argMap := map[string]string{}

	for _, pair := range args {
		if pair == "" {
			continue
		}

		pairParts := strings.SplitN(pair, "=", 2)
		if len(pairParts) < 2 {
			return argMap, errors.New("could not parse arguments")
		}

		// if strings.ToLower(pairParts[0]) == "description" {
		// 	argMap[strings.ToLower(pairParts[0])] = fmt.Sprintf("%s %s", pairParts[1], strings.Join(parts[i+1:], " "))
		// 	break
		// }

		argMap[strings.ToLower(pairParts[0])] = pairParts[1]
	}

	return argMap, nil
}

type roleCtEmo struct {
	role string
	ct   uint64
	emo  string
}

func parseRolesString(args string) ([]roleCtEmo, error) {
	roles := strings.Split(strings.TrimSpace(args), ",")
	roleEmoCt := make([]roleCtEmo, 0, len(roles))

	for _, roleStr := range roles {
		if roleStr == "" {
			continue
		}

		roleParts := strings.SplitN(roleStr, ":", 3)
		if len(roleParts) < 2 {
			return roleEmoCt, errors.New("could not parse roles")
		}

		roleCt, err := strconv.Atoi(roleParts[1])
		if err != nil {
			return roleEmoCt, err
		}

		var emo string
		if len(roleParts) == 3 {
			emo = roleParts[2]
		}

		roleEmoCt = append(roleEmoCt, roleCtEmo{
			role: roleParts[0],
			ct:   uint64(roleCt),
			emo:  emo,
		})
	}

	return roleEmoCt, nil
}

func getTrialRoleSignups(signups []storage.TrialSignup, rc storage.RoleCount) ([]string, []string) {
	lowerRole := strings.ToLower(rc.GetRole())
	suNames := make([]string, 0, len(signups))
	ofNames := make([]string, 0, len(signups))
	for _, su := range signups {
		if strings.ToLower(su.GetRole()) != lowerRole {
			continue
		}

		if uint64(len(suNames)) < rc.GetCount() {
			suNames = append(suNames, su.GetName())
		} else {
			ofNames = append(ofNames, su.GetName())
		}
	}

	return suNames, ofNames
}

func formatTrialDisplay(trial storage.Trial, withState bool) *cmdhandler.EmbedResponse {
	r := &cmdhandler.EmbedResponse{}

	if withState {
		r.Title = fmt.Sprintf("__%s__ (%s)", trial.GetName(), string(trial.GetState()))
	} else {
		r.Title = fmt.Sprintf("__%s__", trial.GetName())
	}
	r.Description = trial.GetDescription()
	r.Fields = []cmdhandler.EmbedField{}

	overflowFields := []cmdhandler.EmbedField{}

	roleCounts := trial.GetRoleCounts() // already sorted by name
	signups := trial.GetSignups()

	for _, rc := range roleCounts {
		suNames, ofNames := getTrialRoleSignups(signups, rc)

		if len(suNames) > 0 {
			r.Fields = append(r.Fields, cmdhandler.EmbedField{
				Name: fmt.Sprintf("*%s* (%d/%d)", rc.GetRole(), len(suNames), rc.GetCount()),
				Val:  rc.GetEmoji() + strings.Join(suNames, fmt.Sprintf("\n%s", rc.GetEmoji())) + "\n_ _\n",
			})
		} else {
			r.Fields = append(r.Fields, cmdhandler.EmbedField{
				Name: fmt.Sprintf("*%s* (%d/%d)", rc.GetRole(), len(suNames), rc.GetCount()),
				Val:  "(empty)\n_ _\n",
			})
		}

		if len(ofNames) > 0 {
			overflowFields = append(overflowFields, cmdhandler.EmbedField{
				Name: fmt.Sprintf("*Overflow %s* (%d)", rc.GetRole(), len(ofNames)),
				Val:  rc.GetEmoji() + strings.Join(ofNames, fmt.Sprintf("\n%s", rc.GetEmoji())) + "\n_ _\n",
			})
		}
	}

	r.Fields = append(r.Fields, overflowFields...)

	return r
}

func signupUser(trial storage.Trial, userMentionStr, role string) (bool, error) {
	roleCounts := trial.GetRoleCounts() // already sorted by name
	rc, known := roleCountByName(role, roleCounts)
	if !known {
		return false, ErrUnknownRole
	}

	trial.AddSignup(userMentionStr, role)

	signups := trial.GetSignups()
	roleSignups := signupsForRole(role, signups, false)

	overflow := uint64(len(roleSignups)) > rc.GetCount()

	return overflow, nil
}
