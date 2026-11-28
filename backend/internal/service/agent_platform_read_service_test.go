package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/raids-lab/crater/dao/model"
)

//nolint:gocyclo // The table-driven cases intentionally cover all authorization and paging branches.
func TestAgentListJobsScopesAndLimits(t *testing.T) {
	db := newAgentConversationTestDB(t)
	for _, sql := range []string{
		`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, nickname TEXT, attributes TEXT, deleted_at DATETIME)`,
		`CREATE TABLE accounts (id INTEGER PRIMARY KEY, deleted_at DATETIME)`,
		`CREATE TABLE jobs (id INTEGER PRIMARY KEY, user_id INTEGER, account_id INTEGER, job_name TEXT,
			name TEXT, status TEXT, job_type TEXT, nodes TEXT, resources TEXT,
			creation_timestamp DATETIME, running_timestamp DATETIME, completed_timestamp DATETIME, deleted_at DATETIME)`,
		`INSERT INTO users (id, name, nickname) VALUES (1, 'alice', '周霄'), (2, 'bob', '周奕航')`,
		`INSERT INTO accounts (id) VALUES (1), (2)`,
		`INSERT INTO jobs (id, user_id, account_id, job_name, name, status, creation_timestamp) VALUES
            (1, 1, 1, 'job-a', 'example', 'Failed', '2026-01-01'),
            (2, 1, 1, 'job-b', 'example', 'Failed', '2026-01-01'),
            (3, 1, 2, 'job-c', 'example', 'Failed', '2026-01-02'),
            (4, 2, 1, 'job-d', 'example', 'Failed', '2026-01-03')`,
	} {
		mustAgentTestNoError(t, db.Exec(sql).Error)
	}
	s := NewAgentPlatformReadService(db, nil)
	options := AgentJobListOptions{Identity: AgentIdentityUser, UserID: 1, AccountID: 1, PageSize: 1, IncludeTotal: true}
	page, err := s.ListJobs(context.Background(), &options)
	mustAgentTestNoError(t, err)
	if page.Total == nil || *page.Total != 2 || !page.HasMore || len(page.Items) != 1 || page.Items[0].JobName != "job-b" {
		t.Fatalf("unexpected scoped, bounded page: %+v", page)
	}
	options.Search = "job-d"
	page, err = s.ListJobs(context.Background(), &options)
	mustAgentTestNoError(t, err)
	if page.Total == nil || *page.Total != 0 {
		t.Fatal("search escaped the owner/account restriction")
	}
	options.Username = "bob"
	_, err = s.ListJobs(context.Background(), &options)
	mustAgentTestError(t, err)
	options.Identity, options.Search = AgentIdentityAdmin, ""
	page, err = s.ListJobs(context.Background(), &options)
	mustAgentTestNoError(t, err)
	if page.Total == nil || *page.Total != 1 || page.Items[0].JobName != "job-d" {
		t.Fatalf("admin owner filter failed: %+v", page)
	}
	options.Username = "周霄"
	page, err = s.ListJobs(context.Background(), &options)
	mustAgentTestNoError(t, err)
	if page.Total == nil || *page.Total != 3 || page.Items[0].Username != "alice" || page.Items[0].Nickname != "周霄" {
		t.Fatalf("admin nickname filter failed: %+v", page)
	}
	mustAgentTestNoError(t, db.Exec(`INSERT INTO users (id, name, nickname) VALUES (3, 'charlie', '周霄')`).Error)
	_, err = s.ListJobs(context.Background(), &options)
	mustAgentTestError(t, err)
	options.Page = 10001
	_, err = s.ListJobs(context.Background(), &options)
	mustAgentTestError(t, err)
}

