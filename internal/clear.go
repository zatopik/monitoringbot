package internal

import (
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ollama/ollama/api"
)

func Clear(bot *tgbotapi.BotAPI, chatID int64, history map[int64][]api.Message) {
	delete(history, chatID)
	bot.Send(tgbotapi.NewMessage(chatID, "🧹 Контекст беседы очищен!"))
}
