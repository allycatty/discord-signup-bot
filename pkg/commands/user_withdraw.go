package commands

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v3/deferutil"
	"github.com/gsmcwhirter/go-util/v3/errors"
	"github.com/gsmcwhirter/go-util/v3/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v7/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v7/logging"
)

func (c *userCommands) withdraw(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling rootCommand", "command", "withdraw", "trial_name", msg.Contents())

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 1 {
		return r, errors.New("missing event name")
	}

	trialName := strings.TrimSpace(msg.Contents()[0])

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	if !isSignupChannel(logger, msg, trial.GetSignupChannel(), gsettings.AdminChannel, gsettings.AdminRole, c.deps.BotSession()) {
		level.Info(logger).Message("command not in signup channel", "signup_channel", trial.GetSignupChannel())
		return r, msghandler.ErrNoResponse
	}

	if trial.GetState() != storage.TrialStateOpen {
		return r, errors.New("cannot withdraw from a closed trial")
	}

	trial.RemoveSignup(cmdhandler.UserMentionString(msg.UserID()))

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not save trial withdraw")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not save trial withdraw")
	}

	level.Info(logger).Message("withdrew", "trial_name", trialName)
	descStr := fmt.Sprintf("Withdrew from %s", trialName)

	if gsettings.ShowAfterWithdraw == "true" {
		level.Debug(logger).Message("auto-show after withdraw", "trial_name", trialName)

		r2 := formatTrialDisplay(trial, true)
		r2.To = cmdhandler.UserMentionString(msg.UserID())
		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		return r2, nil
	}

	r.Description = descStr

	return r, nil
}