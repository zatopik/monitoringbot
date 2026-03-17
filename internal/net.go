package internal

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ollama/ollama/api"
)

func Net(bot *tgbotapi.BotAPI, client *api.Client, chatID int64, ctx context.Context, history map[int64][]api.Message) {
	// Получаем статистику интерфейсов (RX/TX байты)
	out, _ := exec.Command("ip", "-s", "link").Output()

	prompt := fmt.Sprintf("Проанализируй сетевой трафик. Если видишь аномально высокие значения RX/TX, предупреди о возможной сетевой атаке или перегрузке:\n%s", string(out))

	var analysis string
	req := &api.GenerateRequest{Model: "qwen2.5-coder:3b", Prompt: prompt}
	_ = client.Generate(ctx, req, func(resp api.GenerateResponse) error {
		analysis += resp.Response
		return nil
	})

	history[chatID] = append(history[chatID], api.Message{Role: "assistant", Content: "Сетевой анализ: " + analysis})
	msg := tgbotapi.NewMessage(chatID, "🌐 *Анализ сети:* \n\n"+analysis)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}
