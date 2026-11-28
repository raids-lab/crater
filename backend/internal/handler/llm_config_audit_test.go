// Copyright 2026 The Crater Project Team, RAIDS-Lab
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

package handler

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/raids-lab/crater/pkg/constants"
)

func TestLLMConfigAuditEvents(t *testing.T) {
	t.Parallel()
	for _, scope := range []string{"personal", "platform"} {
		for _, opType := range []string{constants.OpTypeUpdateLLMConfig, constants.OpTypeResetLLMConfig} {
			for _, success := range []bool{true, false} {
				event := newLLMConfigAuditEvent(opType, scope, 7, true, success)
				wantStatus := constants.OpStatusFailed
				if success {
					wantStatus = constants.OpStatusSuccess
				}
				if event.Status != wantStatus || event.Details["scope"] != scope {
					t.Fatalf("incorrect audit metadata: %+v", event)
				}
				if (event.Message == "") != success {
					t.Fatalf("incorrect audit outcome message: %+v", event)
				}
				wantFields := 1
				if scope == "personal" {
					wantFields++
					if event.Target != "personal-llm-config:user:7" || event.Details["userId"] != uint(7) {
						t.Fatalf("incorrect personal target: %+v", event)
					}
				} else if event.Target != "platform-llm-config" {
					t.Fatalf("incorrect platform target: %+v", event)
				}
				if opType == constants.OpTypeUpdateLLMConfig {
					wantFields++
					if event.Details["keySubmitted"] != true {
						t.Fatal("update must record the key submission flag")
					}
				}
				if len(event.Details) != wantFields {
					t.Fatalf("audit contains non-allowlisted fields: %+v", event.Details)
				}
			}
		}
	}
}

func TestLLMConfigAuditNeverIncludesSubmittedKey(t *testing.T) {
	t.Parallel()
	const masked = "********"
	cases := []struct {
		key       string
		submitted bool
	}{
		{"", false},
		{masked, false},
		{"  " + masked + "  ", true},
		{"audit-secret-sentinel", true},
	}
	for _, test := range cases {
		submitted := hasSubmittedLLMKey(test.key, masked)
		if submitted != test.submitted {
			t.Fatalf("incorrect key submission flag for a masked, empty, or new credential")
		}
		for _, success := range []bool{true, false} {
			event := newLLMConfigAuditEvent(constants.OpTypeUpdateLLMConfig, "personal", 7, submitted, success)
			encoded, err := json.Marshal(event)
			if err != nil {
				t.Fatal(err)
			}
			for _, secret := range []string{"audit-secret-sentinel", masked, "baseUrl", "modelName", "apiKey"} {
				if strings.Contains(string(encoded), secret) {
					t.Fatal("audit serialized credentials or non-allowlisted configuration data")
				}
			}
			if event.Details["keySubmitted"] != submitted {
				t.Fatal("audit must preserve the submission flag without the credential")
			}
		}
	}
	// Personal settings normalize whitespace before resolving the masked placeholder.
	if hasSubmittedLLMKey(strings.TrimSpace("  "+masked+"  "), masked) {
		t.Fatal("personal masked key normalization must retain the existing credential")
	}
}
