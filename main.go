package main

import (
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
	"os"
	"tg_bot/db"
)

const (
	couchDBName = "users_creds"
	databaseURL = "http://127.0.0.1:5984"
)

func main() {
	//connection to couchDB
	couch := &db.Couch{}
	err := couch.InitConnection(databaseURL, os.Getenv("COUCHDB_USER"), os.Getenv("COUCHDB_PASSWORD"))
	if err != nil {
		logrus.WithError(err).Fatalf("cannot init connection with couch DMS on %s", databaseURL)
	}
	err = couch.CreateDatabase(couchDBName)
	if err != nil {
		logrus.WithError(err).Fatalf("cannot create %s database in couch DMS on %s", couchDBName, databaseURL)
	}

	bot, err := tg.NewBotAPI(os.Getenv("TG_BOT_TOKEN"))
	if err != nil {
		logrus.WithError(err).Fatal("cannot init bot")
	}

	credStore := CredStore{
		Couch: couch,
		Bot:   bot,
	}

	credStore.MakeUpdatesChan()
	for upd := range credStore.Updates {
		chatID := upd.Message.Chat.ID

		if err = credStore.ValidateUpdate(upd); err != nil {
			msg := tg.NewMessage(chatID, err.Error())
			_, _ = bot.Send(msg)
			continue
		}

		switch upd.Message.Command() {
		case "get":
			err = credStore.HandleGetCommand(chatID)
			if err != nil {
				logrus.WithError(err).Error("service not gotten")
			}
		case "set":
			err = credStore.HandleSetCommand(chatID)
			if err != nil {
				logrus.WithError(err).Error("service not set")
			}
		case "del":
			err = credStore.HandleDeleteCommand(chatID)
			if err != nil {
				logrus.WithError(err).Error("service not deleted")
			}
		}
	}
}
