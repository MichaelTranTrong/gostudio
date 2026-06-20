package models

import (
	"gostudio/internal/database"
	"os"
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

func DeleteJob(id uint64) error {
	var inputFile, outputFile string
	row := database.DB.QueryRow(`SELECT input_file, COALESCE(output_file,'') FROM jobs WHERE id=?`, id)
	_ = row.Scan(&inputFile, &outputFile)

	_, err := database.DB.Exec(`DELETE FROM jobs WHERE id=?`, id)
	if err != nil {
		return err
	}

	if inputFile != "" {
		os.Remove(inputFile)
	}
	if outputFile != "" {
		os.Remove(outputFile)
	}
	return nil
}

func DeleteAllJobs() error {
	rows, err := database.DB.Query(`SELECT input_file, COALESCE(output_file,'') FROM jobs`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var files [][2]string
	for rows.Next() {
		var in, out string
		if rows.Scan(&in, &out) == nil {
			files = append(files, [2]string{in, out})
		}
	}

	_, err = database.DB.Exec(`DELETE FROM jobs`)
	if err != nil {
		return err
	}
	_, err = database.DB.Exec(`ALTER TABLE jobs AUTO_INCREMENT = 1`)
	if err != nil {
		return err
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
