/*
Package main implements a UPF (User Plane Function) client application that provides
functionality to interact with UPF services including flow data, configuration,
IMSI information, and rule validation.
*/
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	"github.com/olekukonko/tablewriter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "upf/pkg/proto"
)

// Color configuration for terminal output
var (
	cyan  = color.New(color.FgCyan).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
)

// RequestData represents the structure of incoming validation requests
type RequestData struct {
	IMSI  string `json:"imsi"`  // International Mobile Subscriber Identity
	Rules Rule   `json:"rules"` // Associated rules for the IMSI
}

// Rule defines the structure for PDR (Packet Detection Rule) and DNN (Data Network Name)
type Rule struct {
	PdrId string `json:"pdr_id"` // Packet Detection Rule ID
	DNN   string `json:"dnn"`    // Data Network Name
}

// Global variables for server management
var (
	validationServer *gin.Engine           // Gin server instance for validation
	shutdownChan     = make(chan struct{}) // Channel for graceful shutdown
)

// printMenu displays the main menu interface in the terminal
func printMenu() {
	fmt.Print("\033[2J\033[H")
	fmt.Printf("%s\n", cyan("┌────────────────────────────���── UPF Client ────────────────────────────────┐"))
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetColumnSeparator(" ")
	table.Append([]string{green("1."), "Get Flow Data"})
	table.Append([]string{green("2."), "Get Config"})
	table.Append([]string{green("3."), "Get IMSI"})
	table.Append([]string{green("4."), "Get Rule"})
	table.Append([]string{green("5."), "Validate Rules"})
	table.Append([]string{green("6."), "Exit"})
	table.Render()
	fmt.Printf("%s\n", cyan("└────────────────────────────────────────────────────────────────────────────┘"))
	fmt.Print(green("Select an option [1-6]: "))
}

// printValidationMenu displays the validation server menu interface
func printValidationMenu() {
	fmt.Print("\033[2J\033[H")
	fmt.Printf("%s\n", cyan("┌─────────────────────────── Validation Server Menu ──────────────────────────┐"))
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetColumnSeparator(" ")
	table.Append([]string{green("1."), "Start Server"})
	table.Append([]string{green("2."), "Stop Server"})
	table.Append([]string{green("3."), "Return to Main Menu"})
	table.Render()
	fmt.Printf("%s\n", cyan("└────────────────────────────────────────────────────────────────────────────┘"))
	fmt.Print(green("Select an option [1-3]: "))
}

// displayValidationResult formats and displays the validation results in a table format
func displayValidationResult(internetPdrs, imsPdrs []string, request RequestData, found, foundIn, errMsg string) {
	fmt.Print("\033[2J\033[H") // Clear screen
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Field", "Value"})
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetRowLine(true)

	// Basic information
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
	fmt.Println()
}

// getData retrieves PDR information for a given IMSI from both internet and IMS services
// Returns two string slices containing internet PDRs and IMS PDRs respectively
func getData(imsi string) ([]string, []string) {
	var internetPdrs, imsPdrs []string

	serverAddr := os.Getenv("SERVER_ADDRESS")
	if serverAddr == "" {
		serverAddr = "localhost"
	}

	// Connect to IMSI service
	conn, err := grpc.Dial(serverAddr+":4678", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to IMSI service: %v", err)
		return nil, nil
	}
	defer conn.Close()

	client := pb.NewRequestClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get IMSI information
	imsiResp, err := client.GetIMSI(ctx, &pb.IMSIRequest{Imsi: imsi})
	if err != nil {
		log.Printf("Failed to get IMSI info: %v", err)
		return nil, nil
	}

	if len(imsiResp.GetImsi()) == 0 {
		log.Printf("No IMSI data found for: %s", imsi)
		return nil, nil
	}

	data := imsiResp.GetImsi()[0]
	interFseid := data.GetInternet()
	imsFseid := data.GetIMS()

	// Connect to Rule service
	ruleConn, err := grpc.Dial(serverAddr+":2000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to Rule service: %v", err)
		return nil, nil
	}
	defer ruleConn.Close()

	ruleClient := pb.NewRequestClient(ruleConn)

	// Get Internet PDRs
	if interFseid != "" {
		internetRule, err := ruleClient.GetRule(ctx, &pb.RuleRequest{Fsied: interFseid})
		if err == nil && internetRule.Session != nil && internetRule.Session.Pdr != nil {
			internetPdrs = internetRule.Session.Pdr.PdrId
		}
	}

	// Get IMS PDRs
	if imsFseid != "" {
		imsRule, err := ruleClient.GetRule(ctx, &pb.RuleRequest{Fsied: imsFseid})
		if err == nil && imsRule.Session != nil && imsRule.Session.Pdr != nil {
			imsPdrs = imsRule.Session.Pdr.PdrId
		}
	}

	return internetPdrs, imsPdrs
}

