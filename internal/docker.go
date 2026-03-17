package internal

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ollama/ollama/api"
)

func Docker(bot *tgbotapi.BotAPI, client *api.Client, chatID int64, ctx context.Context, history map[int64][]api.Message) {

	cmd := exec.Command("docker", "ps", "--format", "table {{.Names}}\t{{.Status}}\t{{.Image}}")
	out, err := cmd.Output()
	dockerData := string(out)
	if err != nil {
		dockerData = "Ошибка: Docker не запущен или нет прав доступа (попробуйте sudo usermod -aG docker $USER)."
	} else if len(strings.TrimSpace(dockerData)) <= 20 { // Если только заголовок таблицы
		dockerData = "Docker запущен, но активных контейнеров нет."
	}

	prompt := fmt.Sprintf("ИНСТРУКЦИЯ: Ты сисадмин. Проанализируй этот список контейнеров и дай отчет. Если список пуст, так и скажи. СПИСОК:\n%s", dockerData)

	var analysis string
	req := &api.GenerateRequest{Model: "qwen2.5-coder:3b", Prompt: prompt}
	_ = client.Generate(ctx, req, func(resp api.GenerateResponse) error {
		analysis += resp.Response
		return nil
	})

	history[chatID] = append(history[chatID], api.Message{Role: "user", Content: "Проверь докер"})
	history[chatID] = append(history[chatID], api.Message{Role: "assistant", Content: analysis})

	msg := tgbotapi.NewMessage(chatID, "🐳 *Статус Docker:* \n\n"+analysis)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}
