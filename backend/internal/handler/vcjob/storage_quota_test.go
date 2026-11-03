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

package vcjob

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/util"
)

func TestCreateJobsPassStorageMountsToQuotaCheck(t *testing.T) {
	mgr := &VolcanojobMgr{}
	for _, endpoint := range []struct {
		name    string
		handler gin.HandlerFunc
	}{
		{"custom", mgr.CreateTrainingJob},
		{"jupyter", mgr.CreateJupyterJob},
		{"webide", mgr.CreateWebIDEJob},
		{"pytorch", mgr.CreatePytorchJob},
		{"tensorflow", mgr.CreateTensorflowJob},
	} {
		for _, mounted := range []bool{false, true} {
			name := "without mounts"
			if mounted {
				name = "with mounts"
			}
			t.Run(endpoint.name+"/"+name, func(t *testing.T) {
				var mounts []util.VolumeMount
				if mounted {
					mounts = []util.VolumeMount{{Type: util.FileType, SubPath: "public/models", MountPath: "/models"}}
				}
				body, err := json.Marshal(map[string]any{
					"name": "quota-test", "workingDir": "/tmp",
					"image":        map[string]string{"imageLink": "example/image:test"},
					"volumeMounts": mounts,
				})
				if err != nil {
					t.Fatal(err)
				}
				tokenMock := mockey.Mock(util.GetToken).Return(util.JWTMessage{Username: "alice"}).Build()
				t.Cleanup(func() { tokenMock.UnPatch() })
				waitingMock := mockey.Mock((*VolcanojobMgr).resolveWaitingTolerance).
					To(func(_ *VolcanojobMgr, _ context.Context) (*int64, error) { return nil, nil }).Build()
				t.Cleanup(func() { waitingMock.UnPatch() })
				checked := false
				quotaMock := mockey.Mock(util.CheckStorageQuota).To(func(username string, actual []util.VolumeMount, _ kubernetes.Interface, _ *rest.Config) error {
					checked = true
					if username != "alice" || !reflect.DeepEqual(actual, mounts) {
						t.Errorf("quota check received user=%q mounts=%+v, want %+v", username, actual, mounts)
					}
					if len(actual) != 0 {
						return bizerr.Internal.FileSystemError.New("quota provider unavailable")
					}
					return nil
				}).Build()
				t.Cleanup(func() { quotaMock.UnPatch() })
				continued := false
				nextMock := mockey.Mock((*VolcanojobMgr).preCheckCreateJob).
					To(func(_ *VolcanojobMgr, _ *gin.Context, _ util.JWTMessage, _ bool) bool {
						continued = true
						// Stop at the next validation boundary to avoid cluster side effects.
						return false
					}).Build()
				t.Cleanup(func() { nextMock.UnPatch() })
				response := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(response)
				c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
				c.Request.Header.Set("Content-Type", "application/json")
				endpoint.handler(c)
				if !checked || continued == mounted {
					t.Fatalf("checked=%v continued=%v mounted=%v response=%s", checked, continued, mounted, response.Body.String())
				}
				if mounted && response.Code != http.StatusInternalServerError {
					t.Fatalf("status=%d, want 500", response.Code)
				}
			})
		}
	}
}

func TestGenerateVolumeMountsWithoutStorage(t *testing.T) {
	for _, mounts := range [][]util.VolumeMount{nil, {}} {
		volumes, volumeMounts, err := GenerateVolumeMounts(context.Background(), mounts, util.JWTMessage{})
		if err != nil {
			t.Fatal(err)
		}
		if len(volumes) != 2 || len(volumeMounts) != 2 {
			t.Fatalf("unexpected generated volumes: %+v", volumes)
		}
		for _, volume := range volumes {
			if volume.PersistentVolumeClaim != nil {
				t.Fatalf("unexpected shared PVC: %+v", volume)
			}
			if volume.EmptyDir == nil && volume.ConfigMap == nil {
				t.Fatalf("unexpected volume source: %+v", volume)
			}
		}
	}
}
