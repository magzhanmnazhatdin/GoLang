package main

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
)

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
	r.GET("/computers", getAllComputers)

	// Защищенные маршруты (только проверка аутентификации)
	r.POST("/clubs", AuthMiddleware(), createClub)
	r.PUT("/clubs/:id", AuthMiddleware(), updateClub)
	r.DELETE("/clubs/:id", AuthMiddleware(), deleteClub)

	// Маршруты для бронирований
	r.GET("/clubs/:id/computers", getClubComputers)
	r.GET("/bookings", AuthMiddleware(), getUserBookings)
	r.POST("/bookings", AuthMiddleware(), createBooking)
	r.PUT("/bookings/:id/cancel", AuthMiddleware(), cancelBooking)
	authRoutes := r.Group("/")
	authRoutes.Use(AuthMiddleware())
	{
		authRoutes.POST("/clubs/:id/computers", createComputerList)
	}

	r.Run(":8080")
}
