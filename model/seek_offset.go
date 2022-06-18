package model

import "gorm.io/gorm"

type SeekOffset struct {
	gorm.Model
	Num  int
	From int
	To   int
}