// cleanup performs necessary cleanup operations before program termination
func cleanup() {
	if validationServer != nil {
		fmt.Println("\nStopping validation server...")
		stopValidationServer()
	}
	fmt.Println("Goodbye!")
}

// startValidationServer initializes and starts the validation server
func startValidationServer() {
	router := gin.Default()
	validationServer = router

	// Set up the validation route
	router.POST("/validate", func(c *gin.Context) {
		var request RequestData
		if err := c.BindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
			return
		}

		internetPdrs, imsPdrs := getData(request.IMSI)

		// First find the PDR in either slice
		pdrFoundInInternet := false
		pdrFoundInIms := false

		for _, pdr := range internetPdrs {
			if pdr == request.Rules.PdrId {
				pdrFoundInInternet = true
				break
			}
		}

		for _, pdr := range imsPdrs {
			if pdr == request.Rules.PdrId {
				pdrFoundInIms = true
				break
			}
		}

		found := "incorrect"
		var foundIn, errMsg string

		// Check if PDR exists and DNN matches
		if pdrFoundInInternet && request.Rules.DNN == "internet" {
			found = "correct"
			foundIn = "internet"
			c.JSON(http.StatusOK, gin.H{"status": "Correct Results", "message": "Validation successful"})
		} else if pdrFoundInIms && request.Rules.DNN == "ims" {
			found = "correct"
			foundIn = "ims"
			c.JSON(http.StatusOK, gin.H{"status": "Correct Results", "message": "Validation successful"})
		} else if pdrFoundInInternet || pdrFoundInIms {
			errMsg = "PDR exists but DNN mismatch"
			if pdrFoundInInternet {
				foundIn = "internet"
			} else {
				foundIn = "ims"
			}
			c.JSON(http.StatusBadRequest, gin.H{"status": "Incorrect Results", "message": "Validation Un-successful"})
		} else {
			errMsg = "PDR not found"
			c.JSON(http.StatusBadRequest, gin.H{"status": "Incorrect Results", "message": "Validation Un-successful"})
		}

		displayValidationResult(internetPdrs, imsPdrs, request, found, foundIn, errMsg)

	})

	// Start server in goroutine
	go func() {
		fmt.Println("\nValidation server started on http://localhost:8081")
		fmt.Println("Waiting for validation requests...")
		if err := router.Run("localhost:8081"); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()
}

// stopValidationServer gracefully stops the validation server
func stopValidationServer() {
	if validationServer != nil {
		// Create a new server with the current engine
		srv := &http.Server{
			Addr:    ":8081",
			Handler: validationServer,
		}

		// Shutdown with timeout
		go func() {
			if err := srv.Shutdown(context.Background()); err != nil {
				log.Printf("Server shutdown error: %v", err)
			}
		}()

		// Give it a moment to shut down
		time.Sleep(time.Second)
		validationServer = nil
		fmt.Println("Server stopped")
	}
}

