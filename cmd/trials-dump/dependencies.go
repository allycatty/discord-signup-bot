package main

import (
	"os"
	"time"

	bolt "github.com/coreos/bbolt"
	"github.com/go-kit/kit/log"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

type dependencies struct {
	logger   log.Logger
	db       *bolt.DB
	trialAPI storage.TrialAPI
	guildAPI storage.GuildAPI
	// botSession *session.Session
}

func createDependencies(conf config) (d *dependencies, err error) {
	d = &dependencies{}
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "timestamp", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	d.logger = logger

	d.db, err = bolt.Open(conf.Database, 0660, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return
	}

	d.trialAPI, err = storage.NewBoltTrialAPI(d.db)
	if err != nil {
		return
	}

	d.guildAPI, err = storage.NewBoltGuildAPI(d.db)
	if err != nil {
		return
	}

	// d.botSession = session.NewSession()

	return
}

func (d *dependencies) Close() {
	if d.db != nil {
		d.db.Close() // nolint: errcheck
	}
}

func (d *dependencies) Logger() log.Logger {
	return d.logger
}

func (d *dependencies) TrialAPI() storage.TrialAPI {
	return d.trialAPI
}

func (d *dependencies) GuildAPI() storage.GuildAPI {
	return d.guildAPI
}

// func (d *dependencies) BotSession() *session.Session {
// 	return d.botSession
// }