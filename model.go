package sunjumig

import (
	"time"

	"gorm.io/gorm"
)

type Migration struct {
	ID        int `gorm:"column:id"`
	Name      string
	Batch     int
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`

	Up   func(*gorm.DB) error `gorm:"-"`
	Down func(*gorm.DB) error `gorm:"-"`

	done bool `gorm:"-"`
}

type Migrator struct {
	db         *gorm.DB
	Migrations map[string]*Migration
	MaxBatch   int
	FilePath   string
}

type SchemaMigration struct {
	ID        int    `gorm:"column:id;primaryKey"`
	Name      string `gorm:"unique"`
	Batch     int
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (s SchemaMigration) Table() string {
	return "schema_migrations"
}
