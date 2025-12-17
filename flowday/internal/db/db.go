package db

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB;


func Init() {
	database , err := gorm.Open(sqlite.Open("flowday.db") , &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database")
	}

	DB = database;
	log.Println("Connected to database")
}