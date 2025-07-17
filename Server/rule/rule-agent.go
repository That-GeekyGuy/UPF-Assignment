/*
Package rule implements the Rule management agent for the UPF service.
It provides gRPC endpoints for retrieving session rules including PDR (Packet Detection Rules),
FAR (Forwarding Action Rules), QER (QoS Enforcement Rules), and URR (Usage Reporting Rules).
*/
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

// Sessions represents a complete set of rules for a UPF session
type Sessions struct {
	pdr Pdrstruct // Packet Detection Rules
	far Farstruct // Forwarding Action Rules
	qer Qerstruct // QoS Enforcement Rules
	urr Urrstruct // Usage Reporting Rules
}

// Pdrstruct defines the structure for Packet Detection Rules
type Pdrstruct struct {
	pdr_id []string // List of PDR identifiers
	fsied  string   // Associated F-SEID
}

// Farstruct defines the structure for Forwarding Action Rules
type Farstruct struct {
	far_id string // FAR identifier
	fsied  string // Associated F-SEID
}

// Qerstruct defines the structure for QoS Enforcement Rules
type Qerstruct struct {
	qer_id string // QER identifier
	fsied  string // Associated F-SEID
}

// Urrstruct defines the structure for Usage Reporting Rules
type Urrstruct struct {
	urr_id string // URR identifier
	fsied  string // Associated F-SEID
}

// ruleServer implements the gRPC Request service for rule management
type ruleServer struct {
	pb.UnimplementedRequestServer
	session map[string]Sessions // Map of F-SEID to session rules
}

// ValidatePDR validates if a PDR is valid for a given IMSI and DNN
func (s *ruleServer) ValidatePDR(ctx context.Context, req *pb.ValidatePDRRequest) (*pb.ValidatePDRReply, error) {
	if req.Imsi == "" || req.PdrId == "" || req.Dnn == "" {
		return &pb.ValidatePDRReply{
			Valid:   false,
			Message: "IMSI, PDR ID, and DNN are required fields",
		}, nil
	}

	// In a real implementation, you would:
	// 1. Look up the IMSI to get the associated F-SEID
	// 2. Find the session using the F-SEID
	// 3. Check if the PDR ID exists in the session's PDRs
	// 4. Validate the DNN against the session's allowed DNNs

	// For this example, we'll do a simple validation
	found := false
	for _, session := range s.session {
		for _, pdrID := range session.pdr.pdr_id {
			if pdrID == req.PdrId {
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return &pb.ValidatePDRReply{
			Valid:   false,
			Message: "PDR not found for the given IMSI",
		}, nil
	}

	// In a real implementation, you would also validate:
	// 1. If the DNN is allowed for this IMSI
	// 2. If the PDR is active and not expired
	// 3. Any other business rules specific to your use case

	return &pb.ValidatePDRReply{
		Valid:   true,
		Message: "PDR validation successful",
	}, nil
}

// GetRule handles requests for retrieving session rules by F-SEID
func (s *ruleServer) GetRule(ctx context.Context, req *pb.RuleRequest) (*pb.RuleReply, error) {
	// Look up session information by F-SEID
	sessionInfo, exists := s.session[req.Fsied]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "Session not found for F-SEID: %s", req.Fsied)
	}

	// Create and return the response with all rule information
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

// StartRuleAgent initializes and starts the rule management gRPC server
// on the specified port with sample session rules
func StartRuleAgent(port string) error {
	// Create TCP listener
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	// Initialize gRPC server
	s := grpc.NewServer()

	// Initialize the rule server with an empty session map
	srv := &ruleServer{
		session: make(map[string]Sessions),
	}

	// Add sample session rules for testing
	// In production, these would be loaded from a persistent store
	srv.session["fseid1"] = Sessions{
		pdr: Pdrstruct{pdr_id: []string{"pdr1", "pdr2"}, fsied: "fseid1"},
		far: Farstruct{far_id: "far1", fsied: "fseid1"},
		qer: Qerstruct{qer_id: "qer1", fsied: "fseid1"},
		urr: Urrstruct{urr_id: "urr1", fsied: "fseid1"},
	}
	srv.session["fseid2"] = Sessions{
		pdr: Pdrstruct{pdr_id: []string{"pdr3", "pdr4"}, fsied: "fseid2"},
		far: Farstruct{far_id: "far2", fsied: "fseid2"},
		qer: Qerstruct{qer_id: "qer2", fsied: "fseid2"},
		urr: Urrstruct{urr_id: "urr2", fsied: "fseid2"},
	}

	// Register the rule server with gRPC
	pb.RegisterRequestServer(s, srv)

	log.Printf("Rule Agent listening on port %s...", port)
	return s.Serve(lis)
}
