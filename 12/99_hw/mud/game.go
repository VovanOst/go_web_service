package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Game хранит глобальное состояние мира: комнаты, игроков и состояние дверей
var game *Game

type Game struct {
	Rooms   map[string]*Room   // все комнаты по имени
	Players map[string]*Player // все игроки по имени
	Doors   map[string]bool    // состояние дверей: true=открыта, false=закрыта
	Mutex   sync.RWMutex       // для синхронизации доступа
}

type Room struct {
	Name        string             // имя комнаты
	Description string             // описание для перемещения
	Neighbors   []string           // соседние комнаты
	Items       map[string]string  // предметы в комнате: "имя"->"локация"
	Players     map[string]*Player // игроки в комнате
	Logic       *RoomLogic         // логика для команды "осмотреться"
}

type RoomLogic struct {
	Look func(p *Player) string // реализация осмотра
}

type Player struct {
	Name        string          // имя игрока
	Inventory   map[string]bool // инвентарь
	Room        *Room           // текущая комната
	HasBackpack bool            // флаг: надет ли рюкзак
	Output      chan string     // канал для отправки сообщений
}

func initGame() {
	game = &Game{
		Rooms:   make(map[string]*Room),
		Players: make(map[string]*Player),
		Doors:   make(map[string]bool),
	}
	game.Doors["дверь"] = false // дверь закрыта по умолчанию

	kitchen := &Room{
		Name:        "кухня",
		Description: "кухня, ничего интересного",
		Neighbors:   []string{"коридор"},
		Items:       map[string]string{"чай": "на столе"},
		Players:     make(map[string]*Player),
	}
	corridor := &Room{
		Name:        "коридор",
		Description: "ничего интересного",
		Neighbors:   []string{"кухня", "комната", "улица"},
		Items:       make(map[string]string),
		Players:     make(map[string]*Player),
	}
	room := &Room{
		Name:        "комната",
		Description: "ты в своей комнате",
		Neighbors:   []string{"коридор"},
		Items:       map[string]string{"ключи": "на столе", "конспекты": "на столе", "рюкзак": "на стуле"},
		Players:     make(map[string]*Player),
	}
	street := &Room{
		Name:        "улица",
		Description: "на улице весна",
		Neighbors:   []string{"домой"},
		Items:       make(map[string]string),
		Players:     make(map[string]*Player),
	}

	// регистрируем комнаты
	game.Rooms[kitchen.Name] = kitchen
	game.Rooms[corridor.Name] = corridor
	game.Rooms[room.Name] = room
	game.Rooms[street.Name] = street

	// логика осмотра кухни: зависит от рюкзака и наличия других игроков
	kitchen.Logic = &RoomLogic{Look: func(p *Player) string {
		// базовое описание
		var desc string
		if p.HasBackpack {
			desc = "ты находишься на кухне, на столе чай, надо идти в универ. можно пройти - коридор"
		} else {
			desc = "ты находишься на кухне, на столе чай, надо собрать рюкзак и идти в универ. можно пройти - коридор"
		}
		// добавляем информацию об остальных игроках
		others := []string{}
		for name := range p.Room.Players {
			if name != p.Name {
				others = append(others, name)
			}
		}
		if len(others) > 0 {
			sort.Strings(others)
			desc += ". Кроме вас тут ещё " + strings.Join(others, ", ")
		}
		return desc
	}}

	corridor.Logic = &RoomLogic{Look: func(p *Player) string {
		return fmt.Sprintf("%s. можно пройти - %s", corridor.Description, strings.Join(corridor.Neighbors, ", "))
	}}

	room.Logic = &RoomLogic{Look: func(p *Player) string {
		r := p.Room
		if len(r.Items) == 0 {
			return fmt.Sprintf("пустая комната. можно пройти - %s", strings.Join(r.Neighbors, ", "))
		}
		parts := []string{}
		// группировка: на столе
		onTable := []string{}
		for item, loc := range r.Items {
			if loc == "на столе" {
				onTable = append(onTable, item)
			}
		}
		if len(onTable) > 0 {
			sort.Strings(onTable)
			parts = append(parts, fmt.Sprintf("на столе: %s", strings.Join(onTable, ", ")))
		}
		// группировка: на стуле
		onChair := []string{}
		for item, loc := range r.Items {
			if loc == "на стуле" {
				onChair = append(onChair, item)
			}
		}
		if len(onChair) > 0 {
			sort.Strings(onChair)
			parts = append(parts, fmt.Sprintf("на стуле - %s", strings.Join(onChair, ", ")))
		}
		return fmt.Sprintf("%s. можно пройти - %s", strings.Join(parts, ", "), strings.Join(r.Neighbors, ", "))
	}}

	street.Logic = &RoomLogic{Look: func(p *Player) string {
		return fmt.Sprintf("%s. можно пройти - %s", street.Description, strings.Join(street.Neighbors, ", "))
	}}
}

