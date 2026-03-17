package main

import (
	"context"
	"diplombot/internal"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ollama/ollama/api"
)

var id = os.Getenv("ID-NV")

func main() {
	botToken := os.Getenv("TOKEN")
	bot, err := tgbotapi.NewBotAPI(botToken)
	strID := os.Getenv("TELEGRAM_USER_ID")

	// 2. Конвертируем строку в int64 (стандарт для Telegram ID)
	id, err := strconv.ParseInt(strID, 10, 64)
	if err != nil {
		log.Fatal("Ошибка: ID пользователя в .env должен быть числом")
	}
	if err != nil {
		log.Panic(err)
	}

	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Panic(err)
	}

	history := make(map[int64][]api.Message)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updates := bot.GetUpdatesChan(updateConfig)

	log.Printf("Бот запущен")
	go startMonitoring(bot, client, id)
	for update := range updates {
		if update.Message == nil {
			continue
		}

		ctx := context.Background()
		chatID := update.Message.Chat.ID
		// Нужно будет на свитч кейс сменить наверное?
		if update.Message.From.ID != id {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "⛔ Доступ запрещен. Я слушаюсь только хозяина.")
			bot.Send(msg)
			continue
		}
		if update.Message.IsCommand() && update.Message.Command() == "clear" {
			internal.Clear(bot, chatID, history)
			continue
		}
		if update.Message.IsCommand() && update.Message.Command() == "docker" {
			internal.Docker(bot, client, chatID, ctx, history)
			continue
		}
		if update.Message.IsCommand() && update.Message.Command() == "systemd" {
			internal.Systemd(bot, client, chatID, ctx, history)
			continue
		}
		if update.Message.IsCommand() && update.Message.Command() == "nginx" {
			internal.Nginx(bot, client, chatID, ctx, history)
			continue
		}
		if update.Message.IsCommand() && update.Message.Command() == "top" {
			internal.Top(bot, client, chatID, ctx, history)
			continue
		}
		if update.Message.IsCommand() && update.Message.Command() == "net" {
			internal.Net(bot, client, chatID, ctx, history)
			continue
		}
		if update.Message.IsCommand() && update.Message.Command() == "ports" {
			// ss -tulpn (TCP/UDP, Listening, Processes, Numbers)
			// берем только TCP: ss -ltn
			internal.Ports(bot, client, chatID, ctx, history)
			continue
		}
		if update.Message.IsCommand() && update.Message.Command() == "status" {
			internal.Status(bot, client, chatID, ctx, history)
			continue
		}
		if update.Message.IsCommand() && update.Message.Command() == "logs" {
			serviceName := update.Message.CommandArguments()
			if serviceName == "" {
				bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Укажите сервис, например: `/logs nginx`"))
				continue
			}
			out, err := exec.Command("journalctl", "-u", serviceName, "-n", "15", "--no-pager").Output()

			logData := string(out)
			if err != nil || len(strings.TrimSpace(logData)) < 10 {
				logData = fmt.Sprintf("Логи для сервиса `%s` не найдены или пусты.", serviceName)
			}

			prompt := fmt.Sprintf("Ты опытный сисадмин. Проанализируй последние строки логов сервиса %s и скажи, есть ли там критические ошибки или всё в порядке:\n%s", serviceName, logData)

			var analysis string
			req := &api.GenerateRequest{Model: "qwen2.5-coder:3b", Prompt: prompt}
			_ = client.Generate(ctx, req, func(resp api.GenerateResponse) error {
				analysis += resp.Response
				return nil
			})

			response := fmt.Sprintf("📜 *Последние логи %s:*\n```\n%s\n```\n\n🧠 *Анализ ИИ:*\n%s", serviceName, logData, analysis)
			history[chatID] = append(history[chatID], api.Message{Role: "user", Content: "Проверь логи"})
			history[chatID] = append(history[chatID], api.Message{Role: "assistant", Content: analysis})
			msg := tgbotapi.NewMessage(chatID, response)
			msg.ParseMode = "Markdown"
			bot.Send(msg)
			continue
		}

		if update.Message.IsCommand() && update.Message.Command() == "restart" {
			serviceName := update.Message.CommandArguments()
			if serviceName == "" {
				bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Укажите имя сервиса, например: `/restart nginx`"))
				continue
			}

			cmd := exec.Command("sudo", "systemctl", "restart", serviceName)
			err := cmd.Run()

			statusMsg := ""
			if err != nil {
				statusMsg = fmt.Sprintf("❌ Ошибка при перезапуске `%s`: %v", serviceName, err)
			} else {
				statusMsg = fmt.Sprintf("🔄 Сервис `%s` успешно перезапущен!", serviceName)
			}

			var aiComment string
			prompt := fmt.Sprintf("Я только что перезапустил сервис %s. Напиши короткое подтверждение как суровый сисадмин.", serviceName)

			_ = client.Generate(ctx, &api.GenerateRequest{Model: "qwen2.5-coder:3b", Prompt: prompt}, func(resp api.GenerateResponse) error {
				aiComment += resp.Response
				return nil
			})

			msg := tgbotapi.NewMessage(chatID, statusMsg+"\n\n"+aiComment)
			msg.ParseMode = "Markdown"
			bot.Send(msg)
			continue
		}

		// ОБЫЧНОЕ ОБЩЕНИЕ (С КОНТЕКСТОМ)
		bot.Send(tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping))
		history[chatID] = append(history[chatID], api.Message{Role: "user", Content: update.Message.Text})

		chatReq := &api.ChatRequest{
			Model:    "qwen2.5-coder:3b",
			Messages: history[chatID],
		}

		var fullResponse string
		err = client.Chat(ctx, chatReq, func(resp api.ChatResponse) error {
			fullResponse += resp.Message.Content
			return nil
		})

		if err != nil || fullResponse == "" {
			fullResponse = "⚠️ Ошибка связи с нейросетью."
		} else {

			history[chatID] = append(history[chatID], api.Message{Role: "assistant", Content: fullResponse})

			if len(history[chatID]) > 10 {
				history[chatID] = history[chatID][len(history[chatID])-10:]
			}
		}

		bot.Send(tgbotapi.NewMessage(chatID, fullResponse))
	}
}
