package pfcp

import (
	"context"
	"log"
	"net"

	pb "upf/pkg/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type flowmeasuredata struct {
	Rx_Packet uint64
	Tx_Packet uint64
	Rx_Byte   uint64
	Tx_Byte   uint64
	All_IMSI  []string
}

type pfcpserver struct {
	pb.UnimplementedRequestServer
	flowData map[string]flowmeasuredata
	imsi     []string
	count    uint64
}

// PutRequest handles client requests and increments count each time it is pinged
func (s *pfcpserver) PutRequest(ctx context.Context, req *pb.FlowRequest) (*pb.Reply, error) {
	s.count++ // increment on *every* request, even invalid ones

	data, ok := s.flowData[req.Fseid]
	if !ok {
		// Return a default Reply with count included to signal it's a valid ping
		return &pb.Reply{
			Rx_Packet: 0,
			Tx_Packet: 0,
			Rx_Byte:   0,
			Tx_Byte:   0,
			All_IMSI:  s.imsi,
			Count:     s.count,
		}, status.Error(codes.NotFound, "FSEID not found")
	}

	return &pb.Reply{
		Rx_Packet: data.Rx_Packet,
		Tx_Packet: data.Tx_Packet,
		Rx_Byte:   data.Rx_Byte,
		Tx_Byte:   data.Tx_Byte,
		All_IMSI:  s.imsi,
		Count:     s.count,
	}, nil
}

func StartPFCPAgent(port string) error {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	imsiList := []string{"IMSI1", "IMSI2", "IMSI3"}
	srv := &pfcpserver{
		imsi: imsiList,
		flowData: map[string]flowmeasuredata{
			"exampleFSEID": {
				Rx_Packet: 100,
				Tx_Packet: 200,
				Rx_Byte:   3000,
				Tx_Byte:   4000,
				All_IMSI:  imsiList,
			},
		},
	}
	pb.RegisterRequestServer(s, srv)

	log.Println("gRPC server listening on port 50051...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	return s.Serve(lis)
}
