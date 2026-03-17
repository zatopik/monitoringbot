package internal

import (
	"context"
	"fmt"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ollama/ollama/api"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

func Status(bot *tgbotapi.BotAPI, client *api.Client, chatID int64, ctx context.Context, history map[int64][]api.Message) {
	v, _ := mem.VirtualMemory()
	c, _ := cpu.Percent(0, false)
	var cpuVal float64
	if len(c) > 0 {
		cpuVal = c[0]
	}

	systemStats := fmt.Sprintf("CPU: %.1f%%, RAM: %.1f%% (Used: %v MB)", cpuVal, v.UsedPercent, v.Used/1024/1024)
	prompt := "Ты сисадмин. Коротко проанализируй: " + systemStats

	var summary string
	req := &api.GenerateRequest{Model: "qwen2.5-coder:3b", Prompt: prompt}
	_ = client.Generate(ctx, req, func(resp api.GenerateResponse) error {
		summary += resp.Response
		return nil
	})

	msg := tgbotapi.NewMessage(chatID, "📊 *Анализ системы:*\n"+summary)
	msg.ParseMode = "Markdown"
	bot.Send(msg)

	history[chatID] = append(history[chatID], api.Message{Role: "user", Content: "Команда /status: покажи метрики"})
	history[chatID] = append(history[chatID], api.Message{Role: "assistant", Content: summary})

}