func TestAgentListJobsResolvesLoginAndBothNicknameFields(t *testing.T) {
	db := newAgentConversationTestDB(t)
	for _, sql := range []string{
		`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, nickname TEXT, attributes TEXT, deleted_at DATETIME)`,
		`CREATE TABLE jobs (id INTEGER PRIMARY KEY, user_id INTEGER, account_id INTEGER, job_name TEXT,
			name TEXT, status TEXT, job_type TEXT, nodes TEXT, resources TEXT,
			creation_timestamp DATETIME, running_timestamp DATETIME, completed_timestamp DATETIME, deleted_at DATETIME)`,
		`INSERT INTO users (id, name, nickname, attributes) VALUES
			(1, 'zhouxiao', 'Zhou Xiao', '{"nickname":"周霄"}'),
			(2, 'bob', 'Captain', '{"nickname":"旧昵称"}'),
			(3, '中文账号', '小王', '{"nickname":"小王"}')`,
		`INSERT INTO jobs (id, user_id, account_id, job_name, name, status, creation_timestamp) VALUES
			(1, 1, 1, 'job-a', 'example-a', 'Failed', '2026-01-01'),
			(2, 2, 1, 'job-b', 'example-b', 'Failed', '2026-01-02'),
			(3, 3, 1, 'job-c', 'example-c', 'Failed', '2026-01-03')`,
	} {
		mustAgentTestNoError(t, db.Exec(sql).Error)
	}
	s := NewAgentPlatformReadService(db, nil)
	for _, tc := range []struct {
		name     string
		username string
		jobName  string
	}{
		{name: "login username", username: "zhouxiao", jobName: "job-a"},
		{name: "English display nickname", username: "zhou xiao", jobName: "job-a"},
		{name: "JSON nickname", username: "周霄", jobName: "job-a"},
		{name: "other English nickname", username: "captain", jobName: "job-b"},
		{name: "Chinese login username", username: "中文账号", jobName: "job-c"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			options := AgentJobListOptions{Identity: AgentIdentityAdmin, Username: tc.username}
			page, err := s.ListJobs(context.Background(), &options)
			mustAgentTestNoError(t, err)
			if len(page.Items) != 1 || page.Items[0].JobName != tc.jobName {
				t.Fatalf("unexpected jobs for %q: %+v", tc.username, page)
			}
		})
	}
	mustAgentTestNoError(t, db.Exec(`INSERT INTO users (id, name, nickname, attributes)
		VALUES (4, 'other', '周霄', '{"nickname":"other"}')`).Error)
	options := AgentJobListOptions{Identity: AgentIdentityAdmin, Username: "周霄"}
	_, err := s.ListJobs(context.Background(), &options)
	mustAgentTestError(t, err)
}

func TestAgentListJobsWithoutTotal(t *testing.T) {
	db := newAgentConversationTestDB(t)
	for _, sql := range []string{
		`CREATE TABLE jobs (id INTEGER PRIMARY KEY, user_id INTEGER, account_id INTEGER, job_name TEXT,
			name TEXT, status TEXT, job_type TEXT, nodes TEXT, resources TEXT,
			creation_timestamp DATETIME, running_timestamp DATETIME, completed_timestamp DATETIME, deleted_at DATETIME)`,
		`INSERT INTO jobs (id, user_id, account_id, job_name, name, status, creation_timestamp) VALUES
			(1, 1, 1, 'job-a', 'example', 'Failed', '2026-01-01'),
			(2, 1, 1, 'job-b', 'example', 'Failed', '2026-01-02')`,
	} {
		mustAgentTestNoError(t, db.Exec(sql).Error)
	}
	s := NewAgentPlatformReadService(db, nil)
	options := AgentJobListOptions{Identity: AgentIdentityUser, UserID: 1, AccountID: 1, PageSize: 1}
	page, err := s.ListJobs(context.Background(), &options)
	mustAgentTestNoError(t, err)
	if page.Total != nil || !page.HasMore || len(page.Items) != 1 || page.Items[0].JobName != "job-b" {
		t.Fatalf("unexpected page without total count: %+v", page)
	}
}

