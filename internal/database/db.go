package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// DB là kết nối tới DB riêng của Go Studio (bảng jobs).
var DB *sql.DB

// AuthDB là kết nối tới DB Go Keep (bảng users) — chỉ mở khi AUTH_ENABLED=true.
// Go Studio dùng chung kho tài khoản với Go Keep qua kết nối này.
var AuthDB *sql.DB

func Connect() error {
	host := getEnv("DB_HOST", "127.0.0.1")
	port := getEnv("DB_PORT", "3306")
	user := getEnv("DB_USER", "root")
	pass := getEnv("DB_PASS", "")
	name := getEnv("DB_NAME", "gostudio")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4", user, pass, host, port, name)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	if err = db.Ping(); err != nil {
		return err
	}
	DB = db
	log.Println("Connected to MySQL database:", name)
	return nil
}

// ConnectAuth mở kết nối tới DB Go Keep để dùng chung tài khoản.
// DSN đầy đủ lấy từ biến môi trường GOKEEP_DB_DSN.
func ConnectAuth() error {
	dsn := os.Getenv("GOKEEP_DB_DSN")
	if dsn == "" {
		return fmt.Errorf("GOKEEP_DB_DSN chưa được đặt (bắt buộc khi AUTH_ENABLED=true)")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	if err = db.Ping(); err != nil {
		return err
	}
	AuthDB = db
	log.Println("Connected to GoKeep account database (auth)")
	return nil
}

func Migrate() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			type       VARCHAR(50)  NOT NULL,
			status     VARCHAR(20)  NOT NULL DEFAULT 'pending',
			input_file VARCHAR(512) NOT NULL,
			output_file VARCHAR(512),
			error_msg  TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`)
	if err != nil {
		return err
	}

	// MySQL 8.4 không hỗ trợ ADD COLUMN IF NOT EXISTS → kiểm tra trước qua information_schema.
	if err := addColumnIfMissing("display_name", `ALTER TABLE jobs ADD COLUMN display_name VARCHAR(255) DEFAULT NULL`); err != nil {
		return err
	}
	if err := addColumnIfMissing("user_id", `ALTER TABLE jobs ADD COLUMN user_id BIGINT UNSIGNED DEFAULT NULL, ADD INDEX idx_jobs_user (user_id)`); err != nil {
		return err
	}
	return nil
}

func addColumnIfMissing(column, alterSQL string) error {
	var exists int
	err := DB.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = 'jobs' AND column_name = ?
	`, column).Scan(&exists)
	if err != nil {
		return err
	}
	if exists == 0 {
		if _, err := DB.Exec(alterSQL); err != nil {
			return err
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
