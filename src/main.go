package main

import (
	"log"
	"time"
	"zappy-server/server"
)

func main() {
	// Configuration initiale du serveur
	port := 12345
	width, height := 10, 10
	t := 100

	// Créer le serveur
	s := server.NewServer(port, width, height, t)

	// Lancer le serveur
	go s.Start()

	// Lancer la boucle principale
	tickDuration := time.Second / time.Duration(t)
	s.MainLoop(tickDuration)
}
