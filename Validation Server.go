package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/olekukonko/tablewriter"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

// Global DB handle
var DB *sql.DB

// Rule represents the PDR and DNN information
type Rule struct {
	PdrId string `json:"pdr_id"`
	DNN   string `json:"dnn"`
}

// RequestData represents the incoming request structure
type RequestData struct {
	IMSI  string `json:"imsi"`
	Rules Rule   `json:"rules"`
}

// ValidationResponse defines the structure for validation responses
type ValidationResponse struct {
	Status       string   `json:"status"`
	Message      string   `json:"message,omitempty"`
	IMSI         string   `json:"imsi,omitempty"`
	PDR          string   `json:"pdr,omitempty"`
	DNN          string   `json:"dnn,omitempty"`
	FoundIn      string   `json:"found_in,omitempty"`
	InternetPDRs []string `json:"internet_pdrs,omitempty"`
	IMSPDRs      []string `json:"ims_pdrs,omitempty"`
	Timestamp    string   `json:"timestamp"`
}

type ErrorResponse struct {
	Error      string `json:"error"`
	Message    string `json:"message,omitempty"`
	StatusCode int    `json:"status_code"`
}

// SeedData represents the structure of initial data
type SeedData struct {
	IMSI   string                       `json:"imsi"`
	FSEIDs map[string]map[string][]Rule // dynamic: fseid1, fseid2, ...
}

func InitDB() error {
	var err error
	dsn := "sqluser:password@tcp(127.0.0.1:3306)/upf?parseTime=true"
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	return DB.Ping()
}

func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

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
		var pdrId, dnn string
		if err := rows.Scan(&pdrId, &dnn); err == nil {
			if dnn == "internet" {
				internetPdrs = append(internetPdrs, pdrId)
			} else if dnn == "ims" {
				imsPdrs = append(imsPdrs, pdrId)
			}
		}
	}
	return internetPdrs, imsPdrs
}

func displayValidationResult(internetPdrs, imsPdrs []string, request RequestData, found, foundIn, errMsg string) {
	fmt.Print("\033[2J\033[H")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Field", "Value"})
	table.SetRowLine(true)

	table.Append([]string{"IMSI", request.IMSI})
	table.Append([]string{"Requested PDR", request.Rules.PdrId})
	table.Append([]string{"Requested DNN", request.Rules.DNN})
	table.Append([]string{"Status", found})
	if foundIn != "" {
		table.Append([]string{"Found In", foundIn})
	}
	if errMsg != "" {
		table.Append([]string{"Error", errMsg})
	}
	if len(internetPdrs) > 0 {
		table.Append([]string{"Internet PDRs", strings.Join(internetPdrs, ", ")})
	}
	if len(imsPdrs) > 0 {
		table.Append([]string{"IMS PDRs", strings.Join(imsPdrs, ", ")})
	}
	table.Render()
}

func processValidation(c *gin.Context, request RequestData) {
	internetPdrs, imsPdrs := getData(request.IMSI)

	var found, foundIn, errMsg string

	if len(internetPdrs) == 0 && len(imsPdrs) == 0 {
		found = "Not Found"
		errMsg = "No PDRs found for the given IMSI"
	} else {
		if (request.Rules.DNN == "internet" && contains(internetPdrs, request.Rules.PdrId)) ||
			(request.Rules.DNN == "ims" && contains(imsPdrs, request.Rules.PdrId)) {
			found = "Found"
			foundIn = request.Rules.DNN
		} else {
			found = "Not Found"
		}
	}

	displayValidationResult(internetPdrs, imsPdrs, request, found, foundIn, errMsg)

	if errMsg != "" {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:      "Validation Failed",
			Message:    errMsg,
			StatusCode: http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, ValidationResponse{
		Status:       found,
		IMSI:         request.IMSI,
		PDR:          request.Rules.PdrId,
		DNN:          request.Rules.DNN,
		FoundIn:      foundIn,
		InternetPDRs: internetPdrs,
		IMSPDRs:      imsPdrs,
		Timestamp:    time.Now().Format(time.RFC3339),
	})
}

func getValidation(c *gin.Context) {
	request := RequestData{
		IMSI: c.Query("imsi"),
		Rules: Rule{
			PdrId: c.Query("pdr_id"),
			DNN:   c.Query("dnn"),
		},
	}
	processValidation(c, request)
}

func postValidation(c *gin.Context) {
	var request RequestData
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:      "Invalid request body",
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
		})
		return
	}

	processValidation(c, request)
}

