package dbconfig

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	ID     int
	Key    string
	Value  string
}

func InitDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "goadblock.db")
	if err != nil {
		return nil, err
	}

	// Create configurations table
	createConfigTableSQL := `CREATE TABLE IF NOT EXISTS configurations (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"key" TEXT NOT NULL UNIQUE,
		"value" TEXT
	);`

	_, err = db.Exec(createConfigTableSQL)
	if err != nil {
		return nil, err
	}

	// Create domain lists table
	createDomainsTableSQL := `CREATE TABLE IF NOT EXISTS domain_lists (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"list_name" TEXT NOT NULL,
		"domain" TEXT NOT NULL,
		"created_at" DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(list_name, domain)
	);`

	_, err = db.Exec(createDomainsTableSQL)
	if err != nil {
		return nil, err
	}

	// Create whitelist table
	createWhitelistTableSQL := `CREATE TABLE IF NOT EXISTS whitelist (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"domain" TEXT NOT NULL UNIQUE,
		"created_at" DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createWhitelistTableSQL)
	if err != nil {
		return nil, err
	}

	// Create regex patterns table
	createRegexTableSQL := `CREATE TABLE IF NOT EXISTS regex_patterns (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"pattern" TEXT NOT NULL UNIQUE,
		"created_at" DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createRegexTableSQL)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func GetConfig(db *sql.DB, key string) (string, error) {
	query := `SELECT value FROM configurations WHERE key = ?`
	var value string
	err := db.QueryRow(query, key).Scan(&value)
	if err != nil {
		return "", err
	}

	return value, nil
}

func SetConfig(db *sql.DB, key, value string) error {
	insertSQL := `INSERT OR REPLACE INTO configurations (key, value) VALUES (?, ?)`
	_, err := db.Exec(insertSQL, key, value)
	return err
}

// Domain List Functions
func AddDomainToList(db *sql.DB, listName, domain string) error {
	insertSQL := `INSERT OR IGNORE INTO domain_lists (list_name, domain) VALUES (?, ?)`
	_, err := db.Exec(insertSQL, listName, domain)
	return err
}

func RemoveDomainFromList(db *sql.DB, listName, domain string) error {
	deleteSQL := `DELETE FROM domain_lists WHERE list_name = ? AND domain = ?`
	_, err := db.Exec(deleteSQL, listName, domain)
	return err
}

func GetDomainsFromList(db *sql.DB, listName string) ([]string, error) {
	query := `SELECT domain FROM domain_lists WHERE list_name = ?`
	rows, err := db.Query(query, listName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}
	return domains, nil
}

// Whitelist Functions
func AddToWhitelist(db *sql.DB, domain string) error {
	insertSQL := `INSERT OR IGNORE INTO whitelist (domain) VALUES (?)`
	_, err := db.Exec(insertSQL, domain)
	return err
}

func RemoveFromWhitelist(db *sql.DB, domain string) error {
	deleteSQL := `DELETE FROM whitelist WHERE domain = ?`
	_, err := db.Exec(deleteSQL, domain)
	return err
}

func GetWhitelist(db *sql.DB) ([]string, error) {
	query := `SELECT domain FROM whitelist`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}
	return domains, nil
}

// Regex Pattern Functions
func AddRegexPattern(db *sql.DB, pattern string) error {
	insertSQL := `INSERT OR IGNORE INTO regex_patterns (pattern) VALUES (?)`
	_, err := db.Exec(insertSQL, pattern)
	return err
}

func RemoveRegexPattern(db *sql.DB, pattern string) error {
	deleteSQL := `DELETE FROM regex_patterns WHERE pattern = ?`
	_, err := db.Exec(deleteSQL, pattern)
	return err
}

func GetRegexPatterns(db *sql.DB) ([]string, error) {
	query := `SELECT pattern FROM regex_patterns`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []string
	for rows.Next() {
		var pattern string
		if err := rows.Scan(&pattern); err != nil {
			return nil, err
		}
		patterns = append(patterns, pattern)
	}
	return patterns, nil
}

