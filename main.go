package main

import (
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
	"os"
	"tg_bot/db"
	"time"
)

const (
	couchDBName = "users_creds"
)

func main() {
	//connection to couchDB
	couch := &db.Couch{}
	databaseURL := os.Getenv("DB_URL")
	// hardcoded timeout for starting up couchdb. Not idiomatic
	time.Sleep(time.Second * 5)
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
		case "start":
			_, _ = credStore.Bot.Send(tg.NewMessage(upd.Message.Chat.ID, "Hello. It's bot for saving your credentials for different services"))
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
		case "help":
			_, _ = credStore.Bot.Send(tg.NewMessage(upd.Message.Chat.ID, "Type /get command to retrieve data. Type /set command to set new or update data. Type /del to delete data"))
		}
	}
}
