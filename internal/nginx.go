package internal

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ollama/ollama/api"
)

func Nginx(bot *tgbotapi.BotAPI, client *api.Client, chatID int64, ctx context.Context, history map[int64][]api.Message) {
	out, _ := exec.Command("systemctl", "is-active", "nginx").Output()
	status := strings.TrimSpace(string(out))

	prompt := fmt.Sprintf("Сервис Nginx сейчас в состоянии: %s. Дай очень краткий комментарий сисадмина.", status)
	var explain string
	req := &api.GenerateRequest{Model: "qwen2.5-coder:3b", Prompt: prompt}
	_ = client.Generate(ctx, req, func(resp api.GenerateResponse) error {
		explain += resp.Response
		return nil
	})

	msgText := fmt.Sprintf("🌐 *Статус Nginx:* `%s` \n\n%s", status, explain)
	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)

	history[chatID] = append(history[chatID], api.Message{Role: "user", Content: "Какой статус у Nginx?"})
	history[chatID] = append(history[chatID], api.Message{Role: "assistant", Content: explain})
}
