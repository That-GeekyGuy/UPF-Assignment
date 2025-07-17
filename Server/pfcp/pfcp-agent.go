/*
Package pfcp implements the PFCP (Packet Forwarding Control Protocol) agent for the UPF service.
It provides gRPC endpoints for streaming flow measurement data and simulates packet forwarding
statistics for testing and demonstration purposes.
*/
package pfcp

import (
	"log"
	"math/rand"
	"net"
	"time"

	pb "upf/pkg/proto"

	"google.golang.org/grpc"
)

// flowmeasuredata holds packet flow statistics and related IMSI information
type flowmeasuredata struct {
	Total_Packets uint64   // Total number of packets (Rx + Tx)
	Rx_Packet     uint64   // Number of received packets
	Tx_Packet     uint64   // Number of transmitted packets
	Rx_Speed      uint64   // Current receive speed
	Tx_Speed      uint64   // Current transmit speed
	Total_Speed   uint64   // Total speed (Rx + Tx)
	All_IMSI      []string // List of all IMSIs associated with the flow
}

// pfcpserver implements the gRPC Request service for PFCP management
type pfcpserver struct {
	pb.UnimplementedRequestServer
	flowData map[string]flowmeasuredata // Map of FSEID to flow measurement data
	imsi     []string                   // List of available IMSIs
	count    uint64                     // Counter for updates sent
}

// Now a **server-streaming** method
// PutRequest implements a server-streaming RPC that continuously sends
// flow measurement updates to the client for a specific FSEID
func (s *pfcpserver) PutRequest(req *pb.FlowRequest, stream pb.Request_PutRequestServer) error {

	for {
		s.count++
		// get & initialize flow data
		data, ok := s.flowData[req.Fseid]
		if !ok {
			// Initialize new flow data if none exists
			data = flowmeasuredata{
				Total_Packets: 0,
				Rx_Packet:     0,
				Tx_Packet:     0,
				Rx_Speed:      0,
				Tx_Speed:      0,
				Total_Speed:   0,
				All_IMSI:      s.imsi,
			}
		}

		// simulate dynamic updates
		data.Rx_Packet += uint64(rand.Intn(50))  // Random RX packet increment
		data.Tx_Packet += uint64(rand.Intn(50))  // Random TX packet increment
		data.Rx_Speed += uint64(rand.Intn(1000)) // Random RX speed change
		data.Tx_Speed += uint64(rand.Intn(1000)) // Random TX speed change
		// ✅ store the updated struct back into the map!
		s.flowData[req.Fseid] = data

		// Prepare and send the flow statistics update
		err := stream.Send(&pb.Reply{
			Total_Packets: data.Rx_Packet + data.Tx_Packet,
			Rx_Packet:     data.Rx_Packet,
			Tx_Packet:     data.Tx_Packet,
			Rx_Speed:      data.Rx_Speed,
			Tx_Speed:      data.Tx_Speed,
			Total_Speed:   data.Rx_Speed + data.Tx_Speed,
			All_IMSI:      data.All_IMSI,
			Count:         s.count,
		})
		if err != nil {
			log.Printf("❌ Error sending stream: %v", err)
			return err
		}

		log.Printf("📤 Sent update: Rx=%d Tx=%d Total=%d",
			data.Rx_Packet, data.Tx_Packet, data.Rx_Packet+data.Tx_Packet)

		// Wait before sending next update
		time.Sleep(2 * time.Second)
	}
}

// StartPFCPAgent initializes and starts the PFCP management gRPC server
// on the specified port with sample flow data
func StartPFCPAgent(port string) error {
	// Create TCP listener
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	// Initialize gRPC server
	s := grpc.NewServer()

	// Define sample IMSI list
	imsiList := []string{"IMSI1", "IMSI2", "IMSI3"}

	// Initialize the PFCP server with sample data
	srv := &pfcpserver{
		imsi: imsiList,
		flowData: map[string]flowmeasuredata{
			"exampleFSEID": {
				Rx_Packet: 100,  // Initial RX packet count
				Tx_Packet: 200,  // Initial TX packet count
				Rx_Speed:  3000, // Initial RX speed
				Tx_Speed:  4000, // Initial TX speed
				All_IMSI:  imsiList,
			},
		},
	}

	// Register the PFCP server with gRPC
	pb.RegisterRequestServer(s, srv)

	log.Printf("PFCP Agent listening on port %s...", port)
	return s.Serve(lis)
}
