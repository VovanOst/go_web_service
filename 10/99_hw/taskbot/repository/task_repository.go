package repository

import (
	"fmt"
	"taskbot/service"
)

// TaskRepository описывает методы по работе с задачами
type TaskRepository interface {
	Create(task service.Task) (int, error)
	GetByID(id int) (service.Task, error)
	GetAll() ([]service.Task, error)
	Update(task service.Task) error
	Delete(id int) error
}

// inMemoryTaskRepository – простая реализация на основе слайса
type inMemoryTaskRepository struct {
	tasks  []service.Task
	nextID int
}

// NewInMemoryTaskRepository создаёт новый репозиторий
func NewInMemoryTaskRepository() TaskRepository {
	return &inMemoryTaskRepository{
		tasks:  make([]service.Task, 0),
		nextID: 1,
	}
}

func (r *inMemoryTaskRepository) Create(task service.Task) (int, error) {
	task.ID = r.nextID
	r.nextID++
	r.tasks = append(r.tasks, task)
	return task.ID, nil
}

func (r *inMemoryTaskRepository) GetByID(id int) (service.Task, error) {
	for _, t := range r.tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return service.Task{}, fmt.Errorf("task with id %d not found", id)
}

func (r *inMemoryTaskRepository) GetAll() ([]service.Task, error) {
	return r.tasks, nil
}

func (r *inMemoryTaskRepository) Update(task service.Task) error {
	for i, t := range r.tasks {
		if t.ID == task.ID {
			r.tasks[i] = task
			return nil
		}
	}
	return fmt.Errorf("task with id %d not found", task.ID)
}

func (r *inMemoryTaskRepository) Delete(id int) error {
	for i, t := range r.tasks {
		if t.ID == id {
			r.tasks = append(r.tasks[:i], r.tasks[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("task with id %d not found", id)
}
