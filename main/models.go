package main

import (
	"time"
)

// Структура клуба
type ComputerClub struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Address      string  `json:"address"`
	PricePerHour float64 `json:"price_per_hour"`
	AvailablePCs int     `json:"available_pcs"`
}

// Модель бронирования
type Booking struct {
	ID         string    `json:"id"`
	ClubID     string    `json:"club_id"`
	ClubName   string    `json:"club_name,omitempty"`
	UserID     string    `json:"user_id"`
	PCNumber   int       `json:"pc_number"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	TotalPrice float64   `json:"total_price"`
	Status     string    `json:"status"` // "active", "cancelled", "completed"
	CreatedAt  time.Time `json:"created_at"`
}

// Модель компьютера в клубе
type Computer struct {
	ID          string `json:"id"`
	ClubID      string `json:"club_id"`
	Number      int    `json:"number"`
	Description string `json:"description"`
	IsAvailable bool   `json:"is_available"`
}
