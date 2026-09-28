package agent

import (
	"fmt"
	"time"

	"itil-agent/internal/transport"
)

func Heartbeat(client *transport.Client, version string) error {
	body := map[string]any{
		"agent_id":      client.AgentID,
		"timestamp":     time.Now().Format(time.RFC3339),
		"agent_version": version,
	}
	parsed, status, err := client.PostMethod(
		"itil_master_control.it_asset_management.api.agent_heartbeat",
		body,
	)
	if err != nil {
		return err
	}
	msg, _ := parsed["message"].(map[string]any)
	if msg == nil {
		return fmt.Errorf("heartbeat failed (%d): %v", status, parsed)
	}
	ok, _ := msg["success"].(bool)
	if !ok {
		return fmt.Errorf("heartbeat rejected: %v", msg["message"])
	}
	return nil
}