// NewPlayer создаёт нового игрока по имени
func NewPlayer(name string) *Player {
	return &Player{Name: name, Inventory: make(map[string]bool), Output: make(chan string, 10)}
}

// addPlayer размещает игрока в стартовой комнате и выводит её описание
func addPlayer(p *Player) {
	game.Mutex.Lock()
	defer game.Mutex.Unlock()
	game.Players[p.Name] = p
	// ставим игрока в начальную комнату
	start := game.Rooms["кухня"]
	p.Room = start
	start.Players[p.Name] = p
	// NOTE: убираем первоначальную рассылку описания, отправляем вывод только по явным командам
}

// GetOutput возвращает канал вывода игрока
func (p *Player) GetOutput() chan string { return p.Output }

// HandleInput разбирает ввод и отправляет ответ
func (p *Player) HandleInput(input string) {
	parts := strings.Split(input, " ")
	cmd, args := parts[0], parts[1:]
	res := handleCommand(p, cmd, args)
	if res != "" {
		p.send(res)
	}
}

// handleCommand обрабатывает все доступные команды
func handleCommand(p *Player, cmd string, args []string) string {
	switch cmd {
	case "осмотреться":
		return p.Room.Logic.Look(p)

	case "идти":
		if len(args) == 0 {
			return "куда?"
		}
		dest := args[0]
		if dest == "улица" && !game.Doors["дверь"] {
			return "дверь закрыта"
		}
		for _, nbr := range p.Room.Neighbors {
			if nbr == dest { // исправлено: dest вместо dist
				delete(p.Room.Players, p.Name)
				newRoom := game.Rooms[dest]
				p.Room = newRoom
				newRoom.Players[p.Name] = p
				return fmt.Sprintf("%s. можно пройти - %s", newRoom.Description, strings.Join(newRoom.Neighbors, ", "))
			}
		}
		return fmt.Sprintf("нет пути в %s", dest)

	case "одеть":
		if len(args) == 0 {
			return "что одеть?"
		}
		item := args[0]
		if item != "рюкзак" {
			return "нет такого"
		}
		if _, ok := p.Room.Items[item]; !ok {
			return "нет такого"
		}
		p.HasBackpack = true
		delete(p.Room.Items, item)
		return fmt.Sprintf("вы одели: %s", item)

	case "взять":
		if len(args) == 0 {
			return "что взять?"
		}
		item := args[0]
		if _, ok := p.Room.Items[item]; !ok {
			return "нет такого"
		}
		if !p.HasBackpack {
			return "некуда класть"
		}
		p.Inventory[item] = true
		delete(p.Room.Items, item)
		return fmt.Sprintf("предмет добавлен в инвентарь: %s", item)

	case "применить":
		if len(args) < 2 {
			return "что применять?"
		}
		item, target := args[0], args[1]
		if !p.Inventory[item] {
			return fmt.Sprintf("нет предмета в инвентаре - %s", item)
		}
		if item == "ключи" && target == "дверь" {
			game.Doors["дверь"] = true
			return "дверь открыта"
		}
		return "не к чему применить"

	case "сказать":
		if len(args) == 0 {
			return "что сказать?"
		}
		msg := strings.Join(args, " ")
		for _, pl := range p.Room.Players {
			pl.send(fmt.Sprintf("%s говорит: %s", p.Name, msg))
		}
		return ""

	case "сказать_игроку":
		if len(args) < 1 {
			return "что сказать?"
		}
		targetName := args[0]
		pl, ok := p.Room.Players[targetName]
		if !ok {
			return "тут нет такого игрока"
		}
		if len(args) < 2 {
			pl.send(fmt.Sprintf("%s выразительно молчит, смотря на вас", p.Name))
			return ""
		}
		text := strings.Join(args[1:], " ")
		pl.send(fmt.Sprintf("%s говорит вам: %s", p.Name, text)) // <<< исправлено кавычки
		return ""

	default:
		return "неизвестная команда"
	}
}

// отправка сообщения
func (p *Player) send(msg string) {
	select {
	case p.Output <- msg:
	default:
	}
}

func main() {}
