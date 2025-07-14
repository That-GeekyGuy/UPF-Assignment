package pfcp

import (
	"log"
	"math/rand"
	"net"
	"time"

	pb "upf/pkg/proto"

	"google.golang.org/grpc"
)

type flowmeasuredata struct {
	Total_Packets uint64
	Rx_Packet     uint64
	Tx_Packet     uint64
	Rx_Speed      uint64
	Tx_Speed      uint64
	Total_Speed   uint64
	All_IMSI      []string
}

type pfcpserver struct {
	pb.UnimplementedRequestServer
	flowData map[string]flowmeasuredata
	imsi     []string
	count    uint64
}

// Now a **server-streaming** method
func (s *pfcpserver) PutRequest(req *pb.FlowRequest, stream pb.Request_PutRequestServer) error {


	for {
		s.count++
		// get & initialize flow data
		data, ok := s.flowData[req.Fseid]
		if !ok {
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
		data.Rx_Packet += uint64(rand.Intn(50))
		data.Tx_Packet += uint64(rand.Intn(50))
		data.Rx_Speed += uint64(rand.Intn(1000))
		data.Tx_Speed += uint64(rand.Intn(1000))
		// ✅ store the updated struct back into the map!
		s.flowData[req.Fseid] = data

		// send the updated data
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

		log.Printf("📤 Sent update: Rx=%d Tx=%d Total=%d", data.Rx_Packet, data.Tx_Packet, data.Total_Packets)

		time.Sleep(2 * time.Second)
	}
}

func StartPFCPAgent(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	imsiList := []string{"IMSI1", "IMSI2", "IMSI3"}

	srv := &pfcpserver{
		imsi: imsiList,
		flowData: map[string]flowmeasuredata{
			"exampleFSEID": {
				Rx_Packet:     100,
				Tx_Packet:     200,
				Rx_Speed:      3000,
				Tx_Speed:      4000,
				All_IMSI:      imsiList,
			},
		},
	}

	pb.RegisterRequestServer(s, srv)
	log.Println("gRPC server listening on port " + port + "...")
	return s.Serve(lis)
}
