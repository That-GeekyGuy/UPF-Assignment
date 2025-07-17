package validation

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

// Global DB handle
var DB *sql.DB

// InitDB initializes the database connection
func initDB() error {
	var err error
	dsn := "sqluser:password@tcp(127.0.0.1:3306)/upf?parseTime=true"
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	return DB.Ping()
}

// CloseDB closes the database connection
func closeDB() {
	if DB != nil {
		DB.Close()
	}
}

// contains checks if a string is present in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// getData retrieves PDR data for a given IMSI
func getData(imsi string) ([]string, []string) {
	var internetPdrs, imsPdrs []string

	query := `
		SELECT p.pdr_id, p.dnn 
		FROM imsi i
		JOIN fseid f ON i.id = f.imsi_id
		JOIN pdr p ON f.id = p.fseid_id
		WHERE i.imsi_number = ? AND p.status = 'active'
	`
	rows, err := DB.Query(query, imsi)
	if err != nil {
		log.Printf("Query error: %v", err)
		return nil, nil
	}
	defer rows.Close()

	for rows.Next() {
		var pdrID, dnn string
		if err := rows.Scan(&pdrID, &dnn); err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}
		if dnn == "ims" {
			imsPdrs = append(imsPdrs, pdrID)
		} else {
			internetPdrs = append(internetPdrs, pdrID)
		}
	}

	return internetPdrs, imsPdrs
}
