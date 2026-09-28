package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"itil-agent/internal/agent"
	"itil-agent/internal/config"
	"itil-agent/internal/transport"
)

const agentVersion = "0.1.0"

func main() {
	cfgPath := flag.String("config", "configs/agent.yaml", "path to agent config")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	id, err := agent.LoadOrCreateUUID(cfg.UUIDFile)
	if err != nil {
		log.Fatalf("uuid: %v", err)
	}
	log.Printf("agent UUID: %s", id)

	hostname, _ := os.Hostname()
	client := transport.NewClient(cfg.ServerURL, id, cfg.AgentSecret, cfg.TLSSkipVerify)

	res, err := agent.Register(client, hostname, agentVersion)
	if err != nil {
		log.Fatalf("register: %v", err)
	}
	log.Printf("registered device_id=%s heartbeat=%vs inventory=%vs",
		res.DeviceID, res.HeartbeatInterval, res.InventoryInterval)

	hbEvery := time.Duration(cfg.HeartbeatInterval) * time.Second
	if res.HeartbeatInterval > 0 {
		hbEvery = time.Duration(res.HeartbeatInterval) * time.Second
	}

	// Heartbeat segera sekali
	if err := agent.Heartbeat(client, agentVersion); err != nil {
		log.Printf("heartbeat error: %v", err)
	} else {
		log.Printf("heartbeat OK")
	}
	// Inventory segera sekali
	if err := agent.SendInventory(client); err != nil {
		log.Printf("inventory error: %v", err)
	} else {
		log.Printf("inventory OK")
	}

	invEvery := time.Duration(cfg.InventoryInterval) * time.Second
	if res.InventoryInterval > 0 {
		invEvery = time.Duration(res.InventoryInterval) * time.Second
	}
	invTicker := time.NewTicker(invEvery)
	defer invTicker.Stop()

	ticker := time.NewTicker(hbEvery)
	defer ticker.Stop()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("heartbeat loop every %s (Ctrl+C to stop)", hbEvery)
	for {
		select {
		case <-invTicker.C:
			if err := agent.SendInventory(client); err != nil {
				log.Printf("inventory error: %v", err)
			} else {
				log.Printf("inventory OK")
			}
				
		case <-ticker.C:
			if err := agent.Heartbeat(client, agentVersion); err != nil {
				log.Printf("heartbeat error: %v", err)
			} else {
				log.Printf("heartbeat OK")
			}
		case <-sig:
			log.Printf("shutting down")
			return
		}
	}
}
