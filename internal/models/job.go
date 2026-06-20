package models

import (
	"gostudio/internal/database"
	"time"
)

type Job struct {
	ID         uint64    `json:"id"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`
	InputFile  string    `json:"input_file"`
	OutputFile string    `json:"output_file"`
	ErrorMsg   string    `json:"error_msg,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func CreateJob(jobType, inputFile string) (uint64, error) {
	res, err := database.DB.Exec(
		`INSERT INTO jobs (type, status, input_file) VALUES (?, 'pending', ?)`,
		jobType, inputFile,
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
		`SELECT id, type, status, input_file, COALESCE(output_file,''), COALESCE(error_msg,''), created_at, updated_at FROM jobs WHERE id=?`, id,
	)
	j := &Job{}
	return j, row.Scan(&j.ID, &j.Type, &j.Status, &j.InputFile, &j.OutputFile, &j.ErrorMsg, &j.CreatedAt, &j.UpdatedAt)
}

func ListJobs() ([]Job, error) {
	rows, err := database.DB.Query(
		`SELECT id, type, status, input_file, COALESCE(output_file,''), COALESCE(error_msg,''), created_at, updated_at FROM jobs ORDER BY id DESC LIMIT 100`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.Type, &j.Status, &j.InputFile, &j.OutputFile, &j.ErrorMsg, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
