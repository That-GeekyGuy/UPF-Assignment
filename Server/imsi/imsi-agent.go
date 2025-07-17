/*
Package imsi implements the IMSI (International Mobile Subscriber Identity) management agent
for the UPF service. It provides gRPC endpoints for retrieving IMSI information and
maintains mappings between IMSIs and their associated network services.
*/
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

// IMSI represents the IMSI information with associated service identifiers
type IMSI struct {
	Inter string // Internet service F-SEID
	Ims   string // IMS service F-SEID
}

// imsiServer implements the gRPC Request service for IMSI management
type imsiServer struct {
	pb.UnimplementedRequestServer
	imsi map[string]IMSI // Map of IMSI to service identifiers
}

// GetIMSI handles IMSI information requests by looking up the IMSI in the server's database
// and returning the associated service information
func (s *imsiServer) GetIMSI(ctx context.Context, req *pb.IMSIRequest) (*pb.IMSIReply, error) {
	// Look up the IMSI info in the map
	imsiInfo, exists := s.imsi[req.Imsi]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "IMSI not found: %s", req.Imsi)
	}

	// Create and return the response with the found IMSI information
	return &pb.IMSIReply{
		Imsi: []*pb.IMSIStruct{{
			Internet: imsiInfo.Inter,
			IMS:      imsiInfo.Ims,
		}},
	}, nil
}

// StartIMSIAgent initializes and starts the IMSI management gRPC server
// on the specified port with sample IMSI data
func StartIMSIAgent(port string) error {
	// Create TCP listener
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	// Initialize gRPC server
	s := grpc.NewServer()

	// Initialize the IMSI server with sample data
	srv := &imsiServer{
		imsi: make(map[string]IMSI),
	}

	// Add sample IMSI data for testing
	// In production, this would be replaced with real IMSI data
	srv.imsi["IMSI1"] = IMSI{Inter: "fseid1", Ims: "fseid2"}
	srv.imsi["IMSI2"] = IMSI{Inter: "fseid3", Ims: "fseid4"}
	srv.imsi["IMSI3"] = IMSI{Inter: "fseid5", Ims: "fseid6"}

	// Register the IMSI server with gRPC
	pb.RegisterRequestServer(s, srv)

	log.Printf("IMSI Agent listening on port %s...", port)
	return s.Serve(lis)
}
