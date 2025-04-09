package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
)

// Структура клуба
type ComputerClub struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Address      string  `json:"address"`
	PricePerHour float64 `json:"price_per_hour"`
	AvailablePCs int     `json:"available_pcs"`
}

// Глобальные переменные
var client *firestore.Client
var firebaseAuth *auth.Client

// Инициализация Firestore и Firebase Auth
func initFirestore() {
	ctx := context.Background()
	opt := option.WithCredentialsFile("space-fcde8-firebase-adminsdk-fbsvc-d19e7b688e.json")
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatalf("Ошибка подключения к Firebase: %v", err)
	}

	client, err = app.Firestore(ctx)
	if err != nil {
		log.Fatalf("Ошибка создания клиента Firestore: %v", err)
	}

	firebaseAuth, err = app.Auth(ctx)
	if err != nil {
		log.Fatalf("Ошибка инициализации Firebase Auth: %v", err)
	}
}

// Middleware для проверки аутентификации (без проверки роли)
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No Authorization header"})
			c.Abort()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header"})
			c.Abort()
			return
		}

		decodedToken, err := firebaseAuth.VerifyIDToken(context.Background(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Добавляем UID пользователя в контекст
		c.Set("uid", decodedToken.UID)
		c.Next()
	}
}

// Получение всех клубов
func getAllClubs(c *gin.Context) {
	var clubs []ComputerClub
	docs, err := client.Collection("clubs").Documents(context.Background()).GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, doc := range docs {
		var club ComputerClub
		doc.DataTo(&club)
		club.ID = doc.Ref.ID
		clubs = append(clubs, club)
	}

	c.JSON(http.StatusOK, clubs)
}

// Получение клуба по ID
func getClubByID(c *gin.Context) {
	id := c.Param("id")
	doc, err := client.Collection("clubs").Doc(id).Get(context.Background())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Клуб не найден"})
		return
	}

	var club ComputerClub
	doc.DataTo(&club)
	club.ID = doc.Ref.ID
	c.JSON(http.StatusOK, club)
}

// Создание клуба (требует аутентификации)
func createClub(c *gin.Context) {
	var club ComputerClub
	if err := c.ShouldBindJSON(&club); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if club.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID обязателен"})
		return
	}

	_, err := client.Collection("clubs").Doc(club.ID).Set(context.Background(), club)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Клуб добавлен", "id": club.ID})
}

// Обновление клуба (требует аутентификации)
func updateClub(c *gin.Context) {
	id := c.Param("id")
	var club ComputerClub
	if err := c.ShouldBindJSON(&club); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := client.Collection("clubs").Doc(id).Set(context.Background(), club)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Клуб обновлен"})
}

// Удаление клуба (требует аутентификации)
func deleteClub(c *gin.Context) {
	id := c.Param("id")
	_, err := client.Collection("clubs").Doc(id).Delete(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Клуб удален"})
}

// Авторизация пользователя
func authHandler(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No Authorization header"})
		return
	}

	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header"})
		return
	}

	decodedToken, err := firebaseAuth.VerifyIDToken(context.Background(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Успешный вход", "uid": decodedToken.UID})
}

func main() {
	initFirestore()
	defer client.Close()

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Открытые маршруты
	r.GET("/clubs", getAllClubs)
	r.GET("/clubs/:id", getClubByID)
	r.POST("/auth", authHandler)

	// Защищенные маршруты (только проверка аутентификации)
	r.POST("/clubs", AuthMiddleware(), createClub)
	r.PUT("/clubs/:id", AuthMiddleware(), updateClub)
	r.DELETE("/clubs/:id", AuthMiddleware(), deleteClub)

	r.Run(":8080")
}
