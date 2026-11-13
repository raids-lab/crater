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
	"errors"
	"reflect"
	"slices"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
)

func jobPermissionDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	old := query.User
	q := query.Use(db)
	query.User = &q.User
	t.Cleanup(func() { query.User = old })
	return db
}

//nolint:gocyclo // Sequential grant and revoke checks verify live authorization and template preservation.
func TestUserJobCapabilitiesFollowCurrentGrant(t *testing.T) {
	db := jobPermissionDB(t)
	user := model.User{Name: "profiling", Space: "profiling", Role: model.RoleAdmin, Status: model.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	pod := corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}, {Name: "worker"}},
		InitContainers: []corev1.Container{{Name: "init"}}}
	// A platform administrator receives no container grant implicitly.
	if err := ApplyUserJobCapabilities(t.Context(), user.ID, false, &pod); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(pod.Containers[0].SecurityContext.Capabilities.Add, "SYS_ADMIN") {
		t.Fatal("platform role granted SYS_ADMIN")
	}
	for _, enabled := range []bool{true, false, true} {
		if err := AdminUpdateUserAttributes(t.Context(), user.Name, &model.UserAttribute{Nickname: "edited"}, ptr.To(enabled)); err != nil {
			t.Fatal(err)
		}
		other := pod.DeepCopy()
		if err := ApplyUserJobCapabilities(t.Context(), user.ID, enabled, &pod, other); err != nil {
			t.Fatal(err)
		}
		for _, spec := range []*corev1.PodSpec{&pod, other} {
			for _, c := range spec.Containers {
				caps := c.SecurityContext.Capabilities.Add
				if slices.Contains(caps, "SYS_ADMIN") != enabled {
					t.Fatalf("grant %v: %v", enabled, caps)
				}
				if !slices.Contains(caps, "SYS_PTRACE") || slices.Contains(caps, "PERFMON") {
					t.Fatal("expected SYS_PTRACE without default PERFMON")
				}
				if c.SecurityContext.Privileged != nil {
					t.Fatal("enabled privileged")
				}
			}
			if spec.InitContainers[0].SecurityContext != nil || spec.HostPID || spec.HostNetwork {
				t.Fatal("changed workload isolation")
			}
		}
	}
	before := pod.DeepCopy()
	if err := ApplyUserJobCapabilities(t.Context(), user.ID, true, &pod); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, &pod) {
		t.Fatal("not idempotent")
	}
	if err := AdminUpdateUserAttributes(t.Context(), user.Name, &model.UserAttribute{Nickname: "legacy client"}, nil); err != nil {
		t.Fatal(err)
	}
	var saved model.User
	if err := db.First(&saved, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !saved.ContainerSysAdminEnabled || saved.Nickname != "legacy client" || saved.Role != user.Role {
		t.Fatal("attribute edit overwrote authorization or role")
	}
	// Revoking the grant leaves an already-created pod template unchanged.
	if err := AdminUpdateUserAttributes(t.Context(), user.Name, &model.UserAttribute{}, ptr.To(false)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, &pod) {
		t.Fatal("revoke mutated an existing template")
	}
	if err := ApplyUserJobCapabilities(t.Context(), user.ID, false, &pod); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(pod.Containers[0].SecurityContext.Capabilities.Add, "SYS_ADMIN") {
		t.Fatal("stale template grant survived revocation")
	}
}

func TestUserJobCapabilitiesFailClosed(t *testing.T) {
	db := jobPermissionDB(t)
	pod := corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}}
	before := pod.DeepCopy()
	if err := ApplyUserJobCapabilities(t.Context(), 999, true, &pod); !errors.Is(err, bizerr.Auth.Base) {
		t.Fatalf("missing user: %v", err)
	}
	if !reflect.DeepEqual(before, &pod) {
		t.Fatal("mutated pod before authorization")
	}
	if err := AdminUpdateUserAttributes(t.Context(), "missing", &model.UserAttribute{}, ptr.To(true)); !errors.Is(err, bizerr.NotFound.Base) {
		t.Fatalf("missing target: %v", err)
	}
	if err := db.Migrator().DropTable(&model.User{}); err != nil {
		t.Fatal(err)
	}
	if err := ApplyUserJobCapabilities(t.Context(), 1, true, &pod); !errors.Is(err, bizerr.Internal.Base) {
		t.Fatalf("database failure: %v", err)
	}
	if !reflect.DeepEqual(before, &pod) {
		t.Fatal("mutated pod on database failure")
	}
}

func TestUserJobSysAdminRequiresGrantAndOptIn(t *testing.T) {
	db := jobPermissionDB(t)
	user := model.User{Name: "opt-in", Space: "opt-in", Role: model.RoleUser, Status: model.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name                       string
		granted, requested, denied bool
	}{
		{"ungranted default", false, false, false},
		{"ungranted explicit request", false, true, true},
		{"granted default remains off", true, false, false},
		{"granted explicit request", true, true, false},
		{"revoked explicit request", false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := db.Model(&user).Update("container_sys_admin_enabled", tc.granted).Error; err != nil {
				t.Fatal(err)
			}
			// Imported/stale pod capabilities must never substitute for a new request.
			pod := corev1.PodSpec{Containers: []corev1.Container{{Name: "main", SecurityContext: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{Add: []corev1.Capability{"SYS_ADMIN", "IPC_LOCK"}},
			}}}}
			before := pod.DeepCopy()
			err := ApplyUserJobCapabilities(t.Context(), user.ID, tc.requested, &pod)
			if tc.denied {
				if !errors.Is(err, bizerr.Forbidden.Base) {
					t.Fatalf("want permission error, got %v", err)
				}
				if !reflect.DeepEqual(before, &pod) {
					t.Fatal("denied request changed template")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			caps := pod.Containers[0].SecurityContext.Capabilities.Add
			if slices.Contains(caps, "SYS_ADMIN") != tc.requested {
				t.Fatalf("unexpected capabilities: %v", caps)
			}
			if !slices.Contains(caps, "SYS_PTRACE") || !slices.Contains(caps, "IPC_LOCK") {
				t.Fatal("lost unrelated capabilities")
			}
		})
	}
}
