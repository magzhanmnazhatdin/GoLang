// handlers.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
)

const CLUBID string = "ClubID"

func generateNewID() string {
	return client.Collection("bookings").NewDoc().ID
}

func getClubComputers(c *gin.Context) {
	clubID := c.Param("id")

	// Получаем список компьютеров для клуба
	docs, err := client.Collection("computers").
		Where("ClubID", "==", clubID).
		Documents(context.Background()).
		GetAll()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	computers := make([]Computer, 0)
	for _, doc := range docs {
		var comp Computer
		if err := doc.DataTo(&comp); err == nil {
			comp.ID = doc.Ref.ID
			computers = append(computers, comp)
		}
	}

	c.JSON(http.StatusOK, computers)
}

//func createBooking(c *gin.Context) {
//	uid := c.MustGet("uid").(string)
//
//	var booking struct {
//		ClubID    string    `json:"ClubID"`
//		PCNumber  int       `json:"Number"`
//		StartTime time.Time `json:"start_time"`
//		Hours     int       `json:"hours"`
//	}
//
//	if err := c.ShouldBindJSON(&booking); err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//		return
//	}
//
//	// Получаем информацию о клубе
//	clubDoc, err := client.Collection("clubs").Doc(booking.ClubID).Get(context.Background())
//	if err != nil {
//		c.JSON(http.StatusNotFound, gin.H{"error": "Клуб не найден"})
//		return
//	}
//
//	var club ComputerClub
//	clubDoc.DataTo(&club)
//
//	// Проверяем доступность компьютера
//	computerDocs, err := client.Collection("computers").
//		Where("ClubID", "==", booking.ClubID).
//		Where("Number", "==", booking.PCNumber).
//		Limit(1).
//		Documents(context.Background()).
//		GetAll()
//
//	if err != nil || len(computerDocs) == 0 {
//		c.JSON(http.StatusBadRequest, gin.H{"error": "Компьютер не найден"})
//		return
//	}
//
//	var computer Computer
//	computerDocs[0].DataTo(&computer)
//
//	if !computer.IsAvailable {
//		c.JSON(http.StatusBadRequest, gin.H{"error": "Компьютер уже занят"})
//		return
//	}
//
//	// Проверяем нет ли пересечений по времени
//	bookingsQuery := client.Collection("bookings").
//		Where("ClubID", "==", booking.ClubID).
//		Where("Number", "==", booking.PCNumber).
//		Where("status", "==", "active").
//		Where("end_time", ">", time.Now())
//
//	existingBookings, err := bookingsQuery.Documents(context.Background()).GetAll()
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//		return
//	}
//
//	if len(existingBookings) > 0 {
//		c.JSON(http.StatusBadRequest, gin.H{"error": "Компьютер уже забронирован на это время"})
//		return
//	}
//
//	// Создаем бронирование
//	endTime := booking.StartTime.Add(time.Duration(booking.Hours) * time.Hour)
//	totalPrice := club.PricePerHour * float64(booking.Hours)
//
//	newBooking := Booking{
//		ID:         generateNewID(), // Заменили firestore.NewDocID().ID на собственную функцию
//		ClubID:     booking.ClubID,  // Исправлено CLU на ClubID
//		UserID:     uid,
//		PCNumber:   booking.PCNumber,  // Исправлено PCM на PCNumber
//		StartTime:  booking.StartTime, // Исправлено Sta на StartTime
//		EndTime:    endTime,
//		TotalPrice: totalPrice,
//		Status:     "active",
//		CreatedAt:  time.Now(),
//	}
//
//	_, err = client.Collection("bookings").Doc(newBooking.ID).Set(context.Background(), newBooking)
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//		return
//	}
//
//	// Обновляем статус компьютера
//	_, err = client.Collection("computers").Doc(computer.ID).Update(context.Background(), []firestore.Update{
//		{Path: "IsAvailable", Value: false},
//	})
//
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//		return
//	}
//
//	c.JSON(http.StatusCreated, newBooking)
//}

