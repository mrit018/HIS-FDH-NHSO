package config

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/joho/godotenv"
)

// InitDB ฟังก์ชันสำหรับเริ่มต้นการเชื่อมต่อฐานข้อมูล
func InitDB() *gorm.DB {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		dbType = "mysql" // ค่าเริ่มต้นเป็น MySQL
	}

	var dialector gorm.Dialector
	var dsn string

	switch dbType {
	case "postgres", "postgresql":
		// รับค่าการเชื่อมต่อ PostgreSQL จาก environment variables
		host := os.Getenv("POSTGRES_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("POSTGRES_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("POSTGRES_USER")
		if user == "" {
			user = "postgres"
		}
		password := os.Getenv("POSTGRES_PASSWORD")
		dbname := os.Getenv("POSTGRES_DATABASE")
		if dbname == "" {
			dbname = "nhso_claim"
		}
		sslmode := os.Getenv("POSTGRES_SSLMODE")
		if sslmode == "" {
			sslmode = "disable"
		}

		// สร้าง DSN สำหรับ PostgreSQL - เพิ่ม client_encoding=UTF8
		dsn = "host=" + host + " user=" + user + " password=" + password +
			" dbname=" + dbname + " port=" + port + " sslmode=" + sslmode +
			" TimeZone=Asia/Bangkok client_encoding=UTF8"
		dialector = postgres.Open(dsn)

	case "mysql":
		fallthrough
	default:
		// รับค่าการเชื่อมต่อ MySQL จาก environment variables
		host := os.Getenv("MYSQL_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("MYSQL_PORT")
		if port == "" {
			port = "3306"
		}
		user := os.Getenv("MYSQL_USER")
		if user == "" {
			user = "root"
		}
		password := os.Getenv("MYSQL_PASSWORD")
		dbname := os.Getenv("MYSQL_DATABASE")
		if dbname == "" {
			dbname = "nhso_claim"
		}

		// สร้าง DSN สำหรับ MySQL
		dsn = user + ":" + password + "@tcp(" + host + ":" + port + ")/" + dbname +
			"?charset=utf8mb4&parseTime=True&loc=Local"
		dialector = mysql.Open(dsn)
	}

	// เปิดการเชื่อมต่อฐานข้อมูล
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// ตั้งค่า connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Printf("Database connection established successfully (%s)", dbType)
	return db
}
