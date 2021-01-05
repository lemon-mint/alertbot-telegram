package main

import (
	"fmt"
	"os"

	tgbot "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/lemon-mint/godotenv"
)

func main() {
	godotenv.Load()
	bot, err := tgbot.NewBotAPI(os.Getenv("TGBOTKEY"))
	if err != nil {
		panic(err)
	}
	fmt.Println(bot)
}
