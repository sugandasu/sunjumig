package migrator

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sort"
	"text/template"
	"time"

	"gorm.io/gorm"
)

func Init(db *gorm.DB, migrations []Migration) (*Migrator, error) {
	var migrator = &Migrator{
		Migrations: map[string]*Migration{},
	}

	migrator.db = db

	for _, migration := range migrations {
		migrator.Migrations[migration.Name] = &migration
	}

	db.AutoMigrate(&SchemaMigration{})

	rows := []Migration{}

	result := db.Table(SchemaMigration{}.Table()).Find(&rows)
	if result.Error != nil {
		println("error get schema_migrations: " + result.Error.Error())
		return migrator, result.Error
	}

	for _, row := range rows {
		if migrator.Migrations[row.Name] != nil {
			migrator.Migrations[row.Name].ID = row.ID
			migrator.Migrations[row.Name].Name = row.Name
			migrator.Migrations[row.Name].Batch = row.Batch
			migrator.Migrations[row.Name].done = true
		}

		if migrator.MaxBatch < row.Batch {
			migrator.MaxBatch = row.Batch
		}
	}

	return migrator, nil
}

func Create(path, name string) error {
	version := time.Now().Format("20060102150405")

	in := struct {
		Version string
		Name    string
	}{
		Version: version,
		Name:    name,
	}

	var out bytes.Buffer

	t := template.Must(template.ParseFiles("./template.txt"))
	if err := t.Execute(&out, in); err != nil {
		return errors.New("Unable to execute template: " + err.Error())
	}

	f, err := os.Create(fmt.Sprintf("%s/%s_%s.go", path, version, name))

	if err != nil {
		return errors.New("Unable to create migration file: " + err.Error())
	}

	defer f.Close()

	if _, err := f.WriteString(out.String()); err != nil {
		return errors.New("Unable to write to migration file: " + err.Error())
	}

	fmt.Println("Generated new migration file...", f.Name())
	return nil
}

func (m *Migrator) AddMigration(mg *Migration) {
	m.Migrations[mg.Name] = mg
}

func (m *Migrator) Up() error {
	tx := m.db.Begin()

	keys := []string{}
	for key := range m.Migrations {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		mg := m.Migrations[key]
		if mg.done {
			continue
		}

		fmt.Println("Running migration", mg.Name)
		if err := mg.Up(tx); err != nil {
			tx.Rollback()
			return err
		}

		mg.Batch = m.MaxBatch + 1
		if err := tx.Table(SchemaMigration{}.Table()).Create(&mg).Error; err != nil {
			tx.Rollback()
			return err
		}

		fmt.Println("Finished running migration", mg.Name)
	}

	tx.Commit()

	return nil
}

func (m *Migrator) Down() error {
	rows := []Migration{}

	err := m.db.Table(SchemaMigration{}.Table()).Where("batch = ?", m.MaxBatch).
		Order("name DESC").
		Find(&rows).
		Error
	if err != nil {
		return err
	}

	tx := m.db.Begin()

	for _, row := range rows {
		mg := m.Migrations[row.Name]
		if !mg.done || mg.Batch != m.MaxBatch {
			continue
		}

		fmt.Println("Reverting migration", mg.Name)
		if err := mg.Down(tx); err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Table(SchemaMigration{}.Table()).Delete(&row).Error; err != nil {
			tx.Rollback()
			return err
		}

		fmt.Println("Finished reverting migration", mg.Name)
	}

	tx.Commit()

	return nil
}

// MigrationStatus .
func (m *Migrator) MigrationStatus() error {
	for _, mg := range m.Migrations {
		if mg.done {
			fmt.Printf("%s", fmt.Sprintf("Migration %s... completed", mg.Name))
		} else {
			fmt.Printf("%s", fmt.Sprintf("Migration %s... pending", mg.Name))
		}
	}

	return nil
}
