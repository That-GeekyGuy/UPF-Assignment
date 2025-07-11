package imsi

import (
	"context"
	"log"
	"net"

	pb "upf/pkg/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IMSI represents the IMSI information
// This is a local struct, different from the protobuf IMSIStruct
type IMSI struct {
	Inter string
	Ims   string
}

type imsiServer struct {
	pb.UnimplementedRequestServer
	imsi map[string]IMSI
}

// GetIMSI handles IMSI information requests
func (s *imsiServer) GetIMSI(ctx context.Context, req *pb.IMSIRequest) (*pb.IMSIReply, error) {
	// Get the IMSI info from the map
	imsiInfo, exists := s.imsi[req.Imsi]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "IMSI not found")
	}

	// Create and return the response
	return &pb.IMSIReply{
		Imsi: []*pb.IMSIStruct{{
			Internet: imsiInfo.Inter,
			IMS:      imsiInfo.Ims,
		}},
	}, nil
}

func StartIMSIAgent(port string) error {
	lis, err := net.Listen("tcp", ":4678")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	// Initialize the server with sample data
	srv := &imsiServer{
		imsi: make(map[string]IMSI),
	}

	// Add some sample IMSI data
	srv.imsi["IMSI1"] = IMSI{Inter: "fseid1", Ims: "fseid2"}
	srv.imsi["IMSI2"] = IMSI{Inter: "fseid3", Ims: "fseid4"}
	srv.imsi["IMSI3"] = IMSI{Inter: "fseid5", Ims: "fseid6"}
	pb.RegisterRequestServer(s, srv)

	log.Println("gRPC server listening on port 4678...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	return s.Serve(lis)
}