func createBooking(c *gin.Context) {
	uid := c.MustGet("uid").(string)

	var booking struct {
		ClubID    string    `json:"ClubID"`    // Соответствует полю ClubID в Firestore
		PCNumber  int       `json:"PCNumber"`  // Номер компьютера
		StartTime time.Time `json:"StartTime"` // Время начала
		Hours     int       `json:"Hours"`     // Количество часов
	}

	if err := c.ShouldBindJSON(&booking); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Получаем информацию о клубе
	clubDoc, err := client.Collection("clubs").Where("ClubID", "==", booking.ClubID).Limit(1).Documents(context.Background()).Next()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Клуб не найден"})
		return
	}

	var club struct {
		Address      string  `firestore:"Address"`
		AvailablePCs int     `firestore:"AvailablePCs"`
		ClubID       string  `firestore:"ClubID"`
		ID           string  `firestore:"ID"`
		Name         string  `firestore:"Name"`
		PricePerHour float64 `firestore:"PricePerHour"`
	}

	if err := clubDoc.DataTo(&club); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем доступность компьютера
	computerDoc, err := client.Collection("computers").
		Where("ClubID", "==", booking.ClubID).
		Where("PCNumber", "==", booking.PCNumber).
		Limit(1).
		Documents(context.Background()).
		Next()

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Компьютер не найден"})
		return
	}

	var computer struct {
		IsAvailable bool `firestore:"IsAvailable"`
	}

	if err := computerDoc.DataTo(&computer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !computer.IsAvailable {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Компьютер уже занят"})
		return
	}

	// Проверяем пересечения по времени
	bookingsQuery := client.Collection("bookings").
		Where("ClubID", "==", booking.ClubID).
		Where("PCNumber", "==", booking.PCNumber).
		Where("Status", "==", "active").
		Where("EndTime", ">", time.Now())

	existing, err := bookingsQuery.Documents(context.Background()).GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(existing) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Компьютер уже забронирован на это время"})
		return
	}

	// Создаем бронирование
	endTime := booking.StartTime.Add(time.Duration(booking.Hours) * time.Hour)
	totalPrice := club.PricePerHour * float64(booking.Hours)

	newBooking := map[string]interface{}{
		"ID":         generateNewID(),
		"ClubID":     booking.ClubID,
		"UserID":     uid,
		"PCNumber":   booking.PCNumber,
		"StartTime":  booking.StartTime,
		"EndTime":    endTime,
		"TotalPrice": totalPrice,
		"Status":     "active",
		"CreatedAt":  time.Now(),
	}

	_, err = client.Collection("bookings").Doc(newBooking["ID"].(string)).Set(context.Background(), newBooking)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Обновляем статус компьютера
	_, err = computerDoc.Ref.Update(context.Background(), []firestore.Update{
		{Path: "IsAvailable", Value: false},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, newBooking)
}

// handlers.go
func createComputerList(c *gin.Context) {
	clubID := c.Param("clubId")

	var computers []Computer
	if err := c.ShouldBindJSON(&computers); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Создаем пакетную запись в Firestore
	batch := client.Batch()
	computersCollection := client.Collection("computers")

	for _, computer := range computers {
		computer.ClubID = clubID
		docRef := computersCollection.NewDoc()
		batch.Set(docRef, computer)
	}

	// Применяем пакетную запись
	_, err := batch.Commit(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": fmt.Sprintf("Добавлено %d компьютеров", len(computers)),
		"clubId":  clubID,
	})
}

// handlers.go
func getUserBookings(c *gin.Context) {
	uid := c.MustGet("uid").(string)

	// Получаем активные бронирования пользователя
	docs, err := client.Collection("bookings").
		Where("user_id", "==", uid).
		Where("status", "==", "active").
		Where("end_time", ">", time.Now()).
		OrderBy("start_time", firestore.Asc).
		Documents(context.Background()).
		GetAll()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var bookings []Booking
	for _, doc := range docs {
		var booking Booking
		doc.DataTo(&booking)
		booking.ID = doc.Ref.ID

		// Получаем информацию о клубе
		clubDoc, err := client.Collection("clubs").Doc(booking.ClubID).Get(context.Background())
		if err == nil {
			var club ComputerClub
			clubDoc.DataTo(&club)
			booking.ClubName = club.Name // Добавляем имя клуба в ответ
		}

		bookings = append(bookings, booking)
	}

	c.JSON(http.StatusOK, bookings)
}

// handlers.go
func cancelBooking(c *gin.Context) {
	uid := c.MustGet("uid").(string)
	bookingID := c.Param("id")

	// Получаем бронирование
	bookingRef := client.Collection("bookings").Doc(bookingID)
	bookingDoc, err := bookingRef.Get(context.Background())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Бронирование не найдено"})
		return
	}

	var booking Booking
	bookingDoc.DataTo(&booking)

	// Проверяем, что бронирование принадлежит пользователю
	if booking.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "Нельзя отменить чужое бронирование"})
		return
	}

	// Проверяем, что бронирование еще активно
	if booking.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Бронирование уже отменено или завершено"})
		return
	}

	// Проверяем, что не слишком поздно отменять (минимум 1 час до начала)
	if time.Until(booking.StartTime) < time.Hour {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Можно отменить только за час до начала"})
		return
	}

	// Обновляем статус бронирования
	_, err = bookingRef.Update(context.Background(), []firestore.Update{
		{Path: "status", Value: "cancelled"},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Освобождаем компьютер
	computerDocs, err := client.Collection("computers").
		Where("club_id", "==", booking.ClubID).
		Where("number", "==", booking.PCNumber).
		Limit(1).
		Documents(context.Background()).
		GetAll()

	if err != nil {
		log.Printf("Ошибка поиска компьютера: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при освобождении компьютера"})
		return
	}

	if len(computerDocs) > 0 {
		_, err = computerDocs[0].Ref.Update(context.Background(), []firestore.Update{
			{Path: "is_available", Value: true},
		})

		if err != nil {
			log.Printf("Ошибка обновления статуса компьютера: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении статуса компьютера"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Бронирование успешно отменено"})
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

func getAllComputers(c *gin.Context) {
	var comps []Computer
	docs, err := client.Collection("computers").Documents(context.Background()).GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, doc := range docs {
		var comp Computer
		doc.DataTo(&comp)
		comp.ID = doc.Ref.ID
		comps = append(comps, comp)
	}

	c.JSON(http.StatusOK, comps)
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
