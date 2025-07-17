/*
Package config implements the configuration management agent for the UPF service.
It provides gRPC endpoints for retrieving UPF configuration and handles the parsing
of configuration files.
*/
package config

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"

	pb "upf/pkg/proto"

	"github.com/tidwall/jsonc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// server implements the gRPC Request service for configuration management
type server struct {
	pb.UnimplementedRequestServer
}

// UPFConfig represents the complete configuration structure for the UPF service
type UPFConfig struct {
	Mode                     string         `json:"mode"`                        // Operating mode of the UPF
	TableSizes               TableSizes     `json:"table_sizes"`                 // Size configurations for lookup tables
	LogLevel                 string         `json:"log_level"`                   // Logging verbosity level
	Sim                      SimConfig      `json:"sim"`                         // Simulation-related configurations
	HWChecksum               bool           `json:"hwcksum"`                     // Hardware checksum enable flag
	GTPPSC                   bool           `json:"gtppsc"`                      // GTP PSC feature enable flag
	DDP                      bool           `json:"ddp"`                         // Dynamic Data Path enable flag
	MeasureUPF               bool           `json:"measure_upf"`                 // UPF measurement enable flag
	MeasureFlow              bool           `json:"measure_flow"`                // Flow measurement enable flag
	Access                   Interface      `json:"access"`                      // Access interface configuration
	Core                     Interface      `json:"core"`                        // Core interface configuration
	Workers                  int            `json:"workers"`                     // Number of worker threads
	MaxReqRetries            int            `json:"max_req_retries"`             // Maximum request retry attempts
	RespTimeout              string         `json:"resp_timeout"`                // Response timeout duration
	EnableNTF                bool           `json:"enable_ntf"`                  // Network Token Function enable flag
	EnableP4RT               bool           `json:"enable_p4rt"`                 // P4 Runtime enable flag
	EnableHBTimer            bool           `json:"enable_hbTimer"`              // Heartbeat timer enable flag
	EnableGTPUPathMonitoring bool           `json:"enable_gtpu_path_monitoring"` // GTPU path monitoring flag
	QCIQoS                   []QoSConfig    `json:"qci_qos_config"`              // QoS configurations per QCI
	SliceRateLimit           SliceRateLimit `json:"slice_rate_limit_config"`     // Slice rate limiting configuration
	CPInterface              CPInterface    `json:"cpiface"`                     // Control Plane interface configuration
	P4RTCInterface           P4RTCInterface `json:"p4rtciface"`                  // P4 Runtime Traffic Control interface
}

// TableSizes defines the sizes for various lookup tables used in the UPF
type TableSizes struct {
	PDRLookup        int `json:"pdrLookup"`        // Packet Detection Rule lookup table size
	FlowMeasure      int `json:"flowMeasure"`      // Flow measurement table size
	AppQERLookup     int `json:"appQERLookup"`     // Application QER lookup table size
	SessionQERLookup int `json:"sessionQERLookup"` // Session QER lookup table size
	FARLookup        int `json:"farLookup"`        // Forward Action Rule lookup table size
}

// SimConfig contains simulation-specific configuration parameters
type SimConfig struct {
	Core        string `json:"core"`          // Core network address
	MaxSessions int    `json:"max_sessions"`  // Maximum number of simultaneous sessions
	StartUEIP   string `json:"start_ue_ip"`   // Starting IP address for UE range
	StartENBIP  string `json:"start_enb_ip"`  // Starting IP address for eNodeB range
	StartAUPFIP string `json:"start_aupf_ip"` // Starting IP address for AUPF range
	N6AppIP     string `json:"n6_app_ip"`     // N6 interface application IP
	N9AppIP     string `json:"n9_app_ip"`     // N9 interface application IP
	StartN3TEID string `json:"start_n3_teid"` // Starting N3 TEID value
	StartN9TEID string `json:"start_n9_teid"` // Starting N9 TEID value
	UplinkMBR   int    `json:"uplink_mbr"`    // Uplink Maximum Bit Rate
	UplinkGBR   int    `json:"uplink_gbr"`    // Uplink Guaranteed Bit Rate
	DownlinkMBR int    `json:"downlink_mbr"`  // Downlink Maximum Bit Rate
	DownlinkGBR int    `json:"downlink_gbr"`  // Downlink Guaranteed Bit Rate
	PktSize     int    `json:"pkt_size"`      // Packet size for simulation
	TotalFlows  int    `json:"total_flows"`   // Total number of flows to simulate
}

// Interface defines network interface configuration
type Interface struct {
	IfName string `json:"ifname"` // Interface name
}

// QoSConfig defines Quality of Service parameters for a specific QCI
type QoSConfig struct {
	QCI             int `json:"qci"`                         // QoS Class Identifier
	CBS             int `json:"cbs"`                         // Committed Burst Size
	EBS             int `json:"ebs"`                         // Excess Burst Size
	PBS             int `json:"pbs"`                         // Peak Burst Size
	BurstDurationMS int `json:"burst_duration_ms,omitempty"` // Burst Duration in milliseconds
	Priority        int `json:"priority"`                    // QoS priority level
}

// SliceRateLimit defines rate limiting parameters for network slices
type SliceRateLimit struct {
	N6Bps        int `json:"n6_bps"`         // N6 interface rate limit in bps
	N6BurstBytes int `json:"n6_burst_bytes"` // N6 interface burst size in bytes
	N3Bps        int `json:"n3_bps"`         // N3 interface rate limit in bps
	N3BurstBytes int `json:"n3_burst_bytes"` // N3 interface burst size in bytes
}

