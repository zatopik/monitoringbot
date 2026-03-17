package main

import (
	"context"
	"fmt"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ollama/ollama/api"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

func startMonitoring(bot *tgbotapi.BotAPI, client *api.Client, chatID int64) {
	var cpuVal float64
	for {
		time.Sleep(1 * time.Minute) // Проверка
		v, _ := mem.VirtualMemory()
		c, _ := cpu.Percent(0, false)
		if len(c) > 0 {
			cpuVal = c[0]
		}
		if v.UsedPercent > 90 {
			msg := fmt.Sprintf("🚨 *ВНИМАНИЕ:* Память заполнена на %.1f%%! (Использовано: %v MB)", v.UsedPercent, v.Used/1024/1024)
			bot.Send(tgbotapi.NewMessage(chatID, msg))
			prompt := "Система перегружена по памяти (90%+). Что мне проверить в первую очередь?"
			var advice string
			client.Generate(context.Background(), &api.GenerateRequest{Model: "qwen2.5-coder:3b", Prompt: prompt}, func(resp api.GenerateResponse) error {
				advice += resp.Response
				return nil
			})
			bot.Send(tgbotapi.NewMessage(chatID, "🧠 *Совет ИИ:* "+advice))
		}
		if cpuVal > 90 {
			msg := fmt.Sprintf("🚨 *ВНИМАНИЕ:* Процессор перегружен на %.1f%%!", cpuVal)
			bot.Send(tgbotapi.NewMessage(chatID, msg))
			prompt := "Система перегружена по процессору (90%+). Что мне проверить в первую очередь?"
			var advice string
			client.Generate(context.Background(), &api.GenerateRequest{Model: "qwen2.5-coder:3b", Prompt: prompt}, func(resp api.GenerateResponse) error {
				advice += resp.Response
				return nil
			})
			bot.Send(tgbotapi.NewMessage(chatID, "🧠 *Совет ИИ:* "+advice))
		}
	}
}
