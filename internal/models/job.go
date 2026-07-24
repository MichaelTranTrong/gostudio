package models

import (
	"database/sql"
	"gostudio/internal/database"
	"os"
	"time"
)

type Job struct {
	ID          uint64    `json:"id"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	InputFile   string    `json:"input_file"`
	OutputFile  string    `json:"output_file"`
	DisplayName string    `json:"display_name,omitempty"`
	ErrorMsg    string    `json:"error_msg,omitempty"`
	UserID      uint64    `json:"-"` // 0 = không gắn user (chế độ local, AUTH tắt)
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateJob tạo job mới. userID = 0 nghĩa là không gắn với user nào
// (chế độ local khi AUTH_ENABLED tắt) → cột user_id để NULL.
func CreateJob(jobType, inputFile string, userID uint64) (uint64, error) {
	var uid interface{}
	if userID != 0 {
		uid = userID
	}
	res, err := database.DB.Exec(
		`INSERT INTO jobs (type, status, input_file, user_id) VALUES (?, 'pending', ?, ?)`,
		jobType, inputFile, uid,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return uint64(id), nil
}

func UpdateJob(id uint64, status, outputFile, errMsg string) error {
	_, err := database.DB.Exec(
		`UPDATE jobs SET status=?, output_file=?, error_msg=? WHERE id=?`,
		status, outputFile, errMsg, id,
	)
	return err
}

func GetJob(id uint64) (*Job, error) {
	row := database.DB.QueryRow(
		`SELECT id, type, status, input_file, COALESCE(output_file,''), COALESCE(display_name,''), COALESCE(error_msg,''), COALESCE(user_id,0), created_at, updated_at FROM jobs WHERE id=?`, id,
	)
	j := &Job{}
	return j, row.Scan(&j.ID, &j.Type, &j.Status, &j.InputFile, &j.OutputFile, &j.DisplayName, &j.ErrorMsg, &j.UserID, &j.CreatedAt, &j.UpdatedAt)
}

// GetJobOwned trả về job chỉ khi userID == 0 (local) hoặc job thuộc về userID đó.
// Ngược lại trả về sql.ErrNoRows (coi như không tồn tại) để không lộ file người khác.
func GetJobOwned(id, userID uint64) (*Job, error) {
	j, err := GetJob(id)
	if err != nil {
		return nil, err
	}
	if userID != 0 && j.UserID != userID {
		return nil, sql.ErrNoRows
	}
	return j, nil
}

// RenameJob đặt tên hiển thị tùy chỉnh cho job (không đổi tên file thật trên disk).
func RenameJob(id uint64, name string, userID uint64) error {
	var res sql.Result
	var err error
	if userID == 0 {
		res, err = database.DB.Exec(`UPDATE jobs SET display_name=? WHERE id=?`, name, id)
	} else {
		res, err = database.DB.Exec(`UPDATE jobs SET display_name=? WHERE id=? AND user_id=?`, name, id, userID)
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func DeleteJob(id, userID uint64) error {
	var inputFile, outputFile string
	var row *sql.Row
	if userID == 0 {
		row = database.DB.QueryRow(`SELECT input_file, COALESCE(output_file,'') FROM jobs WHERE id=?`, id)
	} else {
		row = database.DB.QueryRow(`SELECT input_file, COALESCE(output_file,'') FROM jobs WHERE id=? AND user_id=?`, id, userID)
	}
	if err := row.Scan(&inputFile, &outputFile); err != nil {
		return err // ErrNoRows → không sở hữu / không tồn tại
	}

	if userID == 0 {
		_, _ = database.DB.Exec(`DELETE FROM jobs WHERE id=?`, id)
	} else {
		_, _ = database.DB.Exec(`DELETE FROM jobs WHERE id=? AND user_id=?`, id, userID)
	}

	if inputFile != "" {
		os.Remove(inputFile)
	}
	if outputFile != "" {
		os.Remove(outputFile)
	}
	return nil
}

func DeleteAllJobs(userID uint64) error {
	var rows *sql.Rows
	var err error
	if userID == 0 {
		rows, err = database.DB.Query(`SELECT input_file, COALESCE(output_file,'') FROM jobs`)
	} else {
		rows, err = database.DB.Query(`SELECT input_file, COALESCE(output_file,'') FROM jobs WHERE user_id=?`, userID)
	}
	if err != nil {
		return err
	}
	var files [][2]string
	for rows.Next() {
		var in, out string
		if rows.Scan(&in, &out) == nil {
			files = append(files, [2]string{in, out})
		}
	}
	rows.Close()

	if userID == 0 {
		if _, err = database.DB.Exec(`DELETE FROM jobs`); err != nil {
			return err
		}
		// Bảng riêng của Go Studio (local) → reset ID về 1.
		if _, err = database.DB.Exec(`ALTER TABLE jobs AUTO_INCREMENT = 1`); err != nil {
			return err
		}
	} else {
		// Bảng dùng chung nhiều user → chỉ xóa job của user này, không reset AUTO_INCREMENT.
		if _, err = database.DB.Exec(`DELETE FROM jobs WHERE user_id=?`, userID); err != nil {
			return err
		}
	}

	for _, f := range files {
		if f[0] != "" {
			os.Remove(f[0])
		}
		if f[1] != "" {
			os.Remove(f[1])
		}
	}
	return nil
}

func ListJobs(userID uint64) ([]Job, error) {
	const cols = `SELECT id, type, status, input_file, COALESCE(output_file,''), COALESCE(display_name,''), COALESCE(error_msg,''), COALESCE(user_id,0), created_at, updated_at FROM jobs `
	var rows *sql.Rows
	var err error
	if userID == 0 {
		rows, err = database.DB.Query(cols + `ORDER BY id DESC LIMIT 100`)
	} else {
		rows, err = database.DB.Query(cols+`WHERE user_id=? ORDER BY id DESC LIMIT 100`, userID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.Type, &j.Status, &j.InputFile, &j.OutputFile, &j.DisplayName, &j.ErrorMsg, &j.UserID, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
