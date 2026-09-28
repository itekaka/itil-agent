package agent

import (
	"fmt"
	"runtime"

	"itil-agent/internal/transport"
)

type RegisterResult struct {
	Success           bool
	DeviceID          string
	HeartbeatInterval float64
	InventoryInterval float64
	Message           string
}

func Register(client *transport.Client, hostname, version string) (*RegisterResult, error) {
	body := map[string]any{
		"agent_id":      client.AgentID,
		"hostname":      hostname,
		"agent_version": version,
		"os":            runtime.GOOS,
		"secret":        client.Secret,
	}
	parsed, status, err := client.PostMethod(
		"itil_master_control.it_asset_management.api.agent_register",
		body,
	)
	if err != nil {
		return nil, err
	}
	msg, _ := parsed["message"].(map[string]any)
	if msg == nil {
		return nil, fmt.Errorf("register failed (%d): %v", status, parsed)
	}
	ok, _ := msg["success"].(bool)
	if !ok {
		return nil, fmt.Errorf("register rejected: %v", msg["message"])
	}
	res := &RegisterResult{Success: true, Message: fmt.Sprint(msg["message"])}
	if v, ok := msg["device_id"].(string); ok {
		res.DeviceID = v
	}
	if v, ok := msg["heartbeat_interval"].(float64); ok {
		res.HeartbeatInterval = v
	}
	if v, ok := msg["inventory_interval"].(float64); ok {
		res.InventoryInterval = v
	}
	return res, nil
}
