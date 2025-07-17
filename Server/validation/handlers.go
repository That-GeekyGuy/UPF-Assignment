package validation

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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

// ErrorResponse defines the error response structure
type ErrorResponse struct {
	Error      string `json:"error"`
	Message    string `json:"message,omitempty"`
	StatusCode int    `json:"status_code"`
}

// getValidation handles GET /validate requests
func getValidation(c *gin.Context) {
	imsi := c.Query("imsi")
	if imsi == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:      "missing_parameter",
			Message:    "IMSI parameter is required",
			StatusCode: http.StatusBadRequest,
		})
		return
	}

	internetPdrs, imsPdrs := getData(imsi)
	c.JSON(http.StatusOK, ValidationResponse{
		Status:       "success",
		IMSI:         imsi,
		InternetPDRs: internetPdrs,
		IMSPDRs:      imsPdrs,
		Timestamp:    time.Now().Format(time.RFC3339),
	})
}

// postValidation handles POST /validate requests
func postValidation(c *gin.Context) {
	var request RequestData
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:      "invalid_request",
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
		})
		return
	}

	processValidation(c, request)
}

// putValidation handles PUT /validate requests
func putValidation(c *gin.Context) {
	var request RequestData
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:      "invalid_request",
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
		})
		return
	}

	// Process the validation
	processValidation(c, request)
}

// deleteValidation handles DELETE /validate requests
func deleteValidation(c *gin.Context) {
	imsi := c.Query("imsi")
	pdrId := c.Query("pdr_id")

	if imsi == "" || pdrId == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:      "missing_parameters",
			Message:    "Both imsi and pdr_id parameters are required",
			StatusCode: http.StatusBadRequest,
		})
		return
	}

	// In a real implementation, you would delete the PDR from the database here
	// For now, we'll just return a success response
	c.JSON(http.StatusOK, ValidationResponse{
		Status:    "success",
		Message:   "PDR deleted successfully",
		IMSI:      imsi,
		PDR:       pdrId,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// processValidation processes the validation request
func processValidation(c *gin.Context, request RequestData) {
	if request.IMSI == "" || request.Rules.PdrId == "" || request.Rules.DNN == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:      "missing_parameters",
			Message:    "imsi, rules.pdr_id, and rules.dnn are required",
			StatusCode: http.StatusBadRequest,
		})
		return
	}

	internetPdrs, imsPdrs := getData(request.IMSI)

	// Check if the PDR exists in either list
	found := false
	foundIn := ""

	if contains(internetPdrs, request.Rules.PdrId) {
		found = true
		foundIn = "internet"
	} else if contains(imsPdrs, request.Rules.PdrId) {
		found = true
		foundIn = "ims"
	}

	if found {
		c.JSON(http.StatusOK, ValidationResponse{
			Status:    "success",
			Message:   "PDR found",
			IMSI:      request.IMSI,
			PDR:       request.Rules.PdrId,
			DNN:       request.Rules.DNN,
			FoundIn:   foundIn,
			Timestamp: time.Now().Format(time.RFC3339),
		})
	} else {
		c.JSON(http.StatusNotFound, ValidationResponse{
			Status:    "not_found",
			Message:   "PDR not found for the given IMSI",
			IMSI:      request.IMSI,
			PDR:       request.Rules.PdrId,
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}
}
