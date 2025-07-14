package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"log"
	"os"
	"strings"

	pb "upf/pkg/proto"

	"google.golang.org/grpc"
)

// ANSI color helpers using fatih/color
var (
    cyan  = color.New(color.FgCyan).SprintFunc()
    green = color.New(color.FgGreen).SprintFunc()
)

func printMenu() {
    fmt.Print("\033[2J\033[H")
    fmt.Printf("%s\n", cyan("┌─────────────────────────────── UPF Client ────────────────────────────────┐"))
    table := tablewriter.NewWriter(os.Stdout)
    table.SetBorder(false)
    table.SetColumnSeparator(" ")
    table.Append([]string{green("1."), "Get Flow Data"})
    table.Append([]string{green("2."), "Get Config"})
    table.Append([]string{green("3."), "Get IMSI"})
    table.Append([]string{green("4."), "Get Rule"})
    table.Append([]string{green("5."), "Exit"})
    table.Render()
    fmt.Printf("%s\n", cyan("└────────────────────────────────────────────────────────────────────────────┘"))
    fmt.Print(green("Select an option [1-5]: "))
}

func main() {

	// Disable default log timestamps/clutter in output
	log.SetOutput(io.Discard)

	reader := bufio.NewReader(os.Stdin)
	for {
		printMenu()
		option, _ := reader.ReadString('\n')
		option = strings.TrimSpace(option)

		if option == "1" {
			serverAddr := os.Getenv("SERVER_ADDRESS")
			if serverAddr == "" {
				serverAddr = "grpc-server-service.upf-namespace.svc.cluster.local"
			}
			log.Printf("Dialing gRPC server at: %s", serverAddr+":PORT")
			conn, err := grpc.Dial(serverAddr+":50051", grpc.WithInsecure())
			if err != nil {
				log.Fatalf("failed to connect: %v", err)
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
		} else if option == "2" {
			serverAddr := os.Getenv("SERVER_ADDRESS")
			if serverAddr == "" {
				serverAddr = "grpc-server-service.upf-namespace.svc.cluster.local"
			}
			log.Printf("Dialing gRPC server at: %s", serverAddr+":3000")
			conn, err := grpc.Dial(serverAddr+":3000", grpc.WithInsecure())
			if err != nil {
				log.Fatalf("failed to connect: %v", err)
			}
			defer conn.Close()

			client := pb.NewRequestClient(conn)
			log.Println("Fetching configuration...")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			configResp, err := client.GetConfig(ctx, &pb.ConfigRequest{})
			if err != nil {
				log.Fatalf("could not get config: %v", err)
				continue
			}

			cfg := configResp.GetConfig()
			if cfg == nil {
				log.Println("Empty config received")
				return
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

		} else if option == "3" {
			serverAddr := os.Getenv("SERVER_ADDRESS")
			if serverAddr == "" {
				serverAddr = "grpc-server-service.upf-namespace.svc.cluster.local"
			}
			log.Printf("Dialing gRPC server at: %s", serverAddr+":PORT")
			conn, err := grpc.Dial(serverAddr+":4678", grpc.WithInsecure())
			if err != nil {
				log.Fatalf("Failed to connect: %v", err)
			}
			defer conn.Close()

			client := pb.NewRequestClient(conn)
			reader := bufio.NewReader(os.Stdin)

			fmt.Print("Enter the IMSI to search: ")
			imsi, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalf("Failed to read input: %v", err)
				continue
			}
			imsi = strings.TrimSpace(imsi)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			imsiResp, err := client.GetIMSI(ctx, &pb.IMSIRequest{Imsi: imsi})
			if err != nil {
				log.Fatalf("Could not get IMSI: %v", err)
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
			// Details already shown in table above
		    fmt.Print("\nPress ENTER to return to menu...")
    reader.ReadString('\n')
        
    } else if option == "4" {
			serverAddr := os.Getenv("SERVER_ADDRESS")
			if serverAddr == "" {
				serverAddr = "grpc-server-service.upf-namespace.svc.cluster.local"
			}
			log.Printf("Dialing gRPC server at: %s", serverAddr+":PORT")
			conn, err := grpc.Dial(serverAddr+":2000", grpc.WithInsecure())
			if err != nil {
				log.Fatalf("failed to connect: %v", err)
			}
			defer conn.Close()

			client := pb.NewRequestClient(conn)
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter the FSIED: ")
			fsied, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalf("Failed to read input: %v", err)
				continue
			}
			fsied = strings.TrimSpace(fsied)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			configResp, err := client.GetRule(ctx, &pb.RuleRequest{Fsied: fsied})
			if err != nil {
				log.Fatalf("could not get the rules: %v", err)
				continue
			}

			cfg := configResp.Session
			if cfg == nil {
				log.Println("Empty rules received")
				return
			}
			fmt.Print("\033[2J\033[H")
ruleTable := tablewriter.NewWriter(os.Stdout)
ruleTable.SetHeader([]string{"Field", "Value"})
ruleTable.Append([]string{"FSIED", fsied})
ruleTable.Append([]string{"PDR ID", cfg.Pdr.PdrId})
ruleTable.Append([]string{"FAR ID", cfg.Far.FarId})
ruleTable.Append([]string{"QER ID", cfg.Qer.QerId})
ruleTable.Append([]string{"URR ID", cfg.Urr.UrrId})
ruleTable.Render()
			// Table already covers these fields
			//
			//
			//
			//
		    fmt.Print("\nPress ENTER to return to menu...")
    reader.ReadString('\n')
        
    } else if option == "5" {
			log.Println("Goodbye!")
			break
		} else {
			log.Println("Invalid option selected")
		}
	}
}