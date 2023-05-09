package main

import (
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

var ValidCommands = []string{"get", "set", "del"}

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
FinishPolling:
	for u := range c.Updates {
		if u.Message.IsCommand() {
			_, _ = c.Bot.Send(tg.NewMessage(chatID, "your message is command, not a service name. Aborting process..."))
			return nil
		}
		switch i {
		case 0:
			service.Name = u.Message.Text

			//checking if such service already exists in database
			serviceToUpdate, _ := c.Couch.GetService(service.Name)
			if serviceToUpdate != nil {
				response := fmt.Sprintf("creds under service %s already exists. To update creds you should first do '/del' command, then '/set'", serviceToUpdate.Name)

				_, _ = c.Bot.Send(tg.NewMessage(chatID, response))
				return nil
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
	// saving data to database
	err := c.Couch.SaveServiceCreds(service)
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
	for u := range c.Updates {
		service, err := c.Couch.GetService(u.Message.Text)
		if err != nil {
			return fmt.Errorf("cannot handle get command. %w", err)
		}
		msg, _ := c.Bot.Send(tg.NewMessage(chatID, service.String()))

		// deleting msg with creds after a while
		delMsgConf := tg.NewDeleteMessage(u.Message.Chat.ID, msg.MessageID)
		go func() {
			time.Sleep(time.Second * 30)
			_, _ = c.Bot.DeleteMessage(delMsgConf)
		}()
	}
	return nil
}

func (c *CredStore) HandleDeleteCommand(chatID int64) error {
	if _, err := c.Bot.Send(tg.NewMessage(chatID, "enter the service name")); err != nil {
		logrus.WithError(err).Error()
		return err
	}
	var serviceName string
	for u := range c.Updates {
		serviceName = u.Message.Text
		// if there is not such service in database, then service not deleted (false)
		ok, err := c.Couch.DeleteService(serviceName)
		if err != nil {
			return fmt.Errorf("cannot handle delete command %w", err)
		}
		if !ok {
			_, _ = c.Bot.Send(tg.NewMessage(chatID, "creds for this service not exist in database, so nothing to delete"))
			return nil
		}
	}
	response := fmt.Sprintf("creds for %s deleted", serviceName)
	_, _ = c.Bot.Send(tg.NewMessage(chatID, response))
	return nil
}

func (c *CredStore) MakeUpdatesChan() {
	updConfig := tg.NewUpdate(0)
	updConfig.Timeout = 30
	updates, err := c.Bot.GetUpdatesChan(updConfig)
	if err != nil {
		logrus.WithError(err).Fatal("cannot get updates from tg bot")
	}
	c.Updates = updates
}
