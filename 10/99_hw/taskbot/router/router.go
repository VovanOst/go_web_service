package router

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"strconv"
	"strings"

	"taskbot/service"
)

// CommandRouter принимает команды и вызывает методы сервиса
type CommandRouter struct {
	svc service.TaskService
}

func NewCommandRouter(svc service.TaskService) *CommandRouter {
	return &CommandRouter{svc: svc}
}

// Route обрабатывает входящую строку (сообщение) и возвращает ответ
func (r *CommandRouter) Route(from *tgbotapi.User, text string) map[int]string {
	result := make(map[int]string)
	currentChat := from.ID
	text = strings.TrimSpace(text)
	switch {
	case text == "/tasks":
		// Пример получения списка задач для пользователя
		tasks, err := r.svc.GetAllTasks()
		if err != nil {
			result[currentChat] = fmt.Sprintf("Ошибка: %v", err)
			return result
		}
		if len(tasks) == 0 {
			result[currentChat] = "Нет задач"
			return result
		}
		var outLines []string
		for _, t := range tasks {
			line := fmt.Sprintf("%d. %s by @%s", t.ID, t.Title, t.Owner)
			if t.Assignee == "" {
				outLines = append(outLines, line+"\n/assign_"+strconv.Itoa(t.ID))
			} else if t.AssigneeChatID == currentChat {
				outLines = append(outLines, line+"\nassignee: я\n/unassign_"+strconv.Itoa(t.ID)+" /resolve_"+strconv.Itoa(t.ID))
			} else {
				outLines = append(outLines, line+"\nassignee: @"+t.Assignee)
			}
		}
		result[currentChat] = strings.Join(outLines, "\n\n")
	case strings.HasPrefix(text, "/new"):
		parts := strings.SplitN(text, " ", 2)
		if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
			result[currentChat] = "Ошибка: нужно указать текст задачи"
			return result
		}
		title := parts[1]
		task, err := r.svc.CreateTask(from.UserName, currentChat, title)
		if err != nil {
			result[currentChat] = fmt.Sprintf("Ошибка: %v", err)
			return result
		}
		result[currentChat] = fmt.Sprintf("Задача \"%s\" создана, id=%d", task.Title, task.ID)
	case strings.HasPrefix(text, "/assign_"):
		idStr := strings.TrimPrefix(text, "/assign_")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			result[currentChat] = "Неверный формат ID"
			return result
		}
		oldAssignee, oldAssigneeChatID, task, err := r.svc.AssignTask(id, from.UserName, currentChat)
		if err != nil {
			result[currentChat] = fmt.Sprintf("Ошибка: %v", err)
			return result
		}
		// уведомляем пользователя, который инициировал команду
		result[currentChat] = fmt.Sprintf("Задача \"%s\" назначена на вас", task.Title)
		if oldAssignee != "" {
			// Если задача уже была назначена, уведомляем лишь предыдущего исполнителя.
			if oldAssigneeChatID != currentChat {
				result[oldAssigneeChatID] = fmt.Sprintf("Задача \"%s\" назначена на @%s", task.Title, from.UserName)
			}
		} else {
			// Если задачи раньше не было назначено – уведомляем владельца, если он не делает назначение сам.
			if task.OwnerChatID != currentChat {
				result[task.OwnerChatID] = fmt.Sprintf("Задача \"%s\" назначена на @%s", task.Title, from.UserName)
			}
		}
	case strings.HasPrefix(text, "/unassign_"):
		idStr := strings.TrimPrefix(text, "/unassign_")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			result[currentChat] = "Неверный формат ID"
			return result
		}
		notAllowed, task, err := r.svc.UnassignTask(id, from.UserName, currentChat)
		if err != nil {
			result[currentChat] = fmt.Sprintf("Ошибка: %v", err)
			return result
		}
		if notAllowed {
			result[currentChat] = "Задача не на вас"
			return result
		}
		result[currentChat] = "Принято"
		if task.OwnerChatID != currentChat {
			result[task.OwnerChatID] = fmt.Sprintf("Задача \"%s\" осталась без исполнителя", task.Title)
		}
	case strings.HasPrefix(text, "/resolve_"):
		idStr := strings.TrimPrefix(text, "/resolve_")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			result[currentChat] = "Неверный формат ID"
			return result
		}
		task, err := r.svc.ResolveTask(id, from.UserName, currentChat)
		if err != nil {
			result[currentChat] = fmt.Sprintf("Ошибка: %v", err)
			return result
		}
		result[currentChat] = fmt.Sprintf("Задача \"%s\" выполнена", task.Title)
		// Если владелец задачи отличается от текущего пользователя,
		// отправляем уведомление владельцу, добавляя к тексту информацию об исполнителе
		if task.OwnerChatID != currentChat {
			result[task.OwnerChatID] = fmt.Sprintf("Задача \"%s\" выполнена @%s", task.Title, task.Assignee)
		}
	case text == "/my":
		tasks, err := r.svc.GetTasksByAssignee(currentChat)
		if err != nil {
			result[currentChat] = fmt.Sprintf("Ошибка: %v", err)
			return result
		}
		if len(tasks) == 0 {
			result[currentChat] = "Нет задач"
			return result
		}
		var outLines []string
		for _, t := range tasks {
			// Формат: "2. сделать ДЗ по курсу by @ppetrov\n/unassign_2 /resolve_2"
			line := fmt.Sprintf("%d. %s by @%s", t.ID, t.Title, t.Owner)
			line += fmt.Sprintf("\n/unassign_%d /resolve_%d", t.ID, t.ID)
			outLines = append(outLines, line)
		}
		result[currentChat] = strings.Join(outLines, "\n\n")
	case text == "/owner":
		tasks, err := r.svc.GetTasksByOwner(currentChat)
		if err != nil {
			result[currentChat] = fmt.Sprintf("Ошибка: %v", err)
			return result
		}
		if len(tasks) == 0 {
			result[currentChat] = "Нет задач"
			return result
		}
		var outLines []string
		for _, t := range tasks {
			// Формат: "2. сделать ДЗ по курсу by @ppetrov\n/unassign_2 /resolve_2"
			line := fmt.Sprintf("%d. %s by @%s", t.ID, t.Title, t.Owner)
			line += fmt.Sprintf("\n/assign_%d", t.ID)
			outLines = append(outLines, line)
		}
		result[currentChat] = strings.Join(outLines, "\n\n")
	default:
		result[currentChat] = "Неизвестная команда"
	}
	return result
}
