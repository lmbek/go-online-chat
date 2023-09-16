package main

import (
	"bufio"
	"embed"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"golang.org/x/net/websocket"
)

//go:embed frontend/*
var content embed.FS

var (
	clients      = make(map[*websocket.Conn]string) // Map connections to user names
	clientsMutex = sync.Mutex{}
	chatHistory  []string
	historyMutex = sync.RWMutex{}
	broadcast    = make(chan string)
	historyFile  = "./chat_history.txt"
	domain       = "127.0.0.1"
)

func main() {
	loadChatHistory()

	http.HandleFunc("/", serveHTTP)
	http.Handle("/ws", websocket.Handler(handleConnections))

	go broadcastMessages()
	/*
		// Paths to your TLS certificate and private key files
			onlineCertificateFilePath := "/etc/letsencrypt/live/lmbek.dk/fullchain.pem"
			onlineKeyFilePath := "/etc/letsencrypt/live/lmbek.dk/privkey.pem"

			// Load the TLS certificate and private key
			tlsCert, errCert := tls.LoadX509KeyPair(onlineCertificateFilePath, onlineKeyFilePath)
			if errCert != nil {
				fmt.Println("Error loading TLS certificate and key:", errCert)
				return
			}
		// Create a TLS configuration
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
		}

		// Create an HTTPS server with the TLS configuration
		server := &http.Server{
			Addr:      domain + ":8080",
			TLSConfig: tlsConfig,
		}

		fmt.Println("Server is running on lmbek.dk:8080")
		fmt.Println("Listening on Port 8080 (HTTPS)")

		httpsError := server.ListenAndServeTLS(onlineCertificateFilePath, onlineKeyFilePath)
		if httpsError != nil {
			fmt.Println("Web server (HTTPS): ", httpsError)
		}
	*/

	fmt.Println("Server is running on http://localhost:8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println(err)
	}
}

func serveHTTP(w http.ResponseWriter, r *http.Request) {
	// Serve other files by prepending "/frontend" to the path.
	r.URL.Path = "/frontend" + r.URL.Path
	http.FileServer(http.FS(content)).ServeHTTP(w, r)
}

func handleConnections(ws *websocket.Conn) {
	// Prompt the user for a name
	//websocket.Message.Send(ws, "Please enter your name:")
	var userName string

	// Read and set the user's name
	err := websocket.Message.Receive(ws, &userName)
	if err != nil {
		return
	}

	// Add the user to the map of connections
	clientsMutex.Lock()
	clients[ws] = userName
	clientsMutex.Unlock()

	// Send the updated list of online users to all clients
	updateOnlineUsers()

	// Send the chat history to the new user
	sendChatHistory(ws)

	// Broadcast a welcome message to all clients
	message := fmt.Sprintf("User %s has joined the chat", userName)
	broadcast <- message

	var msg string
	for {
		// Receive messages from the user
		err := websocket.Message.Receive(ws, &msg)
		if err != nil {
			// Remove the user from the map of connections
			clientsMutex.Lock()
			delete(clients, ws)
			clientsMutex.Unlock()

			// Send the updated list of online users to all clients
			updateOnlineUsers()

			// Broadcast a message that the user has left
			message := fmt.Sprintf("User %s has left the chat", userName)
			broadcast <- message

			return
		}

		// Broadcast the message to all clients
		message := fmt.Sprintf("%s: %s", userName, msg)
		appendChatHistory(message)
		broadcast <- message
	}
}

func broadcastMessages() {
	for {
		select {
		case message := <-broadcast:
			clientsMutex.Lock()
			for client := range clients {
				if err := websocket.Message.Send(client, message); err != nil {
					// Remove the disconnected client
					delete(clients, client)
				}
			}
			clientsMutex.Unlock()
		}
	}
}

func sendChatHistory(ws *websocket.Conn) {
	historyMutex.RLock()
	defer historyMutex.RUnlock()

	for _, message := range chatHistory {
		// Send each message to the new user
		if err := websocket.Message.Send(ws, message); err != nil {
			return
		}
	}
}

func appendChatHistory(message string) {
	historyMutex.Lock()
	chatHistory = append(chatHistory, message)
	historyMutex.Unlock()

	// Append the message to the history file
	if err := appendToFile(historyFile, message+"\n"); err != nil {
		fmt.Println("Error appending to history file:", err)
	}
}

func loadChatHistory() {
	// Read chat history from the file and populate the chatHistory slice
	file, err := os.Open(historyFile)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		chatHistory = append(chatHistory, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading history file:", err)
	}
}

func appendToFile(filename, text string) error {
	fmt.Println("writing to file: ", text)
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Println("Saving as: ", filename)

	_, err = file.WriteString(text)
	return err
}

func updateOnlineUsers() {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	var onlineUsers []string
	for _, userName := range clients {
		onlineUsers = append(onlineUsers, userName)
	}

	// Broadcast the updated list of online users to all clients
	message := "Online Users: " + strings.Join(onlineUsers, ", ")
	broadcast <- message
}
