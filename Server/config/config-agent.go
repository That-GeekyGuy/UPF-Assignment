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

type server struct {
	pb.UnimplementedRequestServer
}

type UPFConfig struct {
	Mode                     string         `json:"mode"`
	TableSizes               TableSizes     `json:"table_sizes"`
	LogLevel                 string         `json:"log_level"`
	Sim                      SimConfig      `json:"sim"`
	HWChecksum               bool           `json:"hwcksum"`
	GTPPSC                   bool           `json:"gtppsc"`
	DDP                      bool           `json:"ddp"`
	MeasureUPF               bool           `json:"measure_upf"`
	MeasureFlow              bool           `json:"measure_flow"`
	Access                   Interface      `json:"access"`
	Core                     Interface      `json:"core"`
	Workers                  int            `json:"workers"`
	MaxReqRetries            int            `json:"max_req_retries"`
	RespTimeout              string         `json:"resp_timeout"`
	EnableNTF                bool           `json:"enable_ntf"`
	EnableP4RT               bool           `json:"enable_p4rt"`
	EnableHBTimer            bool           `json:"enable_hbTimer"`
	EnableGTPUPathMonitoring bool           `json:"enable_gtpu_path_monitoring"`
	QCIQoS                   []QoSConfig    `json:"qci_qos_config"`
	SliceRateLimit           SliceRateLimit `json:"slice_rate_limit_config"`
	CPInterface              CPInterface    `json:"cpiface"`
	P4RTCInterface           P4RTCInterface `json:"p4rtciface"`
}

type TableSizes struct {
	PDRLookup        int `json:"pdrLookup"`
	FlowMeasure      int `json:"flowMeasure"`
	AppQERLookup     int `json:"appQERLookup"`
	SessionQERLookup int `json:"sessionQERLookup"`
	FARLookup        int `json:"farLookup"`
}

type SimConfig struct {
	Core        string `json:"core"`
	MaxSessions int    `json:"max_sessions"`
	StartUEIP   string `json:"start_ue_ip"`
	StartENBIP  string `json:"start_enb_ip"`
	StartAUPFIP string `json:"start_aupf_ip"`
	N6AppIP     string `json:"n6_app_ip"`
	N9AppIP     string `json:"n9_app_ip"`
	StartN3TEID string `json:"start_n3_teid"`
	StartN9TEID string `json:"start_n9_teid"`
	UplinkMBR   int    `json:"uplink_mbr"`
	UplinkGBR   int    `json:"uplink_gbr"`
	DownlinkMBR int    `json:"downlink_mbr"`
	DownlinkGBR int    `json:"downlink_gbr"`
	PktSize     int    `json:"pkt_size"`
	TotalFlows  int    `json:"total_flows"`
}

type Interface struct {
	IfName string `json:"ifname"`
}

type QoSConfig struct {
	QCI             int `json:"qci"`
	CBS             int `json:"cbs"`
	EBS             int `json:"ebs"`
	PBS             int `json:"pbs"`
	BurstDurationMS int `json:"burst_duration_ms,omitempty"`
	Priority        int `json:"priority"`
}

type SliceRateLimit struct {
	N6Bps        int `json:"n6_bps"`
	N6BurstBytes int `json:"n6_burst_bytes"`
	N3Bps        int `json:"n3_bps"`
	N3BurstBytes int `json:"n3_burst_bytes"`
}

type CPInterface struct {
	Peers           []string `json:"peers"`
	DNN             string   `json:"dnn"`
	HTTPPort        string   `json:"http_port"`
	EnableUEIPAlloc bool     `json:"enable_ue_ip_alloc"`
	UEIPPool        string   `json:"ue_ip_pool"`
}

type P4RTCInterface struct {
	AccessIP            string `json:"access_ip"`
	P4RTCServer         string `json:"p4rtc_server"`
	P4RTCPort           string `json:"p4rtc_port"`
	SliceID             int    `json:"slice_id"`
	DefaultTC           int    `json:"default_tc"`
	ClearStateOnRestart bool   `json:"clear_state_on_restart"`
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
