# Zappy Server - README

graphisme 3d avec raylib

## Contexte et contraintes du sujet

Le sujet impose de créer un serveur mono-thread pour un système client/serveur. Les contraintes spécifiques sont :
1. **Mono-thread** : Le serveur doit fonctionner avec un seul thread.
2. **Pas de non-bloquant avec `fcntl`** : Les sockets ou fichiers ne doivent pas être configurés en mode non-bloquant.
3. **Pas d'interférence entre clients** : Les commandes d'un client doivent être exécutées séquentiellement, mais une commande lente ne doit pas bloquer les autres clients.
4. **Communication via TCP** : Les clients se connectent au serveur par des sockets TCP.
5. **Traitement indépendant des clients** : Chaque client a sa propre file d'attente de commandes.

---

## Solution en C avec `epoll`

### Description

En C, on peut utiliser `epoll` pour gérer plusieurs sockets en mode bloquant sans configurer explicitement les descripteurs en mode non-bloquant via `fcntl`. `epoll` permet de surveiller les descripteurs pour détecter les événements (lecture ou écriture possible) et ne retourne que lorsque des descripteurs sont prêts.

Chaque client possède :
- Une **file d'attente de commandes**, dans laquelle sont ajoutées les requêtes reçues.
- Une boucle principale qui :
  1. Utilise `epoll_wait` pour surveiller les sockets des clients et détecter les données disponibles.
  2. Traite les commandes de manière séquentielle pour chaque client.

### Mini-exemple de code C avec `epoll`

```
#include <sys/epoll.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <netinet/in.h>

#define MAX_EVENTS 10
#define MAX_QUEUE 10
#define BUFFER_SIZE 1024

typedef struct {
    int fd;                      // Descripteur du client
    char *command_queue[MAX_QUEUE]; // File d'attente des commandes
    int queue_start;
    int queue_end;
} Client;

Client clients[MAX_EVENTS];
int client_count = 0;

// Ajouter une commande dans la file d'attente d'un client
void add_command(Client *client, const char *command) {
    int next_queue = (client->queue_end + 1) % MAX_QUEUE;
    if (next_queue != client->queue_start) {
        client->command_queue[client->queue_end] = strdup(command);
        client->queue_end = next_queue;
    } else {
        printf("Command queue full for client %d\n", client->fd);
    }
}

// Obtenir la prochaine commande d'un client
char *get_next_command(Client *client) {
    if (client->queue_start == client->queue_end) return NULL;
    char *command = client->command_queue[client->queue_start];
    client->queue_start = (client->queue_start + 1) % MAX_QUEUE;
    return command;
}

void remove_client(int epfd, int client_fd) {
    epoll_ctl(epfd, EPOLL_CTL_DEL, client_fd, NULL);
    close(client_fd);
    printf("Client %d disconnected\n", client_fd);
}

int main() {
    int server_fd, epoll_fd;
    struct sockaddr_in server_addr;

    // Créer le socket serveur
    server_fd = socket(AF_INET, SOCK_STREAM, 0);
    server_addr.sin_family = AF_INET;
    server_addr.sin_addr.s_addr = INADDR_ANY;
    server_addr.sin_port = htons(4242);

    bind(server_fd, (struct sockaddr *)&server_addr, sizeof(server_addr));
    listen(server_fd, 10);

    // Créer une instance epoll
    epoll_fd = epoll_create1(0);
    struct epoll_event event, events[MAX_EVENTS];

    event.events = EPOLLIN; // Surveiller les lectures
    event.data.fd = server_fd;
    epoll_ctl(epoll_fd, EPOLL_CTL_ADD, server_fd, &event);

    while (1) {
        int n = epoll_wait(epoll_fd, events, MAX_EVENTS, -1);
        for (int i = 0; i < n; i++) {
            if (events[i].data.fd == server_fd) {
                // Nouvelle connexion
                int client_fd = accept(server_fd, NULL, NULL);
                event.events = EPOLLIN;
                event.data.fd = client_fd;
                epoll_ctl(epoll_fd, EPOLL_CTL_ADD, client_fd, &event);

                clients[client_count].fd = client_fd;
                clients[client_count].queue_start = 0;
                clients[client_count].queue_end = 0;
                client_count++;
                printf("New client connected: %d\n", client_fd);
            } else {
                // Lecture des données d'un client
                int client_fd = events[i].data.fd;
                char buffer[BUFFER_SIZE];
                int bytes_read = read(client_fd, buffer, sizeof(buffer));
                if (bytes_read > 0) {
                    buffer[bytes_read] = '\0';
                    printf("Received from client %d: %s\n", client_fd, buffer);
                    add_command(&clients[client_fd], buffer);
                } else if (bytes_read == 0) {
                    remove_client(epoll_fd, client_fd);
                }
            }
        }

        // Traitement des commandes
        for (int i = 0; i < client_count; i++) {
            char *command = get_next_command(&clients[i]);
            if (command) {
                printf("Processing command from client %d: %s\n", clients[i].fd, command);
                sleep(1); // Simuler un traitement lent
                write(clients[i].fd, "Command processed\n", 18);
                free(command);
            }
        }
    }

    close(server_fd);
    close(epoll_fd);
    return 0;
```
---

