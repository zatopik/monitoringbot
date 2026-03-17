package internal

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ollama/ollama/api"
)

func Systemd(bot *tgbotapi.BotAPI, client *api.Client, chatID int64, ctx context.Context, history map[int64][]api.Message) {
	out, _ := exec.Command("systemctl", "list-units", "--state=failed", "--no-legend").Output()
	failedServices := string(out)

	if len(strings.TrimSpace(failedServices)) == 0 {
		failedServices = "Все системные сервисы работают штатно (failed units не найдено)."
	}

	// Добавь свои сервисы, за которыми следишь
	important := []string{"nginx", "ollama", "grafana", "prometheus"}
	var importantStatus string
	for _, s := range important {
		sOut, _ := exec.Command("systemctl", "is-active", s).Output()
		importantStatus += fmt.Sprintf("- %s: %s", s, string(sOut))
	}

	prompt := fmt.Sprintf("Проанализируй состояние сервисов. \nУпавшие:\n%s\n\nСтатус важных:\n%s",
		failedServices, importantStatus)

	var analysis string
	req := &api.GenerateRequest{Model: "qwen2.5-coder:3b", Prompt: prompt}
	_ = client.Generate(ctx, req, func(resp api.GenerateResponse) error {
		analysis += resp.Response
		return nil
	})

	history[chatID] = append(history[chatID], api.Message{Role: "assistant", Content: "Анализ systemd: " + analysis})

	msg := tgbotapi.NewMessage(chatID, "⚙️ *Статус Systemd:* \n\n"+analysis)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}
