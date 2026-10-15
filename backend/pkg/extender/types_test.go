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

package extender

import (
	"encoding/json"
	"testing"

	. "github.com/bytedance/mockey"
	. "github.com/smartystreets/goconvey/convey"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	schedulinginternal "volcano.sh/apis/pkg/apis/scheduling"
)

func TestRequestJobInfo(t *testing.T) {
	PatchConvey("ownerJobName", t, func() {
		var nilReq *requestJobInfo
		So(nilReq.ownerJobName(), ShouldBeEmpty)
		So((&requestJobInfo{Name: jobA}).ownerJobName(), ShouldBeEmpty)
		So((&requestJobInfo{PodGroup: &requestPodGroup{}}).ownerJobName(), ShouldBeEmpty)

		req := vcjobRequest(jobA)
		So(req.ownerJobName(), ShouldEqual, jobA)

		req.PodGroup.OwnerReferences[0].Controller = ptr.To(false)
		So(req.ownerJobName(), ShouldBeEmpty)

		req = vcjobRequest(jobA)
		req.PodGroup.OwnerReferences = append([]metav1.OwnerReference{{
			APIVersion: "apps/v1", Kind: "Deployment", Name: "not-a-job", Controller: ptr.To(true),
		}}, req.PodGroup.OwnerReferences...)
		So(req.ownerJobName(), ShouldEqual, jobA)
	})

	PatchConvey("decodes the untagged pod group shape", t, func() {
		payload := `{"job":{"Name":"job-a","Namespace":"extender-test","PodGroup":{"name":"job-a-uid",` +
			`"namespace":"extender-test","annotations":{"crater.raids.io/user-id":"7"},"ownerReferences":[` +
			`{"apiVersion":"batch.volcano.sh/v1alpha1","kind":"Job","name":"job-a","uid":"uid","controller":true}],` +
			`"Spec":{"MinMember":1,"Queue":"default","MinResources":{"cpu":"2"}},"Status":{"Phase":"Inqueue"},` +
			`"Version":"v1beta1"}}}`
		var req jobEnqueueableRequest
		So(json.Unmarshal([]byte(payload), &req), ShouldBeNil)
		So(req.Job.ownerJobName(), ShouldEqual, jobA)
		So(req.Job.podGroupName(), ShouldEqual, "job-a-uid")
		So(req.Job.Namespace, ShouldEqual, testNamespace)
		// The pod group's spec, status and annotations ride along on the same payload.
		So(req.Job.PodGroup.Annotations, ShouldContainKey, "crater.raids.io/user-id")
		So(req.Job.PodGroup.Spec.Queue, ShouldEqual, publicQueue)
		So(req.Job.PodGroup.Spec.MinMember, ShouldEqual, 1)
		So(req.Job.PodGroup.Spec.MinResources.Cpu().String(), ShouldEqual, "2")
		So(req.Job.PodGroup.Status.Phase, ShouldEqual, schedulinginternal.PodGroupInqueue)
		So(req.Job.PodGroup.Version, ShouldEqual, "v1beta1")

		So(json.Unmarshal([]byte(`{}`), &req), ShouldBeNil)
	})

	PatchConvey("decodes the tagged pod group shape", t, func() {
		payload := `{"job":{"Name":"job-a-uid","PodGroup":{"metadata":{"name":"job-a-uid",` +
			`"annotations":{"crater.raids.io/user-id":"7"},"ownerReferences":[` +
			`{"apiVersion":"batch.volcano.sh/v1alpha1","kind":"Job","name":"job-a","uid":"uid","controller":true}]},` +
			`"spec":{"minMember":1},"status":{"phase":"Pending"},"Version":"v1beta1"}}}`
		var req jobEnqueueableRequest
		So(json.Unmarshal([]byte(payload), &req), ShouldBeNil)
		So(req.Job.ownerJobName(), ShouldEqual, jobA)
		So(req.Job.podGroupName(), ShouldEqual, "job-a-uid")
		So(req.Job.PodGroup.Metadata.Annotations, ShouldContainKey, "crater.raids.io/user-id")
		// The pinned type has no tags, so the lowercase keys land on Spec and Status case-insensitively.
		So(req.Job.PodGroup.Spec.MinMember, ShouldEqual, 1)
		So(req.Job.PodGroup.Status.Phase, ShouldEqual, schedulinginternal.PodGroupPending)
	})
}
