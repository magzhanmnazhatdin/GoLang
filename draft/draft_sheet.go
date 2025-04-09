package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
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

// JWT ключ
var jwtKey = []byte("secret")

// Структура JWT Claims
type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Генерация JWT токена
func GenerateJWT(username, role string) (string, error) {
	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// Глобальные переменные для Firebase
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

//func AuthMiddleware(client *auth.Client) gin.HandlerFunc {
//	return func(c *gin.Context) {
//		// Получаем токен из заголовка
//		authHeader := c.GetHeader("Authorization")
//		if authHeader == "" {
//			c.JSON(http.StatusUnauthorized, gin.H{"error": "No Authorization header"})
//			c.Abort()
//			return
//		}
//
//		// Проверяем Bearer-токен
//		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
//		token, err := client.VerifyIDToken(context.Background(), tokenString)
//		if err != nil {
//			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
//			c.Abort()
//			return
//		}
//
//		// Добавляем данные пользователя в контекст
//		c.Set("user", token)
//		c.Next()
//	}
//}

// Middleware для проверки аутентификации и роли
func AuthMiddleware(requiredRole string) gin.HandlerFunc {
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

		// Извлекаем роль пользователя (например, из custom claims Firebase)
		role, ok := decodedToken.Claims["role"].(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "No role assigned"})
			c.Abort()
			return
		}

		// Проверяем соответствие роли
		if requiredRole != "" && role != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient permissions"})
			c.Abort()
			return
		}

		// Добавляем пользователя в контекст
		c.Set("uid", decodedToken.UID)
		c.Set("role", role)
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

// Создание клуба (только для админов)
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

// Обновление клуба (только для админов)
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

// Удаление клуба (только для админов)
func deleteClub(c *gin.Context) {
	id := c.Param("id")
	_, err := client.Collection("clubs").Doc(id).Delete(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Клуб удален"})
}

// Авторизация пользователя и выдача токена
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

	role, _ := decodedToken.Claims["role"].(string)
	jwtToken, _ := GenerateJWT(decodedToken.UID, role)

	c.JSON(http.StatusOK, gin.H{"message": "Успешный вход", "jwt": jwtToken})
}

func getUserRole(uid string) (string, error) {
	doc, err := client.Collection("users").Doc(uid).Get(context.Background())
	if err != nil {
		return "", err
	}

	role, ok := doc.Data()["role"].(string)
	if !ok {
		return "", fmt.Errorf("роль не найдена")
	}
	return role, nil
}

func setUserRole(c *gin.Context) {
	var data struct {
		UID  string `json:"uid"`
		Role string `json:"role"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	_, err := client.Collection("users").Doc(data.UID).Set(context.Background(), map[string]interface{}{
		"role": data.Role,
	}, firestore.MergeAll)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления роли"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Роль обновлена"})
}

func RoleMiddleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Нет токена"})
			c.Abort()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Некорректный токен"})
			c.Abort()
			return
		}

		decodedToken, err := firebaseAuth.VerifyIDToken(context.Background(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка проверки токена"})
			c.Abort()
			return
		}

		role, err := getUserRole(decodedToken.UID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Ошибка получения роли"})
			c.Abort()
			return
		}

		if role != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Доступ запрещен"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func main() {
	initFirestore()
	defer client.Close()

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Или укажи свой домен
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Открытые маршруты
	r.GET("/clubs", getAllClubs)
	r.GET("/clubs/:id", getClubByID)
	r.POST("/auth", authHandler)

	// Защищенные маршруты (только для админов)
	r.POST("/clubs", AuthMiddleware("admin"), createClub)
	r.PUT("/clubs/:id", AuthMiddleware("admin"), updateClub)
	r.DELETE("/clubs/:id", AuthMiddleware("admin"), deleteClub)
	r.POST("/setRole", RoleMiddleware("admin"), setUserRole)

	r.Run(":8080")
}
