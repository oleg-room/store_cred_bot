package main

import (
	"errors"
	"fmt"
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"tg_bot/db"
	"tg_bot/models"
	"time"
)

type CredStore struct {
	Bot     *tg.BotAPI
	Couch   *db.Couch
	Updates tg.UpdatesChannel
}

var keyboardButtons = tg.NewReplyKeyboard(
	tg.NewKeyboardButtonRow(
		tg.NewKeyboardButton("override current creds"),
		tg.NewKeyboardButton("decline operation"),
	),
)

var ValidCommands = []string{"get", "set", "del", "start", "help"}

// ValidateUpdate make some pre-handling errors check. In case bot expect the command, but it isn't...
func (c *CredStore) ValidateUpdate(update tg.Update) error {
	if update.CallbackQuery != nil {
		callback := tg.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
		if _, err := c.Bot.AnswerCallbackQuery(callback); err != nil {
			panic(err)
		}
		msg := tg.NewMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Data)
		if _, err := c.Bot.Send(msg); err != nil {
			panic(err)
		}
	}
	if update.Message == nil {
		return models.ErrEmptyMsg
	}

	if !update.Message.IsCommand() {
		return models.ErrMsgNotACommand
	}

	if !lo.Contains[string](ValidCommands, update.Message.Command()) {
		return fmt.Errorf("bad command %s: %w", update.Message.Text, models.ErrNoRecognizedCommand)
	}
	return nil
}

func (c *CredStore) HandleSetCommand(chatID int64) error {
	if _, err := c.Bot.Send(tg.NewMessage(chatID, "enter the service name")); err != nil {
		logrus.WithError(err).Error()
		return err
	}
	i := 0
	service := models.Service{}

	var user *tg.User
FinishPolling:
	for u := range c.Updates {
		if u.Message.IsCommand() {
			_, _ = c.Bot.Send(tg.NewMessage(chatID, "your message is command, not a service name. Aborting process..."))
			return nil
		}
		switch i {
		case 0:
			service.Name = u.Message.Text
			user = u.Message.From

			// check if such service already exists in database under the user
			serviceToUpdate, _ := c.Couch.GetService(service.Name, user.UserName)
			if serviceToUpdate != nil {
				response := fmt.Sprintf("creds under service %s already exists. login: %s pass: ******. Select what bot should do", serviceToUpdate.Name, serviceToUpdate.Login)
				msg := tg.NewMessage(u.Message.Chat.ID, response)
				msg.ReplyMarkup = keyboardButtons
				_, _ = c.Bot.Send(msg)

				upd := <-c.Updates
				switch upd.Message.Text {
				case "override current creds":
					msg.Text = "continuing setting creds"
					msg.ReplyMarkup = tg.NewRemoveKeyboard(true)
					_, _ = c.Bot.Send(msg)
				case "decline operation":
					msg.Text = "aborting..."
					msg.ReplyMarkup = tg.NewRemoveKeyboard(true)
					_, _ = c.Bot.Send(msg)
					return nil
				}
			}
			if _, err := c.Bot.Send(tg.NewMessage(chatID, "enter login")); err != nil {
				logrus.WithError(err).Error()
				return err
			}
		case 1:
			service.Login = u.Message.Text
			if _, err := c.Bot.Send(tg.NewMessage(chatID, "enter password")); err != nil {
				logrus.WithError(err).Error()
				return err
			}
		case 2:
			service.Password = u.Message.Text
			break FinishPolling
		}
		i++
	}
	err := c.Couch.SaveServiceCreds(service, user.UserName)
	if err != nil {
		_, _ = c.Bot.Send(tg.NewMessage(chatID, "creds not saved"))
		return fmt.Errorf("creds not saved. %w", err)
	}
	_, _ = c.Bot.Send(tg.NewMessage(chatID, "creds successfully saved"))
	return nil
}

func (c *CredStore) HandleGetCommand(chatID int64) error {
	if _, err := c.Bot.Send(tg.NewMessage(chatID, "enter the service name")); err != nil {
		logrus.WithError(err).Error()
		return err
	}
	u := <-c.Updates
	service, err := c.Couch.GetService(u.Message.Text, u.Message.From.UserName)
	if err != nil {
		_, _ = c.Bot.Send(tg.NewMessage(chatID, "creds not retrieved"))
		return fmt.Errorf("cannot handle get command. %w", err)
	}
	if service == nil {
		response := fmt.Sprintf("creds under service %s not found", u.Message.Text)
		msg := tg.NewMessage(u.Message.Chat.ID, response)
		_, _ = c.Bot.Send(msg)
		return nil
	}
	msg, _ := c.Bot.Send(tg.NewMessage(chatID, service.String()))
	// deleting msg with creds after a while
	delMsgConf := tg.NewDeleteMessage(u.Message.Chat.ID, msg.MessageID)
	go func() {
		time.Sleep(time.Second * 30)
		_, _ = c.Bot.DeleteMessage(delMsgConf)
	}()
	return nil
}

func (c *CredStore) HandleDeleteCommand(chatID int64) error {
	if _, err := c.Bot.Send(tg.NewMessage(chatID, "enter the service name")); err != nil {
		logrus.WithError(err).Error()
		return err
	}
	u := <-c.Updates
	// if there is not such service in database, then service not deleted (false)
	service, err := c.Couch.GetService(u.Message.Text, u.Message.From.UserName)
	if err != nil {
		return err
	}
	if service == nil {
		response := fmt.Sprintf("No creds for %s service, so nothing to delete", u.Message.Text)
		_, _ = c.Bot.Send(tg.NewMessage(chatID, response))
		return nil
	}
	ok, err := c.Couch.DeleteService(u.Message.Text, u.Message.From.UserName)
	if errors.Is(err, models.ErrServiceNotExistsInDB) {
		_, _ = c.Bot.Send(tg.NewMessage(chatID, "no such service in database, so cannot delete"))
	} else if err != nil {
		_, _ = c.Bot.Send(tg.NewMessage(chatID, "creds not deleted due to unknown reason"))
		return err
	}
	if ok {
		response := fmt.Sprintf("creds for %s deleted", u.Message.Text)
		_, _ = c.Bot.Send(tg.NewMessage(chatID, response))
	}
	return nil
}

// MakeUpdatesChan configure updates settings
func (c *CredStore) MakeUpdatesChan() {
	updConfig := tg.NewUpdate(0)
	updConfig.Timeout = 30
	updates, err := c.Bot.GetUpdatesChan(updConfig)
	if err != nil {
		logrus.WithError(err).Fatal("cannot get updates from tg bot")
	}
	c.Updates = updates
}
