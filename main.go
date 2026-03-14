package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ollama/ollama/api"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

func main() {
	// 1. Инициализация Telegram бота
	botToken := ""
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	// 2. Инициализация клиента Ollama
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Panic(err)
	}

	// Хранилище истории диалогов (ID чата -> список сообщений)
	history := make(map[int64][]api.Message)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updates := bot.GetUpdatesChan(updateConfig)

	log.Printf("Бот запущен. Ожидание сообщений...")

	for update := range updates {
		if update.Message == nil {
			continue
		}

		ctx := context.Background()
		chatID := update.Message.Chat.ID

		// --- КОМАНДА /clear (Очистка контекста) ---
		if update.Message.IsCommand() && update.Message.Command() == "clear" {
			delete(history, chatID)
			bot.Send(tgbotapi.NewMessage(chatID, "🧹 Контекст беседы очищен!"))
			continue
		}
		if update.Message.IsCommand() && update.Message.Command() == "docker" {
			// 1. Проверяем список запущенных контейнеров (ID, Имя, Статус, Образ)
			// Используем --format для чистого вывода, который легче понять нейронке
			cmd := exec.Command("docker", "ps", "--format", "table {{.Names}}\t{{.Status}}\t{{.Image}}")
			out, err := cmd.Output()

			dockerData := string(out)
			if err != nil {
				dockerData = "Ошибка: Docker не запущен или нет прав доступа (попробуйте sudo usermod -aG docker $USER)."
			} else if len(strings.TrimSpace(dockerData)) <= 20 { // Если только заголовок таблицы
				dockerData = "Docker запущен, но активных контейнеров нет."
			}

			// 2. Анализ нейронкой

			prompt := fmt.Sprintf("ИНСТРУКЦИЯ: Ты сисадмин. Проанализируй этот список контейнеров и дай отчет. Если список пуст, так и скажи. СПИСОК:\n%s", dockerData)

			var analysis string
			req := &api.GenerateRequest{Model: "qwen2.5-coder:3b", Prompt: prompt}
			_ = client.Generate(ctx, req, func(resp api.GenerateResponse) error {
				analysis += resp.Response
				return nil
			})

			// 3. Сохраняем в историю для контекста
			history[chatID] = append(history[chatID], api.Message{Role: "user", Content: "Проверь докер"})
			history[chatID] = append(history[chatID], api.Message{Role: "assistant", Content: analysis})

			msg := tgbotapi.NewMessage(chatID, "🐳 *Статус Docker:* \n\n"+analysis)
			msg.ParseMode = "Markdown"
			bot.Send(msg)
			continue
		}
		if update.Message.IsCommand() && update.Message.Command() == "systemd" {
			// 1. Получаем список всех НЕУДАЧНЫХ (failed) сервисов
			// Это самое важное для сисадмина
			out, _ := exec.Command("systemctl", "list-units", "--state=failed", "--no-legend").Output()
			failedServices := string(out)

			if len(strings.TrimSpace(failedServices)) == 0 {
				failedServices = "Все системные сервисы работают штатно (failed units не найдено)."
			}

			// 2. Проверяем конкретно важные для нас сервисы
			// Добавь сюда те, за которыми следишь
			important := []string{"nginx", "ollama", "grafana", "prometheus"}
			var importantStatus string
			for _, s := range important {
				sOut, _ := exec.Command("systemctl", "is-active", s).Output()
				importantStatus += fmt.Sprintf("- %s: %s", s, string(sOut))
			}

			// 3. Скормим всё нейронке для резюме
			prompt := fmt.Sprintf("Проанализируй состояние сервисов. \nУпавшие:\n%s\n\nСтатус важных:\n%s",
				failedServices, importantStatus)

			var analysis string
			req := &api.GenerateRequest{Model: "qwen2.5-coder:3b", Prompt: prompt}
			_ = client.Generate(ctx, req, func(resp api.GenerateResponse) error {
				analysis += resp.Response
				return nil
			})

			// Сохраняем в историю
			history[chatID] = append(history[chatID], api.Message{Role: "assistant", Content: "Анализ systemd: " + analysis})

			msg := tgbotapi.NewMessage(chatID, "⚙️ *Статус Systemd:* \n\n"+analysis)
			msg.ParseMode = "Markdown"
			bot.Send(msg)
			continue
		}
		// --- КОМАНДА /nginx (Проверка статуса) ---
		if update.Message.IsCommand() && update.Message.Command() == "nginx" {

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
			// Добавляем факт запроса и ответ нейронки в историю чата
			history[chatID] = append(history[chatID], api.Message{Role: "user", Content: "Какой статус у Nginx?"})
			history[chatID] = append(history[chatID], api.Message{Role: "assistant", Content: explain})

			continue
		}
		if update.Message.IsCommand() && update.Message.Command() == "top" {
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
			continue
		}
		if update.Message.IsCommand() && update.Message.Command() == "net" {
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
			continue
		}
		if update.Message.IsCommand() && update.Message.Command() == "ports" {
			// Выполняем команду ss -tulpn (TCP/UDP, Listening, Processes, Numbers)
			// Мы берем только TCP для краткости: ss -ltn
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
			continue
		}
		if update.Message.IsCommand() && update.Message.Command() == "restart" {
			serviceName := update.Message.CommandArguments() // Получаем текст после /restart
			if serviceName == "" {
				bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Укажите имя сервиса, например: `/restart nginx`"))
				continue
			}

			// Выполняем перезапуск через sudo
			cmd := exec.Command("sudo", "systemctl", "restart", serviceName)
			err := cmd.Run()

			statusMsg := ""
			if err != nil {
				statusMsg = fmt.Sprintf("❌ Ошибка при перезапуске `%s`: %v", serviceName, err)
			} else {
				statusMsg = fmt.Sprintf("🔄 Сервис `%s` успешно перезапущен!", serviceName)
			}

			// Просим нейронку прокомментировать действие
			var aiComment string
			prompt := fmt.Sprintf("Я только что перезапустил сервис %s. Напиши короткое подтверждение как суровый сисадмин.", serviceName)
			_ = client.Generate(ctx, &api.GenerateRequest{Model: "qwen2.5-coder:3b", Prompt: prompt}, func(resp api.GenerateResponse) {
				aiComment += resp.Response
			})

			msg := tgbotapi.NewMessage(chatID, statusMsg+"\n\n"+aiComment)
			msg.ParseMode = "Markdown"
			bot.Send(msg)
			continue
		}

		// --- КОМАНДА /status (Метрики системы) ---
		if update.Message.IsCommand() && update.Message.Command() == "status" {
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
			// Добавляем факт запроса и ответ нейронки в историю чата
			history[chatID] = append(history[chatID], api.Message{Role: "user", Content: "Команда /status: покажи метрики"})
			history[chatID] = append(history[chatID], api.Message{Role: "assistant", Content: summary})

			continue
		}

		// --- ОБЫЧНОЕ ОБЩЕНИЕ (С КОНТЕКСТОМ) ---
		bot.Send(tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping))

		// Добавляем сообщение пользователя в историю
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
			// Сохраняем ответ нейросети в историю
			history[chatID] = append(history[chatID], api.Message{Role: "assistant", Content: fullResponse})
			// Ограничиваем историю (последние 10 сообщений)
			if len(history[chatID]) > 10 {
				history[chatID] = history[chatID][len(history[chatID])-10:]
			}
		}

		bot.Send(tgbotapi.NewMessage(chatID, fullResponse))
	}
}
