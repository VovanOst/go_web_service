package taskbot

import (
	"context"
	"fmt"
	"net/http"
	"taskbot/delivery"
	"taskbot/repository"
	"taskbot/router"
	"taskbot/service"
	// Импорты других пакетов (delivery, router, service, repository)
)

//var WebhookURL = "http://127.0.0.1:8081"
//var BotToken = "_golangcourse_test"

func StartTaskBot(ctx context.Context, httpListenAddr string) error {
	// Инициализация приложения, как описано ранее
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
}
