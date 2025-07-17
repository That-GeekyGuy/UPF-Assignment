/*
Package main implements a multi-agent gRPC server that manages various UPF (User Plane Function)
services including configuration, IMSI handling, PFCP protocol, rule management and validation.
*/
package main

import (
	"log"
	"sync"

	"upf/Server/config"
	"upf/Server/imsi"
	"upf/Server/pfcp"
	"upf/Server/rule"
	"upf/Server/validation"
)

// main is the entry point of the server application that starts all agent services
func main() {
	log.Println("🚀 Starting Multi-Agent gRPC Server...")

	var wg sync.WaitGroup
	wg.Add(5) // We have 5 agents running concurrently

	// Start Config Agent on port 3000
	go func() {
		defer wg.Done()
		if err := config.StartConfigAgent("3000"); err != nil {
			log.Printf("❌ Config Agent failed: %v", err)
		}
	}()

	// Start IMSI Agent on port 4678
	go func() {
		defer wg.Done()
		if err := imsi.StartIMSIAgent("4678"); err != nil {
			log.Printf("❌ IMSI Agent failed: %v", err)
		}
	}()

	// Start PFCP Agent on port 50051
	go func() {
		defer wg.Done()
		if err := pfcp.StartPFCPAgent("50051"); err != nil {
			log.Printf("❌ PFCP Agent failed: %v", err)
		}
	}()

	// Start Rule Agent on port 2000
	go func() {
		defer wg.Done()
		if err := rule.StartRuleAgent("2000"); err != nil {
			log.Printf("❌ Rule Agent failed: %v", err)
		}
	}()

	// Start Validation Server on port 8080
	go func() {
		defer wg.Done()
		if err := validation.StartValidationServer("8080"); err != nil {
			log.Printf("❌ Validation Server failed: %v", err)
		}
	}()

	// Wait for all agents to complete
	wg.Wait()
	log.Println("✨ All agents have exited.")
}
