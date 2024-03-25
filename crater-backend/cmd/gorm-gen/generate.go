// Generate code for all tables in the connected database
package main

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gorm"
)

func main() {
	g := gen.NewGenerator(gen.Config{
		OutPath: "../../pkg/query",
		Mode:    gen.WithDefaultQuery | gen.WithQueryInterface,
	})

	// Connect to the database
	password := os.Getenv("PGPASSWORD")
	port := os.Getenv("PGPORT")
	if password == "" || port == "" {
		panic("Please read the README.md file to set the environment variable.")
	}
	dsnPattern := "host=localhost user=postgres password=%s dbname=crater port=%s sslmode=require TimeZone=Asia/Shanghai"
	dsn := fmt.Sprintf(dsnPattern, password, port)
	db, err := gorm.Open(postgres.Open(dsn))
	if err != nil {
		panic(fmt.Errorf("connect to postgres: %w", err))
	}
	g.UseDB(db)

	// Generate code for all tables in the connected database
	g.ApplyBasic(g.GenerateAllTable()...)

	// Execute the code generation
	g.Execute()
}