func main() {
	// Disable default log timestamps/clutter in output
	log.SetOutput(io.Discard)

	reader := bufio.NewReader(os.Stdin)
	serverRunning := false

	for {
		printMenu()
		option, _ := reader.ReadString('\n')
		option = strings.TrimSpace(option)

		switch option {
		case "1":
			serverAddr := os.Getenv("SERVER_ADDRESS")
			if serverAddr == "" {
				serverAddr = "localhost"
			}

			conn, err := grpc.Dial(serverAddr+":50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("failed to connect: %v", err)
				continue
			}
			defer conn.Close()

			client := pb.NewRequestClient(conn)
			fmt.Print("Enter FSEID to get flow data (press Enter to skip): ")
			fseid, _ := reader.ReadString('\n')
			fseid = strings.TrimSpace(fseid)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			stream, err := client.PutRequest(ctx, &pb.FlowRequest{Fseid: fseid})
			if err != nil {
				log.Printf("Error starting stream: %v", err)
				continue
			}

			fmt.Println("Press ENTER to stop streaming and return to menu...")
			go func() {
				bufio.NewReader(os.Stdin).ReadString('\n')
				cancel()
			}()

			log.Println("Streaming flow data (table updates)...")
			for {
				resp, err := stream.Recv()
				if err != nil {
					log.Printf("Stream ended: %v", err)
					break
				}

				// Clear the screen and move cursor to top-left
				fmt.Print("\033[2J\033[H")

				// Render table
				fmt.Println("+-------------+-------------+-------------+-------------+--------------+---------------+")
				fmt.Println("| Rx Packet   | Tx Packet   | Rx Speed    | Tx Speed    | Total Packet | Total Speed   |")
				fmt.Println("+-------------+-------------+-------------+-------------+--------------+---------------+")
				fmt.Printf("| %-11d | %-11d | %-11d | %-11d | %-12d | %-13d |\n",
					resp.Rx_Packet, resp.Tx_Packet, resp.Rx_Speed, resp.Tx_Speed, resp.Total_Packets, resp.Total_Speed)
				fmt.Println("+-------------+-------------+-------------+-------------+--------------+---------------+")
				fmt.Printf("All IMSI: %v   Updates: %d\n", resp.All_IMSI, resp.Count)
			}

		case "2":
			serverAddr := os.Getenv("SERVER_ADDRESS")
			if serverAddr == "" {
				serverAddr = "localhost"
			}

			conn, err := grpc.Dial(serverAddr+":3000", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("failed to connect: %v", err)
				continue
			}
			defer conn.Close()

			client := pb.NewRequestClient(conn)
			log.Println("Fetching configuration...")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			configResp, err := client.GetConfig(ctx, &pb.ConfigRequest{})
			if err != nil {
				log.Printf("could not get config: %v", err)
				continue
			}

			cfg := configResp.GetConfig()
			if cfg == nil {
				log.Println("Empty config received")
				continue
			}

			// Render config using a table
			fmt.Print("\033[2J\033[H")
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Field", "Value"})
			table.Append([]string{"Mode", cfg.GetMode()})
			table.Append([]string{"Log Level", cfg.GetLogLevel()})
			table.Append([]string{"Workers", fmt.Sprintf("%d", cfg.GetWorkers())})
			if cfg.GetSim() != nil {
				table.Append([]string{"Sim Max Sessions", fmt.Sprintf("%d", cfg.GetSim().GetMaxSessions())})
				table.Append([]string{"Sim Core", cfg.GetSim().GetCore()})
			}
			if cfg.GetAccess() != nil {
				table.Append([]string{"Access IF", cfg.GetAccess().GetIfname()})
			}
			if cfg.GetCore() != nil {
				table.Append([]string{"Core IF", cfg.GetCore().GetIfname()})
			}
			table.Append([]string{"Enable P4RT", fmt.Sprintf("%v", cfg.GetEnableP4Rt())})
			table.Append([]string{"Enable HB Timer", fmt.Sprintf("%v", cfg.GetEnableHbTimer())})
			if cfg.GetCpiface() != nil {
				table.Append([]string{"CP DNN", cfg.GetCpiface().GetDnn()})
				table.Append([]string{"CP Peers", fmt.Sprintf("%v", cfg.GetCpiface().GetPeers())})
			}
			if cfg.GetP4Rtciface() != nil {
				table.Append([]string{"P4 Server", cfg.GetP4Rtciface().GetP4RtcServer()})
				table.Append([]string{"P4 Port", cfg.GetP4Rtciface().GetP4RtcPort()})
			}
			table.Render()
			fmt.Print("\nPress ENTER to return to menu...")
			reader.ReadString('\n')

		case "3":
			serverAddr := os.Getenv("SERVER_ADDRESS")
			if serverAddr == "" {
				serverAddr = "localhost"
			}

			conn, err := grpc.Dial(serverAddr+":4678", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("Failed to connect: %v", err)
				continue
			}
			defer conn.Close()

			client := pb.NewRequestClient(conn)

			fmt.Print("Enter the IMSI to search: ")
			imsi, err := reader.ReadString('\n')
			if err != nil {
				log.Printf("Failed to read input: %v", err)
				continue
			}
			imsi = strings.TrimSpace(imsi)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			imsiResp, err := client.GetIMSI(ctx, &pb.IMSIRequest{Imsi: imsi})
			if err != nil {
				log.Printf("Could not get IMSI: %v", err)
				continue
			}

			fmt.Print("\033[2J\033[H")
			imsiTable := tablewriter.NewWriter(os.Stdout)
			imsiTable.SetHeader([]string{"Field", "Value"})
			imsiTable.Append([]string{"IMSI", imsi})
			if len(imsiResp.GetImsi()) > 0 {
				data := imsiResp.GetImsi()[0]
				imsiTable.Append([]string{"Internet", data.GetInternet()})
				imsiTable.Append([]string{"IMS", data.GetIMS()})
			}
			imsiTable.Render()
			fmt.Print("\nPress ENTER to return to menu...")
			reader.ReadString('\n')

		case "4":
			serverAddr := os.Getenv("SERVER_ADDRESS")
			if serverAddr == "" {
				serverAddr = "localhost"
			}

			conn, err := grpc.Dial(serverAddr+":2000", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("failed to connect: %v", err)
				continue
			}
			defer conn.Close()

			client := pb.NewRequestClient(conn)
			fmt.Print("Enter the FSEID: ")
			fseid, err := reader.ReadString('\n')
			if err != nil {
				log.Printf("Failed to read input: %v", err)
				continue
			}
			fseid = strings.TrimSpace(fseid)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ruleResp, err := client.GetRule(ctx, &pb.RuleRequest{Fsied: fseid})
			if err != nil {
				log.Printf("could not get the rules: %v", err)
				continue
			}

			if ruleResp.Session == nil {
				log.Println("Empty rules received")
				continue
			}

			fmt.Print("\033[2J\033[H")
			ruleTable := tablewriter.NewWriter(os.Stdout)
			ruleTable.SetHeader([]string{"Field", "Value"})
			ruleTable.SetAutoWrapText(false)
			ruleTable.SetAlignment(tablewriter.ALIGN_LEFT)
			ruleTable.SetRowLine(true)

			ruleTable.Append([]string{"FSEID", fseid})
			if ruleResp.Session.Pdr != nil {
				pdrs := ruleResp.Session.Pdr.PdrId
				if len(pdrs) > 0 {
					ruleTable.Append([]string{"PDR IDs", strings.Join(pdrs, ", ")})
				}
			}
			if ruleResp.Session.Far != nil {
				ruleTable.Append([]string{"FAR ID", ruleResp.Session.Far.FarId})
			}
			if ruleResp.Session.Qer != nil {
				ruleTable.Append([]string{"QER ID", ruleResp.Session.Qer.QerId})
			}
			if ruleResp.Session.Urr != nil {
				ruleTable.Append([]string{"URR ID", ruleResp.Session.Urr.UrrId})
			}

			ruleTable.Render()
			fmt.Print("\nPress ENTER to return to menu...")
			reader.ReadString('\n')

		case "5":
			for {
				printValidationMenu()
				subOption, _ := reader.ReadString('\n')
				subOption = strings.TrimSpace(subOption)

				switch subOption {
				case "1":
					if !serverRunning {
						startValidationServer()
						serverRunning = true
						fmt.Println("\nValidation server is now running on http://localhost:8081")
						fmt.Println("You can send POST requests to /validate endpoint")
						fmt.Print("\nPress ENTER to continue...")
						reader.ReadString('\n')
					} else {
						fmt.Println("\nValidation server is already running")
						fmt.Print("\nPress ENTER to continue...")
						reader.ReadString('\n')
					}
				case "2":
					if serverRunning {
						stopValidationServer()
						serverRunning = false
						fmt.Println("\nValidation server stopped")
						fmt.Print("\nPress ENTER to continue...")
						reader.ReadString('\n')
					} else {
						fmt.Println("\nNo validation server is running")
						fmt.Print("\nPress ENTER to continue...")
						reader.ReadString('\n')
					}
				case "3":
					break
				default:
					fmt.Println("Invalid option selected")
				}

				if subOption == "3" {
					break
				}
			}

		case "6":
			cleanup()
			return
		default:
			fmt.Println("Invalid option selected")
		}
	}
}