## Solution en Go

### Description

En Go, le runtime gère les sockets bloquantes de manière transparente grâce à des mécanismes comme `epoll` ou `kqueue`. Les goroutines permettent de gérer les clients de manière indépendante tout en respectant la contrainte de mono-thread.

Chaque client est géré dans une goroutine dédiée :
1. Lecture ligne par ligne des commandes avec `bufio.Scanner`.
2. Ajout des commandes dans une file d'attente partagée avec le serveur.
3. Le serveur traite les commandes séquentiellement pour chaque client dans une boucle principale.

### Mini-exemple de code Go

```
package main

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"
)

// Client représente une connexion avec une file d'attente de commandes.
type Client struct {
	conn   net.Conn
	queue  []string
	mu     sync.Mutex
	closed bool
}

// Ajouter une commande dans la file d'attente.
func (c *Client) AddCommand(cmd string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.queue = append(c.queue, cmd)
	}
}

// Récupérer et retirer la prochaine commande.
func (c *Client) NextCommand() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.queue) == 0 || c.closed {
		return "", false
	}
	cmd := c.queue[0]
	c.queue = c.queue[1:]
	return cmd, true
}

// Serveur
type Server struct {
	clients   map[int]*Client
	nextID    int
	mu        sync.Mutex
	newClient chan *Client
}

// Nouveau serveur
func NewServer() *Server {
	return &Server{
		clients:   make(map[int]*Client),
		newClient: make(chan *Client),
	}
}

// Ajouter un client au serveur.
func (s *Server) AddClient(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	client := &Client{conn: conn, queue: []string{}}
	s.clients[s.nextID] = client
	s.nextID++
	s.newClient <- client
}

// Supprimer un client.
func (s *Server) RemoveClient(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	client, ok := s.clients[id]
	if ok {
		client.conn.Close()
		delete(s.clients, id)
	}
}

// Lancer le traitement des clients.
func (s *Server) ProcessCommands() {
	for {
		s.mu.Lock()
		for id, client := range s.clients {
			cmd, ok := client.NextCommand()
			if ok {
				fmt.Printf("Processing command from client %d: %s\n", id, cmd)
				time.Sleep(1 * time.Second) // Simuler une commande lente
				client.conn.Write([]byte("Command processed\n"))
			}
		}
		s.mu.Unlock()
		time.Sleep(100 * time.Millisecond) // Prévenir le 100% CPU
	}
}

func main() {
	server := NewServer()
	go server.ProcessCommands()

	listener, err := net.Listen("tcp", ":4242")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server is running on port 4242")

	for {
		conn, err := listener.Accept() // Bloquant
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		server.AddClient(conn)
		go func(client *Client) {
			scanner := bufio.NewScanner(client.conn)
			for scanner.Scan() {
				cmd := scanner.Text()
				fmt.Printf("Received command: %s\n", cmd)
				client.AddCommand(cmd)
			}
			server.RemoveClient(client.conn.RemoteAddr().(*net.TCPAddr).Port)
			client.conn.Close()
		}(server.clients[server.nextID-1])
	}
}
```