func TestAgentNodeEstimatesIncludeInitAndOverhead(t *testing.T) {
	node := &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "worker"}, Status: v1.NodeStatus{
		Capacity:    v1.ResourceList{v1.ResourceCPU: resource.MustParse("8"), "nvidia.com/gpu": resource.MustParse("4")},
		Allocatable: v1.ResourceList{v1.ResourceCPU: resource.MustParse("8"), "nvidia.com/gpu": resource.MustParse("3")},
		Conditions:  []v1.NodeCondition{{Type: v1.NodeReady, Status: v1.ConditionTrue}},
	}}
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "work", Namespace: "test"}, Spec: v1.PodSpec{
		NodeName: "worker",
		Containers: []v1.Container{{Name: "main", Resources: v1.ResourceRequirements{
			Requests: v1.ResourceList{v1.ResourceCPU: resource.MustParse("1")},
		}}},
		InitContainers: []v1.Container{{Name: "init", Resources: v1.ResourceRequirements{
			Requests: v1.ResourceList{v1.ResourceCPU: resource.MustParse("4")},
		}}},
		Overhead: v1.ResourceList{v1.ResourceCPU: resource.MustParse("1")},
	}}
	s := NewAgentPlatformReadService(nil, fake.NewSimpleClientset(node, pod))
	capacity, err := s.GetNodeCapacity(context.Background(), "worker")
	mustAgentTestNoError(t, err)
	if capacity.Requested["cpu"] != "5" || capacity.EstimatedAvailable["cpu"] != "3" ||
		capacity.Accelerators["nvidia.com/gpu"] != "4" {
		t.Fatalf("incorrect resource accounting: %+v", capacity)
	}
	node.Spec.Unschedulable = true
	_, err = s.kubeClient.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
	mustAgentTestNoError(t, err)
	cluster, err := s.GetClusterCapacity(context.Background())
	mustAgentTestNoError(t, err)
	if cluster.SchedulableNodes != 0 || cluster.Accelerators["nvidia.com/gpu"] != "4" {
		t.Fatalf("physical capacity was confused with schedulable capacity: %+v", cluster)
	}
}

func TestAgentActiveJobsIncludeAllUnfinishedPhases(t *testing.T) {
	db := newAgentConversationTestDB(t)
	mustAgentTestNoError(t, db.Exec(`CREATE TABLE jobs (id INTEGER PRIMARY KEY, user_id INTEGER,
		account_id INTEGER, job_name TEXT, name TEXT, status TEXT, job_type TEXT, nodes TEXT, resources TEXT,
		creation_timestamp DATETIME, running_timestamp DATETIME, completed_timestamp DATETIME, deleted_at DATETIME)`).Error)
	active := []string{"", "Prequeue", "Pending", "Running", "Restarting", "Aborting", "Completing", "Terminating"}
	terminal := []string{"Completed", "Failed", "Aborted", "Terminated", "Deleted", "Freed", "Cancelled"} //nolint:misspell // Platform phase.
	for index, status := range append(append([]string{}, active...), terminal...) {
		mustAgentTestNoError(t, db.Exec(`INSERT INTO jobs (id,user_id,account_id,job_name,status)
			VALUES (?,1,1,?,?)`, index+1, status, status).Error)
	}
	s := NewAgentPlatformReadService(db, nil)
	options := AgentJobListOptions{Identity: AgentIdentityUser, UserID: 1, AccountID: 1, Scope: AgentJobScopeActive}
	page, err := s.ListJobs(context.Background(), &options)
	mustAgentTestNoError(t, err)
	if len(page.Items) != len(active) || page.Scope != AgentJobScopeActive || page.HasMore {
		t.Fatalf("unexpected active scope: %+v", page)
	}
	options.Statuses = []string{"Failed"}
	page, err = s.ListJobs(context.Background(), &options)
	mustAgentTestNoError(t, err)
	if len(page.Items) != 0 || page.HasMore {
		t.Fatal("explicit status filter escaped the active scope")
	}
	options.Scope = AgentJobScopeAll
	page, err = s.ListJobs(context.Background(), &options)
	mustAgentTestNoError(t, err)
	if len(page.Items) != 1 || page.Items[0].Status != "Failed" {
		t.Fatal("explicit history filter did not return the failed job")
	}
}

func TestAgentJobSummaryOmitsAbsentTimes(t *testing.T) {
	t.Parallel()
	now := time.Now()
	items := agentJobSummaries([]model.Job{{}, {RunningTimestamp: now, CompletedTimestamp: now}}, AgentIdentityUser)
	data, err := json.Marshal(items[0])
	mustAgentTestNoError(t, err)
	if strings.Contains(string(data), "startedAt") || strings.Contains(string(data), "completedAt") {
		t.Fatalf("zero timestamps were exposed: %s", data)
	}
	if items[1].StartedAt == nil || !items[1].StartedAt.Equal(now) || items[1].CompletedAt == nil {
		t.Fatal("valid timestamps were lost")
	}
}
