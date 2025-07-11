package rule

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
type Sessions struct {
	pdr Pdrstruct
	far Farstruct
	qer Qerstruct
	urr Urrstruct
}

type Pdrstruct struct {
	pdr_id string
	fsied  string
}

type Farstruct struct {
	far_id string
	fsied  string
}

type Qerstruct struct {
	qer_id string
	fsied  string
}

type Urrstruct struct {
	urr_id string
	fsied  string
}

type ruleServer struct {
	pb.UnimplementedRequestServer
	session map[string]Sessions
}

func (s *ruleServer) GetRule(ctx context.Context, req *pb.RuleRequest) (*pb.RuleReply, error) {
	// Get the IMSI info from the map
	sessionInfo, exists := s.session[req.Fsied]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "Session not found")
	}

	// Create and return the response
	return &pb.RuleReply{
		Session: &pb.Rulestruct{
			Pdr: &pb.Pdrstruct{
				PdrId: sessionInfo.pdr.pdr_id,
				Fsied: sessionInfo.pdr.fsied,
			},
			Far: &pb.Farstruct{
				FarId: sessionInfo.far.far_id,
				Fsied: sessionInfo.far.fsied,
			},
			Qer: &pb.Qerstruct{
				QerId: sessionInfo.qer.qer_id,
				Fsied: sessionInfo.qer.fsied,
			},
			Urr: &pb.Urrstruct{
				UrrId: sessionInfo.urr.urr_id,
				Fsied: sessionInfo.urr.fsied,
			},
		},
	}, nil
}

func StartRuleAgent(port string) error {
	lis, err := net.Listen("tcp", ":2000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	// Initialize the server with sample data
	srv := &ruleServer{
		session: make(map[string]Sessions),
	}

	// Add some sample IMSI data
	srv.session["fseid1"] = Sessions{pdr: Pdrstruct{pdr_id: "pdr1", fsied: "fsied1"}, far: Farstruct{far_id: "far1", fsied: "fsied1"}, qer: Qerstruct{qer_id: "qer1", fsied: "fsied1"}, urr: Urrstruct{urr_id: "urr1", fsied: "fsied1"}}
	srv.session["fseid2"] = Sessions{pdr: Pdrstruct{pdr_id: "pdr2", fsied: "fsied2"}, far: Farstruct{far_id: "far2", fsied: "fsied2"}, qer: Qerstruct{qer_id: "qer2", fsied: "fsied2"}, urr: Urrstruct{urr_id: "urr2", fsied: "fsied2"}}
	srv.session["fseid3"] = Sessions{pdr: Pdrstruct{pdr_id: "pdr3", fsied: "fsied3"}, far: Farstruct{far_id: "far3", fsied: "fsied3"}, qer: Qerstruct{qer_id: "qer3", fsied: "fsied3"}, urr: Urrstruct{urr_id: "urr3", fsied: "fsied3"}}
	srv.session["fseid4"] = Sessions{pdr: Pdrstruct{pdr_id: "pdr4", fsied: "fsied4"}, far: Farstruct{far_id: "far4", fsied: "fsied4"}, qer: Qerstruct{qer_id: "qer4", fsied: "fsied4"}, urr: Urrstruct{urr_id: "urr4", fsied: "fsied4"}}
	srv.session["fseid5"] = Sessions{pdr: Pdrstruct{pdr_id: "pdr5", fsied: "fsied5"}, far: Farstruct{far_id: "far5", fsied: "fsied5"}, qer: Qerstruct{qer_id: "qer5", fsied: "fsied5"}, urr: Urrstruct{urr_id: "urr5", fsied: "fsied5"}}
	srv.session["fseid6"] = Sessions{pdr: Pdrstruct{pdr_id: "pdr6", fsied: "fsied6"}, far: Farstruct{far_id: "far6", fsied: "fsied6"}, qer: Qerstruct{qer_id: "qer6", fsied: "fsied6"}, urr: Urrstruct{urr_id: "urr6", fsied: "fsied6"}}

	pb.RegisterRequestServer(s, srv)

	log.Println("gRPC server listening on port 2000...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	return s.Serve(lis)
}
