package processor

import (
	"testing"
)

func TestInjectLocalConnection_NestedChildren(t *testing.T) {
	input := map[string]interface{}{
		"all": map[string]interface{}{
			"children": map[string]interface{}{
				"consul_staging": map[string]interface{}{
					"children": map[string]interface{}{
						"alertmanager_staging": map[string]interface{}{
							"hosts": map[string]interface{}{
								"alertmanager0": map[string]interface{}{
									"ansible_host": "192.168.2.7",
								},
								"alertmanager1": map[string]interface{}{
									"ansible_host": "192.168.2.8",
								},
							},
						},
					},
				},
			},
		},
	}

	injectLocalConnection(input)

	// Verify hosts have ansible_connection: local
	hosts := input["all"].(map[string]interface{})["children"].(map[string]interface{})["consul_staging"].(map[string]interface{})["children"].(map[string]interface{})["alertmanager_staging"].(map[string]interface{})["hosts"].(map[string]interface{})

	for hostName, hostVars := range hosts {
		vars := hostVars.(map[string]interface{})
		if vars["ansible_connection"] != "local" {
			t.Errorf("host %s: ansible_connection = %v, want \"local\"", hostName, vars["ansible_connection"])
		}
	}
}
