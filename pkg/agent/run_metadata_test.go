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

package agent

import "testing"

func TestHasMetadataInterception(t *testing.T) {
	tests := []struct {
		name string
		env  []string
		want bool
	}{
		{
			name: "assign mode",
			env:  []string{"FOO=bar", "SCION_METADATA_MODE=assign", "BAZ=qux"},
			want: true,
		},
		{
			name: "block mode",
			env:  []string{"SCION_METADATA_MODE=block"},
			want: true,
		},
		{
			name: "passthrough mode",
			env:  []string{"SCION_METADATA_MODE=passthrough"},
			want: false,
		},
		{
			name: "no metadata mode",
			env:  []string{"FOO=bar", "BAZ=qux"},
			want: false,
		},
		{
			name: "empty env",
			env:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasMetadataInterception(tt.env)
			if got != tt.want {
				t.Errorf("hasMetadataInterception(%v) = %v, want %v", tt.env, got, tt.want)
			}
		})
	}
}
