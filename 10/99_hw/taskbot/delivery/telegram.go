package delivery

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"taskbot/config"

	"taskbot/router"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type TelegramDelivery struct {
	router *router.CommandRouter
}

func NewTelegramDelivery(router *router.CommandRouter) (*TelegramDelivery, error) {
	// здесь можно добавить инициализацию, если потребуется
	return &TelegramDelivery{router: router}, nil
}

func (d *TelegramDelivery) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "не могу прочитать тело запроса", http.StatusBadRequest)
			return
		}
		var update tgbotapi.Update
		if err := json.Unmarshal(body, &update); err != nil {
			http.Error(w, "неверный формат json", http.StatusBadRequest)
			return
		}

		var from *tgbotapi.User
		var text string
		if update.Message != nil {
			from = update.Message.From
			text = update.Message.Text
		} else {
			http.Error(w, "нет сообщения", http.StatusBadRequest)
			return
		}

		log.Printf("Получено сообщение от %s: %s", from.UserName, text)

		// Получаем уведомления от роутера: ключ — chatID, значение — текст сообщения.
		responses := d.router.Route(from, text)

		// Для каждого сформированного уведомления отправляем POST запрос к Telegram API.
		for chat, txt := range responses {
			msg := tgbotapi.NewMessage(int64(chat), txt)
			// Вместо формирования JSON, сформируем данные формы.
			formData := url.Values{}
			formData.Set("chat_id", strconv.FormatInt(msg.ChatID, 10))
			formData.Set("text", msg.Text)

			// Формируем URL для вызова sendMessage
			urlStr := fmt.Sprintf(tgbotapi.APIEndpoint, config.BotToken, "sendMessage")

			// Отправляем POST запрос с данными формы
			resp, err := http.Post(urlStr, "application/x-www-form-urlencoded", strings.NewReader(formData.Encode()))
			if err != nil {
				http.Error(w, "ошибка отправки сообщения", http.StatusInternalServerError)
				return
			}
			resp.Body.Close()
			log.Printf("Отправлено сообщение в чат %d: %s", chat, txt)
		}

		// Можно вернуть пустой ответ с кодом 200
		w.WriteHeader(http.StatusOK)
	})
}
