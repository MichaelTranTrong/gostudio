package auth

import (
	"database/sql"
	"strings"

	"gostudio/internal/database"
)

// User khớp với bảng users của Go Keep (chỉ các cột Go Studio cần).
type User struct {
	ID            uint64 `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Role          string `json:"role"`
	IsLocked      bool   `json:"is_locked"`
	EmailVerified bool   `json:"email_verified"`
}

const userCols = `SELECT id, email, name, password_hash, role, is_locked, email_verified FROM users `

func scanUser(row *sql.Row) (*User, string, error) {
	var u User
	var hash string
	err := row.Scan(&u.ID, &u.Email, &u.Name, &hash, &u.Role, &u.IsLocked, &u.EmailVerified)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return &u, hash, nil
}

// GetUserByEmail trả về user + password hash (rỗng nếu không có).
func GetUserByEmail(email string) (*User, string, error) {
	return scanUser(database.AuthDB.QueryRow(userCols+`WHERE email = ?`, strings.ToLower(strings.TrimSpace(email))))
}

func GetUserByID(id uint64) (*User, string, error) {
	return scanUser(database.AuthDB.QueryRow(userCols+`WHERE id = ?`, id))
}

// CreateUser tạo tài khoản mới trong bảng users của Go Keep (email_verified mặc định 0).
func CreateUser(email, name, passwordHash string) (uint64, error) {
	res, err := database.AuthDB.Exec(
		`INSERT INTO users (email, name, password_hash) VALUES (?, ?, ?)`,
		strings.ToLower(strings.TrimSpace(email)), name, passwordHash,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return uint64(id), nil
}

func SetVerifyToken(userID uint64, token string) error {
	_, err := database.AuthDB.Exec(`UPDATE users SET verify_token = ? WHERE id = ?`, token, userID)
	return err
}

// VerifyEmailByToken đánh dấu email đã xác thực dựa vào verify_token.
func VerifyEmailByToken(token string) error {
	res, err := database.AuthDB.Exec(`UPDATE users SET email_verified = 1, verify_token = NULL WHERE verify_token = ?`, token)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
