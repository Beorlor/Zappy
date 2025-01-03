package server

type Player struct {
	ID             int
	Team           string
	X, Y           int
	Orientation    int
	Inventory      map[string]int
	CommandChannel chan string
	Life           int
}

func NewPlayer(id int, team string, mapWidth, mapHeight int) *Player {
	return &Player{
		ID:             id,
		Team:           team,
		X:              id % mapWidth,
		Y:              id % mapHeight,
		Orientation:    1,
		Inventory:      map[string]int{"nourriture": 10},
		CommandChannel: make(chan string, 10),
		Life:           1260, // 126/t secondes au départ
	}
}

func (p *Player) AddCommand(command string) bool {
	select {
	case p.CommandChannel <- command:
		return true
	default:
		return false // Le channel est plein
	}
}

func (p *Player) Move(mapWidth, mapHeight int) {
	p.X = (p.X + 1) % mapWidth
}
