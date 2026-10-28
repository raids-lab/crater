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

package service

import (
	"context"
	"testing"

	. "github.com/bytedance/mockey"
	. "github.com/smartystreets/goconvey/convey"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
)

func TestUpdateSchedulerExtenderConfig(t *testing.T) {
	svc := &ConfigService{}

	PatchConvey("validates before touching the table", t, func() {
		update := Mock((*ConfigService).updateConfigs).Return(nil).Build()

		So(svc.UpdateSchedulerExtenderConfig(t.Context(), nil), ShouldNotBeNil)
		zero := &UpdateSchedulerExtenderConfigReq{JobWaitingToleranceSeconds: ptr.To(int64(0))}
		So(svc.UpdateSchedulerExtenderConfig(t.Context(), zero), ShouldNotBeNil)
		So(svc.UpdateSchedulerExtenderConfig(t.Context(), &UpdateSchedulerExtenderConfigReq{}), ShouldBeNil)
		So(update.MockTimes(), ShouldEqual, 0)
	})

	PatchConvey("writes only the provided keys", t, func() {
		var written map[string]string
		Mock((*ConfigService).updateConfigs).To(func(_ *ConfigService, _ context.Context, updates map[string]string) error {
			written = updates
			return nil
		}).Build()

		req := &UpdateSchedulerExtenderConfigReq{QueueQuotaEnabled: ptr.To(true), JobWaitingToleranceSeconds: ptr.To(int64(600))}
		So(svc.UpdateSchedulerExtenderConfig(t.Context(), req), ShouldBeNil)
		So(written, ShouldResemble, map[string]string{
			model.ConfigKeyQueueQuotaEnabled:          "true",
			model.ConfigKeyJobWaitingToleranceSeconds: "600",
		})
	})
}
