package validation

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

// StartValidationServer starts the validation server on the specified port
// SeedData represents the structure of initial data
type SeedData struct {
	IMSI   string                       `json:"imsi"`
	FSEIDs map[string]map[string][]Rule `json:"fseids"`
}

// seedDatabase populates the database with initial data
func seedDatabase() {
	// Check if the database is already seeded
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM imsi").Scan(&count)
	if err != nil {
		log.Printf("Error checking if database is seeded: %v", err)
		return
	}

	if count > 0 {
		log.Println("Database already seeded, skipping...")
		return
	}

	// In a real implementation, you would load this from a configuration file
	seedData := SeedData{
		IMSI: "001011234567890",
		FSEIDs: map[string]map[string][]Rule{
			"fseid1": {
				"internet": {
					{PdrId: "pdr1", DNN: "internet"},
					{PdrId: "pdr2", DNN: "internet"},
				},
				"ims": {
					{PdrId: "pdr3", DNN: "ims"},
				},
			},
		},
	}

	// Insert seed data into the database
	tx, err := DB.Begin()
	if err != nil {
		log.Printf("Error beginning transaction: %v", err)
		return
	}

	// Insert IMSI
	result, err := tx.Exec("INSERT INTO imsi (imsi_number) VALUES (?)", seedData.IMSI)
	if err != nil {
		log.Printf("Error inserting IMSI: %v", err)
		tx.Rollback()
		return
	}

	imsiID, _ := result.LastInsertId()

	// Insert FSEIDs and PDRs
	for fseidName, dnnMap := range seedData.FSEIDs {
		// Insert FSEID
		result, err = tx.Exec("INSERT INTO fseid (fseid_value, imsi_id) VALUES (?, ?)", fseidName, imsiID)
		if err != nil {
			log.Printf("Error inserting FSEID: %v", err)
			tx.Rollback()
			return
		}

		fseidID, _ := result.LastInsertId()

		// Insert PDRs
		for _, rules := range dnnMap {
			for _, rule := range rules {
				_, err = tx.Exec(
					"INSERT INTO pdr (fseid_id, pdr_id, dnn, status) VALUES (?, ?, ?, 'active')",
					fseidID, rule.PdrId, rule.DNN,
				)
				if err != nil {
					log.Printf("Error inserting PDR: %v", err)
					tx.Rollback()
					return
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		return
	}

	log.Println("Database seeded successfully!")
}

// StartValidationServer initializes and starts the validation server
func StartValidationServer(port string) error {
	if err := initDB(); err != nil {
		log.Printf("DB Init Failed: %v", err)
		return err
	}
	defer closeDB()

	seedDatabase()

	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "up"})
	})

	// Validation endpoints
	router.GET("/validate", getValidation)
	router.POST("/validate", postValidation)
	router.PUT("/validate", putValidation)
	router.DELETE("/validate", deleteValidation)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("Shutting down Validation Server...")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("Server forced to shutdown: %v", err)
		}
	}()

	log.Printf("🚀 Validation Server started on port %s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("Error starting Validation Server: %v", err)
		return err
	}

	return nil
}
