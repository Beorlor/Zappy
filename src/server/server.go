package server

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"
)

type Server struct {
	Port       int
	Width      int
	Height     int
	TickRate   int
	Map        [][]*Tile
	Players    map[int]*Player
	playerLock sync.RWMutex
}

func NewServer(port, width, height, tickRate int) *Server {
	// Initialisation de la carte et des joueurs
	mapData := CreateMap(width, height)
	return &Server{
		Port:     port,
		Width:    width,
		Height:   height,
		TickRate: tickRate,
		Map:      mapData,
		Players:  make(map[int]*Player),
	}
}

func (s *Server) Start() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Printf("Serveur démarré sur le port %d\n", s.Port)

	playerID := 0
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		playerID++
		go s.HandleConnection(conn, playerID)
	}
}

func (s *Server) HandleConnection(conn net.Conn, id int) {
	defer conn.Close()

	// Envoi du message de bienvenue
	fmt.Fprint(conn, "BIENVENUE\n")

	// Lecture du nom de l'équipe
	reader := bufio.NewScanner(conn)
	if !reader.Scan() {
		return
	}
	teamName := reader.Text()

	// Ajouter un nouveau joueur
	player := NewPlayer(id, teamName, s.Width, s.Height)
	s.AddPlayer(player)

	// Répondre au client
	fmt.Fprintf(conn, "0\n%d %d\n", s.Width, s.Height)

	// Gérer les commandes
	for reader.Scan() {
		command := reader.Text()
		if !player.AddCommand(command) {
			fmt.Fprint(conn, "suc\n") // Trop de commandes
		}
	}
}

func (s *Server) AddPlayer(player *Player) {
	s.playerLock.Lock()
	defer s.playerLock.Unlock()
	s.Players[player.ID] = player
}

func (s *Server) MainLoop(tickDuration time.Duration) {
	ticker := time.NewTicker(tickDuration)
	defer ticker.Stop()

	for range ticker.C {
		//important ne pas defer ici
		s.playerLock.RLock()
		for _, player := range s.Players {


			// gerer la vie et la mort du joueurs
			// gerer la duree de son action en cours,
			// si pas daction en cours, gerer les commandes

			select {
			case command := <-player.CommandChannel:
				s.ExecuteCommand(player, command)
			default:
				// Pas de commande à traiter
			}
		}
		s.playerLock.RUnlock()
	}
}

func (s *Server) ExecuteCommand(player *Player, command string) {
	switch command {
	case "avance":
		player.Move(s.Width, s.Height)
		fmt.Printf("Joueur %d avancé à (%d, %d)\n", player.ID, player.X, player.Y)
	default:
		fmt.Printf("Commande inconnue: %s\n", command)
	}
}
