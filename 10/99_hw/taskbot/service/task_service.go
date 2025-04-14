package service

import "fmt"

// Task описывает задачу
type Task struct {
	ID             int
	Title          string
	Assignee       string // имя или ID исполнителя
	Owner          string // создатель задачи
	OwnerChatID    int
	AssigneeChatID int

	// Можно расширить поля по потребностям
}

// TaskService описывает набор бизнес-операций над задачами
type TaskService interface {
	CreateTask(owner string, ownerChatID int, title string) (Task, error)
	AssignTask(taskID int, assigner string, assignerChatID int) (oldAssignee string, oldAssigneeChatID int, task Task, err error)
	UnassignTask(taskID int, requester string, requesterChatID int) (notAllowed bool, task Task, err error)
	ResolveTask(taskID int, requester string, requesterChatID int) (task Task, err error)
	GetAllTasks() ([]Task, error)
	GetTasksByAssignee(chatID int) ([]Task, error)
	GetTasksByOwner(chatID int) ([]Task, error)
}

type taskService struct {
	repo TaskRepository
}

// TaskRepository интерфейс, объявленный в слое репозитория
type TaskRepository interface {
	Create(task Task) (int, error)
	GetByID(id int) (Task, error)
	GetAll() ([]Task, error)
	Update(task Task) error
	Delete(id int) error
}

func NewTaskService(repo TaskRepository) TaskService {
	return &taskService{repo: repo}
}

func (s *taskService) CreateTask(owner string, ownerChatID int, title string) (Task, error) {
	task := Task{
		Title:       title,
		Owner:       owner,
		OwnerChatID: ownerChatID,
	}
	id, err := s.repo.Create(task)
	if err != nil {
		return Task{}, err
	}
	task.ID = id
	return task, nil
}

func (s *taskService) AssignTask(taskID int, assigner string, assignerChatID int) (oldAssignee string, oldAssigneeChatID int, task Task, err error) {
	task, err = s.repo.GetByID(taskID)
	if err != nil {
		return "", 0, Task{}, err
	}

	// Сохраняем информацию о предыдущем исполнителе, если он был назначен
	oldAssignee = task.Assignee
	oldAssigneeChatID = task.AssigneeChatID

	// Назначаем нового исполнителя
	task.Assignee = assigner
	task.AssigneeChatID = assignerChatID

	// Обновляем задачу в репозитории
	err = s.repo.Update(task)
	if err != nil {
		return "", 0, Task{}, err
	}

	return oldAssignee, oldAssigneeChatID, task, nil
}

func (s *taskService) UnassignTask(taskID int, requester string, requesterChatID int) (bool, Task, error) {
	// Получаем задачу по ID
	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return false, Task{}, err
	}
	// Если задача назначена не на того, кто делает запрос — сигнализируем о запрете операции.
	if task.Assignee != requester {
		return true, task, nil // notAllowed == true
	}
	// Снимаем назначение
	task.Assignee = ""
	task.AssigneeChatID = 0
	if err := s.repo.Update(task); err != nil {
		return false, Task{}, err
	}
	return false, task, nil
}

func (s *taskService) ResolveTask(taskID int, requester string, requesterChatID int) (Task, error) {
	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return Task{}, err
	}
	// Проверяем, что разрешить завершение может только назначенный исполнитель
	if task.AssigneeChatID != requesterChatID {
		return Task{}, fmt.Errorf("задачу может завершить только исполнитель")
	}
	// Удаляем задачу из хранилища
	if err := s.repo.Delete(taskID); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *taskService) GetTasksByAssignee(chatID int) ([]Task, error) {
	allTasks, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}
	var result []Task
	for _, t := range allTasks {
		if t.AssigneeChatID == chatID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (s *taskService) GetTasksByOwner(chatID int) ([]Task, error) {
	allTasks, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}
	var result []Task
	for _, t := range allTasks {
		if t.OwnerChatID == chatID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (s *taskService) GetAllTasks() ([]Task, error) {
	return s.repo.GetAll()
}
