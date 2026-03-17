package internal

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ollama/ollama/api"
)

func Ports(bot *tgbotapi.BotAPI, client *api.Client, chatID int64, ctx context.Context, history map[int64][]api.Message) {
	out, err := exec.Command("ss", "-ltn").Output()
	portsList := string(out)
	if err != nil || portsList == "" {
		portsList = "Не удалось получить список портов."
	}

	// Просим нейронку проанализировать порты
	prompt := fmt.Sprintf("Вот список открытых TCP-портов на сервере:\n%s\nКакие из них стандартные, а на какие стоит обратить внимание?", portsList)

	var analysis string
	req := &api.GenerateRequest{Model: "qwen2.5-coder:3b", Prompt: prompt}
	_ = client.Generate(ctx, req, func(resp api.GenerateResponse) error {
		analysis += resp.Response
		return nil
	})

	// Сохраняем в историю, чтобы бот "помнил" порты в чате
	history[chatID] = append(history[chatID], api.Message{Role: "user", Content: "Какие порты открыты?"})
	history[chatID] = append(history[chatID], api.Message{Role: "assistant", Content: analysis})

	msg := tgbotapi.NewMessage(chatID, "🔌 *Открытые порты:* \n\n"+analysis)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}
