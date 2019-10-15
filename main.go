package main

import (
	"fmt"

	"github.com/DrDelphi/BonBot2/helpers"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func main() {
	helpers.LoadNodesJson()

	err := helpers.DB.Init()
	if err != nil {
		fmt.Println("Unable to connect to database")
		panic("Unable to connect to database")
	}
	defer helpers.DB.Close()

	helpers.TgBot, _ = tgbotapi.NewBotAPI(helpers.TgBotToken)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, _ := helpers.TgBot.GetUpdatesChan(u)

	go helpers.GetHeartBeat()

	for update := range updates {
		if update.Message != nil {
			if update.Message.IsCommand() {
				helpers.CommandReceived(update)
			}
			if update.Message.ReplyToMessage != nil {
				helpers.ReplyReceived(update.Message)
			}
		}
		if update.CallbackQuery != nil {
			helpers.CallbackQueryReceived(update.CallbackQuery)
		}
	}
}
