package commands

import (
	"fmt"

	"github.com/gsmcwhirter/go-util/v5/deferutil"
	"github.com/gsmcwhirter/go-util/v5/errors"
	"github.com/gsmcwhirter/go-util/v5/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v12/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v12/logging"
)

func (c *adminCommands) delete(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.delete", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "delete", "args", msg.Contents())

	gsettings, err := storage.GetSettings(msg.Context(), c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		level.Info(logger).Message("command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return nil, msghandler.ErrUnauthorized
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 1 {
		return r, errors.New("need event name")
	}

	if len(msg.Contents()) > 1 {
		return r, errors.New("too many arguments")
	}

	trialName := msg.Contents()[0]

	t, err := c.deps.TrialAPI().NewTransaction(msg.Context(), msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(msg.Context()) })

	if err = t.DeleteTrial(msg.Context(), trialName); err != nil {
		return r, errors.Wrap(err, "could not delete event")
	}

	if err = t.Commit(msg.Context()); err != nil {
		return r, errors.Wrap(err, "could not delete event")
	}

	level.Info(logger).Message("trial deleted", "trial_name", trialName)
	r.Description = fmt.Sprintf("Deleted event %q", trialName)

	return r, nil
}
