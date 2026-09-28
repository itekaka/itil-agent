package agent

import (
	"fmt"

	"itil-agent/internal/collectors"
	"itil-agent/internal/transport"
)

func SendInventory(client *transport.Client) error {
	body := collectors.BuildInventoryPayload(client.AgentID)
	parsed, status, err := client.PostMethod(
		"itil_master_control.it_asset_management.api.agent_inventory",
		body,
	)
	if err != nil {
		return err
	}
	msg, _ := parsed["message"].(map[string]any)
	if msg == nil {
		return fmt.Errorf("inventory failed (%d): %v", status, parsed)
	}
	ok, _ := msg["success"].(bool)
	if !ok {
		return fmt.Errorf("inventory rejected: %v", msg["message"])
	}
	return nil
}