// CPInterface defines Control Plane interface configuration
type CPInterface struct {
	Peers           []string `json:"peers"`              // List of CP peer addresses
	DNN             string   `json:"dnn"`                // Data Network Name
	HTTPPort        string   `json:"http_port"`          // HTTP port for CP interface
	EnableUEIPAlloc bool     `json:"enable_ue_ip_alloc"` // Enable UE IP allocation
	UEIPPool        string   `json:"ue_ip_pool"`         // UE IP address pool
}

// P4RTCInterface defines P4 Runtime Traffic Control interface configuration
type P4RTCInterface struct {
	AccessIP            string `json:"access_ip"`              // Access IP address for P4RTC
	P4RTCServer         string `json:"p4rtc_server"`           // P4RTC server address
	P4RTCPort           string `json:"p4rtc_port"`             // P4RTC port
	SliceID             int    `json:"slice_id"`               // Slice identifier
	DefaultTC           int    `json:"default_tc"`             // Default traffic class
	ClearStateOnRestart bool   `json:"clear_state_on_restart"` // Clear state on restart flag
}

func (s *server) GetConfig(ctx context.Context, req *pb.ConfigRequest) (*pb.ConfigReply, error) {
	data, err := ioutil.ReadFile("upf.jsonc")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read config: %v", err)
	}

	cleanJSON := jsonc.ToJSON(data)

	var config UPFConfig
	err = json.Unmarshal(cleanJSON, &config)
	if err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}

	log.Printf("Loaded config in mode: %s", config.Mode)

	return &pb.ConfigReply{
		Config: &pb.UPFConfig{
			Mode:                     config.Mode,
			LogLevel:                 config.LogLevel,
			Hwcksum:                  config.HWChecksum,
			Gtppsc:                   config.GTPPSC,
			Ddp:                      config.DDP,
			MeasureUpf:               config.MeasureUPF,
			MeasureFlow:              config.MeasureFlow,
			Workers:                  int32(config.Workers),
			MaxReqRetries:            int32(config.MaxReqRetries),
			RespTimeout:              config.RespTimeout,
			EnableNtf:                config.EnableNTF,
			EnableP4Rt:               config.EnableP4RT,
			EnableHbTimer:            config.EnableHBTimer,
			EnableGtpuPathMonitoring: config.EnableGTPUPathMonitoring,
			TableSizes: &pb.TableSizes{
				PdrLookup:        int32(config.TableSizes.PDRLookup),
				FlowMeasure:      int32(config.TableSizes.FlowMeasure),
				AppQERLookup:     int32(config.TableSizes.AppQERLookup),
				SessionQERLookup: int32(config.TableSizes.SessionQERLookup),
				FarLookup:        int32(config.TableSizes.FARLookup),
			},
			Sim: &pb.SimConfig{
				Core:        config.Sim.Core,
				MaxSessions: int32(config.Sim.MaxSessions),
				StartUeIp:   config.Sim.StartUEIP,
				StartEnbIp:  config.Sim.StartENBIP,
				StartAupfIp: config.Sim.StartAUPFIP,
				N6AppIp:     config.Sim.N6AppIP,
				N9AppIp:     config.Sim.N9AppIP,
				StartN3Teid: config.Sim.StartN3TEID,
				StartN9Teid: config.Sim.StartN9TEID,
				UplinkMbr:   int32(config.Sim.UplinkMBR),
				UplinkGbr:   int32(config.Sim.UplinkGBR),
				DownlinkMbr: int32(config.Sim.DownlinkMBR),
				DownlinkGbr: int32(config.Sim.DownlinkGBR),
				PktSize:     int32(config.Sim.PktSize),
				TotalFlows:  int32(config.Sim.TotalFlows),
			},
			Access: &pb.Interface{Ifname: config.Access.IfName},
			Core:   &pb.Interface{Ifname: config.Core.IfName},
			QciQosConfig: func() []*pb.QoSConfig {
				var qos []*pb.QoSConfig
				for _, q := range config.QCIQoS {
					qos = append(qos, &pb.QoSConfig{
						Qci: int32(q.QCI), Cbs: int32(q.CBS), Ebs: int32(q.EBS),
						Pbs: int32(q.PBS), BurstDurationMs: int32(q.BurstDurationMS), Priority: int32(q.Priority),
					})
				}
				return qos
			}(),
			SliceRateLimitConfig: &pb.SliceRateLimit{
				N6Bps: int32(config.SliceRateLimit.N6Bps), N6BurstBytes: int32(config.SliceRateLimit.N6BurstBytes),
				N3Bps: int32(config.SliceRateLimit.N3Bps), N3BurstBytes: int32(config.SliceRateLimit.N3BurstBytes),
			},
			Cpiface: &pb.CPInterface{
				Peers: config.CPInterface.Peers, Dnn: config.CPInterface.DNN,
				HttpPort: config.CPInterface.HTTPPort, EnableUeIpAlloc: config.CPInterface.EnableUEIPAlloc,
				UeIpPool: config.CPInterface.UEIPPool,
			},
			P4Rtciface: &pb.P4RTCInterface{
				AccessIp: config.P4RTCInterface.AccessIP, P4RtcServer: config.P4RTCInterface.P4RTCServer,
				P4RtcPort: config.P4RTCInterface.P4RTCPort, SliceId: int32(config.P4RTCInterface.SliceID),
				DefaultTc: int32(config.P4RTCInterface.DefaultTC), ClearStateOnRestart: config.P4RTCInterface.ClearStateOnRestart,
			},
		},
	}, nil
}

func StartConfigAgent(port string) error {
	lis, err := net.Listen("tcp", ":3000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	srv := &server{}
	pb.RegisterRequestServer(s, srv)

	log.Println("gRPC server listening on port 3000...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	return s.Serve(lis)
}
