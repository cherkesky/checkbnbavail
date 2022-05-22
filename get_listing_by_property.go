package main

import (
	"fmt"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// gorm.Model definition
type Model struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Listing struct {
	ID                 uint
	PlatformId         int
	PlatformListingId  string
	ListingNickname    string
	PlatformPropertyId string
}

var listings []Listing

func getListingByProperty(properties []string) ([]string, error) {
	availProperties := []string{}
	host := os.Getenv("HOST")
	user := os.Getenv("USER")
	password := os.Getenv("PASSWORD")
	dbname := os.Getenv("DBNAME")
	port := os.Getenv("PORT")
	sslmode := os.Getenv("SSLMODE")
	timeZone := os.Getenv("TIMEZOME")

	db, err := gorm.Open(postgres.Open(fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s timezone=%s", host, port, user, dbname, password, sslmode, timeZone)), &gorm.Config{})
	if err != nil {
		panic(err.Error())
	}

	// Get all records
	for p := range properties {
		fmt.Println(properties[p])

		db.Find(&listings, "platform_property_id = ?", properties[p])
		availProperties = append(availProperties, listings[0].ListingNickname)
	}

	fmt.Println(availProperties)
	return availProperties, nil

}
