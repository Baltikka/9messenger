package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

// Message структура для представления сообщений
type Message struct {
	UserId  int `json:"userId"`
	ID      int
	chatId  int
	Content string `json:"content"`
}

type User struct {
	ID       int
	Username string `json:"username"`
}

type Chat struct {
	ID   int
	Name string `json:"name"`
}

type ChatUsers struct {
	userID int
	chatID int
}

var db *sql.DB

func main() {
	// Устанавливаем соединение с базой данных
	var err error
	db, err = sql.Open("mysql", "root:12345678@tcp(localhost:3306)/messenger")
	if err != nil {
		log.Fatal(err)
	}
	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")
	defer db.Close()
	// Инициализируем маршрутизатор
	r := mux.NewRouter()

	// Устанавливаем обработчики маршрутов
	r.HandleFunc("/users/register", registerUser).Methods("POST")               //+
	r.HandleFunc("/chats/create", createChat).Methods("POST")                   //+
	r.HandleFunc("/chats/users/add", addUsersToChat).Methods("PUT")             //+
	r.HandleFunc("/chats/{chatId}/messages", getMessages).Methods("GET")        //+
	r.HandleFunc("/chats/{chatId}/messages/add", createMessage).Methods("POST") //+
	r.HandleFunc("/messages/{Id}", deleteMessage).Methods("DELETE")             //+
	r.HandleFunc("/messages/{Id}", getMessage).Methods("GET")                   //+

	// Запускаем сервер на порту 8080
	log.Fatal(http.ListenAndServe(":8080", r))
}

func registerUser(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}
	var newUser User
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	// Валидация наличия имени
	if newUser.Username == "" {
		http.Error(w, "Username cannot be empty", http.StatusBadRequest)
		return
	}

	// Вставляем нового пользователя в базу данных
	result, err := db.Exec("INSERT INTO users (username) VALUES (?)", newUser.Username)
	if err != nil {
		log.Println("Error inserting user into the database:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Получаем ID только что созданного пользователя
	userID, _ := result.LastInsertId()
	newUser.ID = int(userID)
	// Возвращаем информацию о новом пользователе в формате JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newUser)
}

func createChat(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}
	var newChat Chat
	err := json.NewDecoder(r.Body).Decode(&newChat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	// Валидация наличия названия
	if newChat.Name == "" {
		http.Error(w, "Chat name cannot be empty", http.StatusBadRequest)
		return
	}

	// Вставляем нового чата в базу данных
	result, err := db.Exec("INSERT INTO chats (chatname) VALUES (?)", newChat.Name)
	if err != nil {
		log.Println("Error inserting user into the database:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Получаем ID только что созданного чата
	chatID, _ := result.LastInsertId()
	newChat.ID = int(chatID)
	// Возвращаем информацию о новом чате в формате JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newChat)
}

func addUsersToChat(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}
	var requestBody map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Невозможно разобрать JSON-запрос", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	// Извлечение ID чата и массива ID пользователей
	chatID, chatIDExists := requestBody["chatID"].(float64)
	userIDs, userIDsExist := requestBody["userIDs"].([]interface{})

	chatIDInt := int(chatID)

	if !chatIDExists || !userIDsExist {
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	for _, userID := range userIDs {
		// Добавляем нового пользователя в чате в базе данных
		_, err := db.Exec("INSERT INTO chatuser (userID, chatID) VALUES (?, ?)", userID, chatIDInt)
		if err != nil {
			log.Println("Error inserting user into the database:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func getMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var err error
	chatId, err := strconv.Atoi(vars["chatId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Получение сообщения из базы данных
	rows, err := db.Query(`SELECT chats.chatname, messages.content, users.username
	FROM messages
	JOIN users ON users.ID = messages.userId
	JOIN chats ON chats.ID = messages.chatId
	where messages.chatId = (?)`, chatId)
	if err != nil {
		log.Println("Error deleting message from the database:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type message struct {
		Chatname string
		Content  string
		Username string
	}
	var messagesinfo []message
	for rows.Next() {
		var messageinfo message
		if err := rows.Scan(&messageinfo.Chatname, &messageinfo.Content, &messageinfo.Username); err != nil {
			log.Println(err.Error())
		}
		messagesinfo = append(messagesinfo, messageinfo)
	}
	// Возвращаем информацию о сообщениях в формате JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messagesinfo)
}

func getMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var requiredMessage Message
	var err error
	requiredMessage.ID, err = strconv.Atoi(vars["Id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Получение сообщения из базы данных
	rows, err := db.Query(`SELECT chats.chatname, messages.content, users.username
	FROM messages
	JOIN users ON users.ID = messages.userId
	JOIN chats ON chats.ID = messages.chatId
	where messages.ID = (?)`, requiredMessage.ID)
	if err != nil {
		log.Println("Error deleting message from the database:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type message struct {
		Chatname string
		Content  string
		Username string
	}
	var messageinfo message
	for rows.Next() {
		if err := rows.Scan(&messageinfo.Chatname, &messageinfo.Content, &messageinfo.Username); err != nil {
			log.Println(err.Error())
		}
	}
	// Возвращаем информацию о сообщении в формате JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messageinfo)
}

func createMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var newMessage Message
	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&newMessage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	newMessage.chatId, err = strconv.Atoi(vars["chatId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	// Валидации
	if newMessage.UserId == 0 {
		http.Error(w, "userId cannot be empty", http.StatusBadRequest)
		return
	}
	if newMessage.Content == "" {
		http.Error(w, "content cannot be empty", http.StatusBadRequest)
		return
	}

	// Вставляем нового сообщения в базу данных
	result, err := db.Exec("INSERT INTO messages (userId, chatId, content) VALUES (?, ?, ?)", newMessage.UserId, newMessage.chatId, newMessage.Content)
	if err != nil {
		log.Println("Error inserting message into the database:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Получаем ID сообщения
	messageID, _ := result.LastInsertId()
	newMessage.ID = int(messageID)
	// Возвращаем информацию о новом сообщении в формате JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newMessage)
}

func deleteMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var newMessage Message
	var err error
	newMessage.ID, err = strconv.Atoi(vars["Id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Удаление сообщения из базы данных
	_, err = db.Exec("DELETE FROM messages WHERE messages.ID = (?)", newMessage.ID)
	if err != nil {
		log.Println("Error deleting message from the database:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
