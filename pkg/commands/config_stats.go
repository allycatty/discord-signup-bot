package commands

import (
	"fmt"

	"github.com/gsmcwhirter/go-util/v3/deferutil"
	"github.com/gsmcwhirter/go-util/v3/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v7/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v7/logging"
)

func (c *configCommands) collectStats(gid string) (stat, error) {
	s := stat{}

	t, err := c.deps.TrialAPI().NewTransaction(gid, false)
	if err != nil {
		return s, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	trials := t.GetTrials()

	for _, trial := range trials {
		s.trials++
		if trial.GetState() == storage.TrialStateClosed {
			s.closed++
		} else {
			s.open++
		}
	}

	return s, nil
}

func (c *configCommands) stats(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling configCommand", "command", "stats")

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	allGuilds, err := c.deps.GuildAPI().AllGuilds()
	if err != nil {
		return r, err
	}

	s := stat{}

	for _, guild := range allGuilds {
		stat, err := c.collectStats(guild)
		if err != nil {
			return r, err
		}

		s.trials += stat.trials
		s.open += stat.open
		s.closed += stat.closed
	}

	r.Description = fmt.Sprintf("Total guilds: %d\nTotal events: %d\nCurrently open: %d\nCurrently closed: %d\n", len(allGuilds), s.trials, s.open, s.closed)
	return r, nil
}