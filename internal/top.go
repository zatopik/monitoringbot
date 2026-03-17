package internal

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ollama/ollama/api"
)

func Top(bot *tgbotapi.BotAPI, client *api.Client, chatID int64, ctx context.Context, history map[int64][]api.Message) {
	// Берем первые 15 строк из top в режиме одного снимка (-b -n 1)
	out, _ := exec.Command("top", "-b", "-n", "1").Output()
	// Обрезаем вывод, чтобы не перегружать контекст нейронки
	topData := string(out)
	if len(topData) > 1500 {
		topData = topData[:1500]
	}

	prompt := fmt.Sprintf("Вот вывод команды top. Найди процессы, которые потребляют больше всего ресурсов, и скажи, не они ли причина тормозов:\n%s", topData)

	var analysis string
	req := &api.GenerateRequest{Model: "qwen2.5-coder:3b", Prompt: prompt}
	_ = client.Generate(ctx, req, func(resp api.GenerateResponse) error {
		analysis += resp.Response
		return nil
	})

	history[chatID] = append(history[chatID], api.Message{Role: "assistant", Content: "Анализ процессов: " + analysis})
	msg := tgbotapi.NewMessage(chatID, "🔝 *Топ процессов:* \n\n"+analysis)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}
