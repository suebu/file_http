package common

import (
	"go_demo/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("ceshi.db"), &gorm.Config{})

	if err != nil {
		panic("failed to connect database")
	}

	// 迁移 schema
	db.AutoMigrate(&model.SeekOffset{})
	return db
}
