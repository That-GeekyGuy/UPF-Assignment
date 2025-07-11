package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"strings"
	"time"

	pb "upf/pkg/proto"

	"google.golang.org/grpc"
)

func main() {

	reader := bufio.NewReader(os.Stdin)
	for {
		log.Println("Welcome to the UPF Client!")
		log.Print("======Menu======")
		log.Print("1. Get Flow Data")
		log.Print("2. Get Config")
		log.Print("3. Get IMSI")
		log.Print("4. Get Rule")
		log.Print("5. Exit")
		log.Print("===================")
		log.Printf("Select an option (1 or 2 or 3 or 4 or 5): ")
		option, _ := reader.ReadString('\n')
		option = strings.TrimSpace(option)

		if option == "1" {
			serverAddr := os.Getenv("SERVER_ADDRESS")
			if serverAddr == "" {
				serverAddr = "grpc-server-service.upf-namespace.svc.cluster.local"
			}
			conn, err := grpc.Dial(serverAddr+":50051", grpc.WithInsecure())
			if err != nil {
				log.Fatalf("failed to connect: %v", err)
			}
			defer conn.Close()

			client := pb.NewRequestClient(conn)
			log.Print("Enter FSEID to get flow data (press Enter to skip): ")
			fseid, _ := reader.ReadString('\n')
			fseid = strings.TrimSpace(fseid)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			resp, err := client.PutRequest(ctx, &pb.FlowRequest{Fseid: fseid})
			if err != nil {
				log.Printf("Error: %v", err)
				continue
			}

			log.Println("Rx Packet:", resp.Rx_Packet)
			log.Println("Tx Packet:", resp.Tx_Packet)
			log.Println("Rx Byte:", resp.Rx_Byte)
			log.Println("Tx Byte:", resp.Tx_Byte)
			log.Println("All IMSI:", resp.All_IMSI)
			log.Println("Count:", resp.Count)

		} else if option == "2" {
			serverAddr := os.Getenv("SERVER_ADDRESS")
			if serverAddr == "" {
				serverAddr = "grpc-server-service.upf-namespace.svc.cluster.local"
			}
			conn, err := grpc.Dial(serverAddr+":3000", grpc.WithInsecure())
			if err != nil {
				log.Fatalf("failed to connect: %v", err)
			}
			defer conn.Close()

			client := pb.NewRequestClient(conn)
			log.Println("Fetching configuration...")
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
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

			log.Println("=== Config Summary ===")
			log.Printf("Mode: %s", cfg.GetMode())
			log.Printf("Log Level: %s", cfg.GetLogLevel())
			log.Printf("Workers: %d", cfg.GetWorkers())

			if cfg.GetSim() != nil {
				log.Printf("Simulation Max Sessions: %d", cfg.GetSim().GetMaxSessions())
				log.Printf("Sim Core: %s", cfg.GetSim().GetCore())
			}

			if cfg.GetAccess() != nil {
				log.Printf("Access IF: %s", cfg.GetAccess().GetIfname())
			}
			if cfg.GetCore() != nil {
				log.Printf("Core IF: %s", cfg.GetCore().GetIfname())
			}

			log.Printf("Enable P4RT: %v", cfg.GetEnableP4Rt())
			log.Printf("Enable Heartbeat Timer: %v", cfg.GetEnableHbTimer())

			if cfg.GetCpiface() != nil {
				log.Printf("CP DNN: %s", cfg.GetCpiface().GetDnn())
				log.Printf("CP Peers: %v", cfg.GetCpiface().GetPeers())
			}

			if cfg.GetP4Rtciface() != nil {
				log.Printf("P4 Server: %s", cfg.GetP4Rtciface().GetP4RtcServer())
				log.Printf("P4 Port: %s", cfg.GetP4Rtciface().GetP4RtcPort())
			}
		} else if option == "3" {
			serverAddr := os.Getenv("SERVER_ADDRESS")
			if serverAddr == "" {
				serverAddr = "grpc-server-service.upf-namespace.svc.cluster.local"
			}
			conn, err := grpc.Dial(serverAddr+":4678", grpc.WithInsecure())
			if err != nil {
				log.Fatalf("Failed to connect: %v", err)
			}
			defer conn.Close()

			client := pb.NewRequestClient(conn)
			reader := bufio.NewReader(os.Stdin)

			log.Print("Enter the IMSI to search: ")
			imsi, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalf("Failed to read input: %v", err)
				continue
			}
			imsi = strings.TrimSpace(imsi)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			imsiResp, err := client.GetIMSI(ctx, &pb.IMSIRequest{Imsi: imsi})
			if err != nil {
				log.Fatalf("Could not get IMSI: %v", err)
				continue
			}

			log.Println("=== IMSI Summary ===")
			log.Printf("IMSI: %v", imsi)

			if len(imsiResp.GetImsi()) > 0 {
				data := imsiResp.GetImsi()[0]
				log.Printf("Internet: %s", data.GetInternet())
				log.Printf("IMS: %s", data.GetIMS())
			} else {
				log.Println("No IMSI data received.")
			}
		} else if option == "4" {
			serverAddr := os.Getenv("SERVER_ADDRESS")
			if serverAddr == "" {
				serverAddr = "grpc-server-service.upf-namespace.svc.cluster.local"
			}
			conn, err := grpc.Dial(serverAddr+":2000", grpc.WithInsecure())
			if err != nil {
				log.Fatalf("failed to connect: %v", err)
			}
			defer conn.Close()

			client := pb.NewRequestClient(conn)
			reader := bufio.NewReader(os.Stdin)
			log.Println("Enter the FSIED")
			fsied, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalf("Failed to read input: %v", err)
				continue
			}
			fsied = strings.TrimSpace(fsied)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
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
			log.Println("=== Rules Summary ===")
			log.Printf("FSIED: %s", fsied)
			log.Printf("PDR ID: %s", cfg.Pdr.PdrId)
			log.Printf("FAR ID: %s", cfg.Far.FarId)
			log.Printf("QER ID: %s", cfg.Qer.QerId)
			log.Printf("URR ID: %s", cfg.Urr.UrrId)
		} else if option == "5" {
			log.Println("Goodbye!")
			break
		} else {
			log.Println("Invalid option selected")
		}
	}
}
