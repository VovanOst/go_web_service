package main

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"taskbot/pkg/taskbot"
)

var WebhookURL = "http://127.0.0.1:8081"
var BotToken = "_golangcourse_test"

// Если используется в тестах:
const BotChatID = 100500

/*func startTaskBot(ctx context.Context, httpListenAddr string) error {
	// Инициализируем хранилище (repository) - используем in-memory реализацию
	taskRepo := repository.NewInMemoryTaskRepository()

	// Инициализируем бизнес-логику (service)
	taskService := service.NewTaskService(taskRepo)

	// Инициализируем роутер, который будет парсить команды и вызывать методы сервиса
	cmdRouter := router.NewCommandRouter(taskService)

	// Инициализируем delivery слой, который связывает работу с Телеграм API и HTTP-сервер
	botDelivery, err := delivery.NewTelegramDelivery(cmdRouter)
	if err != nil {
		return fmt.Errorf("failed to create telegram delivery: %w", err)
	}

	// Получаем http-хэндлер от delivery слоя
	httpHandler := botDelivery.Handler()

	fmt.Println("start server at", httpListenAddr)
	// Стартуем HTTP сервер, который будет получать уведомления от эмулятора телеграма
	return http.ListenAndServe(httpListenAddr, httpHandler)
}*/

func main() {
	if err := taskbot.StartTaskBot(context.Background(), ":8081"); err != nil {
		log.Fatalln(err)
	}
}

// это заглушка чтобы импорт сохранился
func __dummy() {
	tgbotapi.APIEndpoint = "_dummy"
}
