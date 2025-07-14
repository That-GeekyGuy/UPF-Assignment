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
	s.count++

	// simulate a stream of data for this FSEID
	for i := 0; i < 10; i++ {
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

		// simulate random changes in data
		data.Total_Packets += uint64(rand.Intn(100))
		data.Rx_Packet += uint64(rand.Intn(50))
		data.Tx_Packet += uint64(rand.Intn(50))
		data.Rx_Speed += uint64(rand.Intn(1000))
		data.Tx_Speed += uint64(rand.Intn(1000))
		data.Total_Speed = data.Rx_Speed + data.Tx_Speed

		// Send response
		err := stream.Send(&pb.Reply{
			Total_Packets: data.Total_Packets,
			Rx_Packet:     data.Rx_Packet,
			Tx_Packet:     data.Tx_Packet,
			Rx_Speed:      data.Rx_Speed,
			Tx_Speed:      data.Tx_Speed,
			Total_Speed:   data.Total_Speed,
			All_IMSI:      data.All_IMSI,
			Count:         s.count + uint64(i),
		})
		if err != nil {
			return err
		}

		// Wait before sending next message
		time.Sleep(5 * time.Second)
	}

	return nil
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
				Total_Packets: 100,
				Rx_Packet:     100,
				Tx_Packet:     200,
				Rx_Speed:      3000,
				Tx_Speed:      4000,
				Total_Speed:   7000,
				All_IMSI:      imsiList,
			},
		},
	}

	pb.RegisterRequestServer(s, srv)
	log.Println("gRPC server listening on port " + port + "...")
	return s.Serve(lis)
}

