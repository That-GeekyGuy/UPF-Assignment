package main

import (
	"log"
	"sync"

	"upf/Server/config"
	"upf/Server/imsi"
	"upf/Server/pfcp"
	"upf/Server/rule"
)

func main() {
	log.Println("🚀 Starting Multi-Agent gRPC Server...")

	var wg sync.WaitGroup

	wg.Add(4) // We have 4 agents

	go func() {
		defer wg.Done()
		if err := config.StartConfigAgent("3000"); err != nil {
			log.Printf("❌ Config Agent failed: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := imsi.StartIMSIAgent("4678"); err != nil {
			log.Printf("❌ IMSI Agent failed: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := pfcp.StartPFCPAgent("50051"); err != nil {
			log.Printf("❌ PFCP Agent failed: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := rule.StartRuleAgent("2000"); err != nil {
			log.Printf("❌ Rule Agent failed: %v", err)
		}
	}()

	wg.Wait()
	log.Println("✨ All agents have exited.")
}
