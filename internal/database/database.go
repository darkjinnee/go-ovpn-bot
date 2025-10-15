package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Limit    int    `json:"limit"`
	Configs  []Config `json:"configs"`
}

type Config struct {
	ID       int64  `json:"id"`
	UserID   int64  `json:"user_id"`
	Name     string `json:"name"`
	FilePath string `json:"file_path"`
}

type ActivationCode struct {
	ID     int64  `json:"id"`
	Code   string `json:"code"`
	Status string `json:"status"` // "active", "used"
	Limit  int    `json:"limit"`
}

func New(dbPath string) (*DB, error) {
	// Создаем директорию для базы данных если она не существует
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			telegram_id INTEGER UNIQUE NOT NULL,
			username TEXT,
			limit_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			file_path TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS activation_codes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT UNIQUE NOT NULL,
			status TEXT DEFAULT 'active',
			limit_count INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_configs_user_id ON configs (user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_activation_codes_code ON activation_codes (code)`,
	}

	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}
	}

	return nil
}

func (db *DB) GetOrCreateUser(telegramID int64, username string) (*User, error) {
	// Сначала пытаемся найти пользователя
	var userID int64
	var dbUsername sql.NullString
	var limit int
	
	err := db.conn.QueryRow(
		"SELECT id, username, limit_count FROM users WHERE telegram_id = ?",
		telegramID,
	).Scan(&userID, &dbUsername, &limit)
	
	if err == sql.ErrNoRows {
		// Пользователь не найден, создаем нового
		result, err := db.conn.Exec(
			"INSERT INTO users (telegram_id, username, limit_count) VALUES (?, ?, 0)",
			telegramID, username,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
		
		userID, err = result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get user ID: %w", err)
		}
		
		return &User{
			ID:       userID,
			Username: username,
			Limit:    0,
			Configs:  []Config{},
		}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Пользователь найден, получаем его конфиги
	configs, err := db.GetUserConfigs(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user configs: %w", err)
	}

	usernameStr := ""
	if dbUsername.Valid {
		usernameStr = dbUsername.String
	}

	return &User{
		ID:       userID,
		Username: usernameStr,
		Limit:    limit,
		Configs:  configs,
	}, nil
}

func (db *DB) GetUserConfigs(userID int64) ([]Config, error) {
	rows, err := db.conn.Query(
		"SELECT id, name, file_path FROM configs WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query configs: %w", err)
	}
	defer rows.Close()

	var configs []Config
	for rows.Next() {
		var config Config
		if err := rows.Scan(&config.ID, &config.Name, &config.FilePath); err != nil {
			return nil, fmt.Errorf("failed to scan config: %w", err)
		}
		config.UserID = userID
		configs = append(configs, config)
	}

	return configs, nil
}

func (db *DB) CreateConfig(userID int64, name, filePath string) (*Config, error) {
	result, err := db.conn.Exec(
		"INSERT INTO configs (user_id, name, file_path) VALUES (?, ?, ?)",
		userID, name, filePath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	configID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get config ID: %w", err)
	}

	return &Config{
		ID:       configID,
		UserID:   userID,
		Name:     name,
		FilePath: filePath,
	}, nil
}

func (db *DB) DeleteConfig(configID int64) error {
	_, err := db.conn.Exec("DELETE FROM configs WHERE id = ?", configID)
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}
	return nil
}

func (db *DB) GetConfigByID(configID int64) (*Config, error) {
	var config Config
	err := db.conn.QueryRow(
		"SELECT id, user_id, name, file_path FROM configs WHERE id = ?",
		configID,
	).Scan(&config.ID, &config.UserID, &config.Name, &config.FilePath)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("config not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to query config: %w", err)
	}

	return &config, nil
}

// GetActivationCodeByCode получает код активации по коду
func (db *DB) GetActivationCodeByCode(code string) (*ActivationCode, error) {
	var activationCode ActivationCode
	err := db.conn.QueryRow(
		"SELECT id, code, status, limit_count FROM activation_codes WHERE code = ?",
		code,
	).Scan(&activationCode.ID, &activationCode.Code, &activationCode.Status, &activationCode.Limit)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("activation code not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to query activation code: %w", err)
	}

	return &activationCode, nil
}

// UseActivationCode помечает код как использованный
func (db *DB) UseActivationCode(codeID int64) error {
	_, err := db.conn.Exec(
		"UPDATE activation_codes SET status = 'used' WHERE id = ?",
		codeID,
	)
	if err != nil {
		return fmt.Errorf("failed to use activation code: %w", err)
	}
	return nil
}

// UpdateUserLimit обновляет лимит пользователя
func (db *DB) UpdateUserLimit(userID int64, newLimit int) error {
	_, err := db.conn.Exec(
		"UPDATE users SET limit_count = ? WHERE id = ?",
		newLimit, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user limit: %w", err)
	}
	return nil
}

// CreateActivationCode создает новый код активации
func (db *DB) CreateActivationCode(code string, limit int) (*ActivationCode, error) {
	result, err := db.conn.Exec(
		"INSERT INTO activation_codes (code, limit_count) VALUES (?, ?)",
		code, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create activation code: %w", err)
	}

	codeID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get activation code ID: %w", err)
	}

	return &ActivationCode{
		ID:     codeID,
		Code:   code,
		Status: "active",
		Limit:  limit,
	}, nil
}
