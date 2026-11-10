// cdig-sync is an optional administrative command; --apply is always explicit.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/integrations/cdig"
	mysqlconfig "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	apply := flag.Bool("apply", false, "Commit synchronization (default is read-only simulation)")
	dry := flag.Bool("dry-run", false, "Read-only simulation (also the default)")
	flag.Parse()
	if *apply && *dry {
		return fmt.Errorf("choose either --apply or --dry-run")
	}
	_ = godotenv.Load(".env")
	c, reason := cdig.Load()
	if !c.Enabled {
		b, _ := json.Marshal(map[string]any{"enabled": false, "reason": reason})
		fmt.Println(string(b))
		return nil
	}
	if expected := os.Getenv("EXPECTED_DATABASE_NAME"); expected != "" && expected != os.Getenv("DB_NAME") {
		return fmt.Errorf("database does not match this deployment")
	}
	dsn := mysqlconfig.NewConfig()
	dsn.User = os.Getenv("DB_USER")
	dsn.Passwd = os.Getenv("DB_PASSWORD")
	dsn.Net = "tcp"
	dsn.Addr = os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT")
	dsn.DBName = os.Getenv("DB_NAME")
	dsn.ParseTime = true
	dsn.Params = map[string]string{"charset": "utf8mb4"}
	db, err := gorm.Open(mysql.Open(dsn.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return fmt.Errorf("cannot connect to destination database")
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	report, err := (&cdig.Sync{Config: c, Client: cdig.NewClient(c), DB: db}).Run(ctx, !*apply)
	b, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(b))
	return err
}
