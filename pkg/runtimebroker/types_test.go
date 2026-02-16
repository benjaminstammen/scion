// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtimebroker

import (
	"testing"

	"github.com/ptone/scion-agent/pkg/api"
)

func TestAgentInfoToResponse(t *testing.T) {
	tests := []struct {
		name           string
		info           api.AgentInfo
		expectedStatus string
		expectedReady  bool
	}{
		{
			name: "status already set passes through unchanged",
			info: api.AgentInfo{
				Name:            "agent-1",
				Status:          "running",
				ContainerStatus: "created", // should be ignored
			},
			expectedStatus: "running",
			expectedReady:  true,
		},
		{
			name: "status already set to non-running value",
			info: api.AgentInfo{
				Name:            "agent-2",
				Status:          "resumed",
				ContainerStatus: "Up 5 minutes",
			},
			expectedStatus: "resumed",
			expectedReady:  false,
		},
		{
			name: "empty status with container up maps to running",
			info: api.AgentInfo{
				Name:            "agent-3",
				ContainerStatus: "Up 2 hours",
			},
			expectedStatus: AgentStatusRunning,
			expectedReady:  true,
		},
		{
			name: "empty status with container running maps to running",
			info: api.AgentInfo{
				Name:            "agent-4",
				ContainerStatus: "running",
			},
			expectedStatus: AgentStatusRunning,
			expectedReady:  true,
		},
		{
			name: "empty status with container created maps to provisioning",
			info: api.AgentInfo{
				Name:            "agent-5",
				ContainerStatus: "created",
			},
			expectedStatus: AgentStatusProvisioning,
			expectedReady:  false,
		},
		{
			name: "empty status with container exited maps to stopped",
			info: api.AgentInfo{
				Name:            "agent-6",
				ContainerStatus: "Exited (0) 5 minutes ago",
			},
			expectedStatus: AgentStatusStopped,
			expectedReady:  false,
		},
		{
			name: "empty status with container stopped maps to stopped",
			info: api.AgentInfo{
				Name:            "agent-7",
				ContainerStatus: "stopped",
			},
			expectedStatus: AgentStatusStopped,
			expectedReady:  false,
		},
		{
			name: "empty status with empty container status maps to pending",
			info: api.AgentInfo{
				Name: "agent-8",
			},
			expectedStatus: AgentStatusPending,
			expectedReady:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := AgentInfoToResponse(tt.info)
			if resp.Status != tt.expectedStatus {
				t.Errorf("Status = %q, want %q", resp.Status, tt.expectedStatus)
			}
			if resp.Ready != tt.expectedReady {
				t.Errorf("Ready = %v, want %v", resp.Ready, tt.expectedReady)
			}
		})
	}
}