func putValidation(c *gin.Context) {
	var request RequestData
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:      "Invalid request body",
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
		})
		return
	}

	// Check if the PDR exists
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM pdr WHERE pdr_id = ? AND dnn = ?", 
		request.Rules.PdrId, request.Rules.DNN).Scan(&count)
	if err != nil || count == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:      "PDR not found",
			Message:    "The specified PDR does not exist",
			StatusCode: http.StatusNotFound,
		})
		return
	}

	// Update the PDR
	_, err = DB.Exec("UPDATE pdr SET status = 'active' WHERE pdr_id = ? AND dnn = ?", 
		request.Rules.PdrId, request.Rules.DNN)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:      "Failed to update PDR",
			Message:    err.Error(),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "PDR updated successfully",
		"pdr_id":  request.Rules.PdrId,
		"dnn":     request.Rules.DNN,
	})
}

func deleteValidation(c *gin.Context) {
	pdrID := c.Query("pdr_id")
	dnn := c.Query("dnn")

	if pdrID == "" || dnn == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:      "Missing parameters",
			Message:    "Both pdr_id and dnn parameters are required",
			StatusCode: http.StatusBadRequest,
		})
		return
	}

	// Check if the PDR exists
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM pdr WHERE pdr_id = ? AND dnn = ?", pdrID, dnn).Scan(&count)
	if err != nil || count == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:      "PDR not found",
			Message:    "The specified PDR does not exist",
			StatusCode: http.StatusNotFound,
		})
		return
	}

	// Soft delete the PDR (set status to 'inactive')
	_, err = DB.Exec("UPDATE pdr SET status = 'inactive' WHERE pdr_id = ? AND dnn = ?", pdrID, dnn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:      "Failed to delete PDR",
			Message:    err.Error(),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "PDR deleted successfully",
		"pdr_id":  pdrID,
		"dnn":     dnn,
	})
}

func seedDatabase() {
	log.Println("Seeding database with mock data...")

	data := []byte(`[
		{
			"imsi": "IMSI1",
			"fseid1": {"rules": [{"pdr_id": "pdr1", "dnn": "internet"}]},
			"fseid2": {"rules": [{"pdr_id": "pdr2", "dnn": "ims"}]}
		},
		{
			"imsi": "IMSI2",
			"fseid3": {"rules": [{"pdr_id": "pdr3", "dnn": "internet"}]},
			"fseid4": {"rules": [{"pdr_id": "pdr4", "dnn": "ims"}]}
		},
		{
			"imsi": "IMSI3",
			"fseid5": {"rules": [{"pdr_id": "pdr5", "dnn": "internet"}]},
			"fseid6": {"rules": [{"pdr_id": "pdr6", "dnn": "ims"}]}
		}
	]`)

	var entries []map[string]any
	json.Unmarshal(data, &entries)

	for _, entry := range entries {
		imsi := entry["imsi"].(string)
		res, _ := DB.Exec(`INSERT IGNORE INTO imsi (imsi_number) VALUES (?)`, imsi)
		imsiId, _ := res.LastInsertId()
		if imsiId == 0 {
			_ = DB.QueryRow(`SELECT id FROM imsi WHERE imsi_number = ?`, imsi).Scan(&imsiId)
		}
		for key, val := range entry {
			if key == "imsi" {
				continue
			}
			fseidName := key
			_, _ = DB.Exec(`INSERT INTO fseid (fseid_value, imsi_id) VALUES (?, ?)`, fseidName, imsiId)
			var fseidId int64
			_ = DB.QueryRow(`SELECT id FROM fseid WHERE fseid_value = ? AND imsi_id = ?`, fseidName, imsiId).Scan(&fseidId)
			fseidData := val.(map[string]any)
			rules := fseidData["rules"].([]any)
			for _, ruleRaw := range rules {
				rule := ruleRaw.(map[string]any)
				DB.Exec(`INSERT INTO pdr (fseid_id, pdr_id, dnn, status) VALUES (?, ?, ?, 'active')`,
					fseidId, rule["pdr_id"], rule["dnn"])
			}
		}
	}
}

func main() {
	if err := InitDB(); err != nil {
		log.Fatalf("DB Init Failed: %v", err)
	}
	defer CloseDB()

	seedDatabase()

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "up"})
	})
	
	// Validation endpoints
	router.GET("/validate", getValidation)
	router.POST("/validate", postValidation)
	router.PUT("/validate", putValidation)
	router.DELETE("/validate", deleteValidation)

	server := &http.Server{
		Addr:    ":8081",
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()
	log.Println("Server started on :8081")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown failed: %v", err)
	}
	log.Println("Server exited gracefully.")
}
