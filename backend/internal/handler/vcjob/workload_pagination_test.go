package vcjob

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
)

const workloadTestJobsTable = "jobs"

func TestBatchWorkloadsPageInDatabase(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	connection, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	connection.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = connection.Close() })
	for _, statement := range []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY, nickname TEXT, deleted_at DATETIME)",
		"CREATE TABLE accounts (id INTEGER PRIMARY KEY, nickname TEXT, deleted_at DATETIME)",
		"CREATE TABLE jobs (id INTEGER PRIMARY KEY, name TEXT, job_name TEXT, creation_timestamp DATETIME, running_timestamp DATETIME, completed_timestamp DATETIME, user_id INTEGER, account_id INTEGER, job_type TEXT DEFAULT 'custom', status TEXT DEFAULT 'Pending', deleted_at DATETIME)",
		"INSERT INTO jobs (id, job_name, creation_timestamp) VALUES (1, 'first', '2026-01-01'), (2, 'second', '2026-01-02'), (3, 'third', '2026-01-03')",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	query.SetDefault(db)
	var selected string
	if err := db.Callback().Query().After("gorm:query").Register("check-page", func(tx *gorm.DB) {
		sql := tx.Statement.SQL.String()
		if tx.Statement.Table == workloadTestJobsTable && !strings.Contains(sql, "count(") {
			selected = sql
		}
	}); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"/?days=-1&page=2&page_size=1&workload_kind=volcano-job", http.NoBody)
	manager := &VolcanojobMgr{}
	manager.listWorkloads(ctx, -1, jobListScope{})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data resputil.Page[WorkloadResp] `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Total != 3 || len(response.Data.Items) != 1 || response.Data.Items[0].JobName != "second" {
		t.Fatalf("unexpected page: %s", recorder.Body.String())
	}
	if !strings.Contains(selected, "LIMIT 1 OFFSET 1") {
		t.Fatalf("query loaded unbounded jobs: %s", selected)
	}
}

func mixedWorkloadFixture(t *testing.T) (*gorm.DB, *VolcanojobMgr) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	connection, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	connection.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = connection.Close() })
	if err := db.AutoMigrate(&model.SystemConfig{}); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY, nickname TEXT, deleted_at DATETIME)",
		"CREATE TABLE accounts (id INTEGER PRIMARY KEY, nickname TEXT, deleted_at DATETIME)",
		"CREATE TABLE jobs (id INTEGER PRIMARY KEY, name TEXT, job_name TEXT, creation_timestamp DATETIME, running_timestamp DATETIME, completed_timestamp DATETIME, user_id INTEGER, account_id INTEGER, job_type TEXT DEFAULT 'custom', status TEXT DEFAULT 'Pending', deleted_at DATETIME)",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	for day := 1; day <= 20; day++ {
		if err := db.Exec("INSERT INTO jobs (id,job_name,creation_timestamp) VALUES (?,?,?)",
			day, fmt.Sprintf("job-%d", day), time.Date(2026, 1, day, 0, 0, 0, 0, time.UTC)).Error; err != nil {
			t.Fatal(err)
		}
	}
	query.SetDefault(db)
	configService := service.NewConfigService(query.Use(db))
	if err := configService.SetKthenaInferenceEnabled(t.Context(), true); err != nil {
		t.Fatal(err)
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(kthena.ServingGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(kthena.ServingGVK.GroupVersion().WithKind("ModelServingList"), &unstructured.UnstructuredList{})
	manager := &VolcanojobMgr{configService: configService, workloadNamespace: "crater-workspace",
		client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			testModelServing("service-first", "1", "2", time.Date(2026, 1, 21, 0, 0, 0, 0, time.UTC)),
			testModelServing("service-middle", "1", "2", time.Date(2026, 1, 11, 12, 0, 0, 0, time.UTC)),
		).Build()}
	return db, manager
}

func TestMixedWorkloadsUseBoundedDatabaseWindow(t *testing.T) {
	db, manager := mixedWorkloadFixture(t)
	var selected string
	if err := db.Callback().Query().After("gorm:query").Register("check-mixed-page", func(tx *gorm.DB) {
		if tx.Statement.Table == workloadTestJobsTable && !strings.Contains(strings.ToLower(tx.Statement.SQL.String()), "count(") {
			selected = tx.Statement.SQL.String()
		}
	}); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		page   int
		want   []string
		window string
	}{
		{1, []string{"service-first", "job-20"}, "LIMIT 4"},
		{4, []string{"job-15", "job-14"}, "LIMIT 4 OFFSET 4"},
		{6, []string{"service-middle", "job-11"}, "LIMIT 4 OFFSET 8"},
		{20, []string{}, "LIMIT 4 OFFSET 36"},
	} {
		t.Run(fmt.Sprintf("page-%d", test.page), func(t *testing.T) {
			request := &jobListQuery{Page: test.page, PageSize: 2, Days: ptr.To(-1),
				sorts: []jobSort{{field: "createdAt", descending: true}, {field: "id", descending: true}}}
			rows, total, err := manager.findWorkloadPage(t.Context(), jobListScope{}, request, -1)
			if err != nil || total != 22 || len(rows) != len(test.want) {
				t.Fatalf("mixed page total=%d rows=%d error=%v, want total=22 rows=%d", total, len(rows), err, len(test.want))
			}
			for i, name := range test.want {
				if rows[i].JobName != name {
					t.Fatalf("row %d=%s want %s", i, rows[i].JobName, name)
				}
			}
			if !strings.Contains(selected, test.window) {
				t.Fatalf("unbounded or misplaced job query: %s; want %s", selected, test.window)
			}
		})
	}
}

func TestUnifiedJobRowsPreserveAdmissionPhaseAndNumericOrder(t *testing.T) {
	rows := []WorkloadResp{
		workloadFromJob(&model.Job{Model: gorm.Model{ID: 2}, JobName: "job-z"}),
		workloadFromJob(&model.Job{Model: gorm.Model{ID: 10}, JobName: "job-a", PodGroupPhase: scheduling.PodGroupInqueue}),
	}
	sortWorkloads(rows, []jobSort{{field: "id", descending: true}})
	if rows[0].JobName != "job-a" || rows[0].PodGroupPhase != string(scheduling.PodGroupInqueue) {
		t.Fatalf("unified row lost numeric ordering or admission phase: %+v", rows[0])
	}
}

func TestMixedWorkloadFacetsAggregateWithoutLoadingJobs(t *testing.T) {
	db, manager := mixedWorkloadFixture(t)
	if err := db.Callback().Query().After("gorm:query").Register("check-facet-aggregation", func(tx *gorm.DB) {
		sql := strings.ToLower(tx.Statement.SQL.String())
		if tx.Statement.Table == workloadTestJobsTable && !strings.Contains(sql, "count(") {
			t.Errorf("facet query loaded job rows: %s", sql)
		}
	}); err != nil {
		t.Fatal(err)
	}
	facets, err := manager.findWorkloadFacets(t.Context(), jobListScope{}, &jobListQuery{}, -1, false)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]map[string]int64{
		workloadFacetJobType: {"custom": 20, workloadJobTypeModelDeploy: 2},
		workloadFacetKind:    {workloadKindVolcanoJob: 20, workloadKindKthenaInference: 2},
		workloadFacetStatus:  {workloadStatusPending: 20, workloadStatusRunning: 2},
	}
	for facet, counts := range want {
		if len(facets[facet]) != len(counts) {
			t.Fatalf("facet %s=%+v, want %v", facet, facets[facet], counts)
		}
		for _, item := range facets[facet] {
			if counts[item.Value] != item.Count {
				t.Errorf("facet %s/%s count=%d, want %d", facet, item.Value, item.Count, counts[item.Value])
			}
		}
	}
}

func TestMixedWorkloadPagesMatchCompleteOrdering(t *testing.T) {
	db, manager := mixedWorkloadFixture(t)
	if err := db.Exec("UPDATE jobs SET name = ? WHERE id = ?", "Alpha", 1).Error; err != nil {
		t.Fatal(err)
	}
	var jobs []*model.Job
	if err := db.Find(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	inference, err := manager.findKthenaWorkloads(t.Context(), jobListScope{}, &jobListQuery{}, -1)
	if err != nil {
		t.Fatal(err)
	}
	all := append([]WorkloadResp{}, inference...)
	for _, job := range jobs {
		all = append(all, workloadFromJob(job))
	}
	for _, name := range []string{workloadSortName, workloadSortJobName, workloadSortStartedAt, workloadSortCompletedAt} {
		for _, descending := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s-desc-%t", name, descending), func(t *testing.T) {
				request := &jobListQuery{Page: 4, PageSize: 3,
					sorts: []jobSort{{field: name, descending: descending}, {field: workloadSortID, descending: true}}}
				want := append([]WorkloadResp{}, all...)
				sortWorkloads(want, request.sorts)
				rows, _, err := manager.findWorkloadPage(t.Context(), jobListScope{}, request, -1)
				if err != nil {
					t.Fatal(err)
				}
				if len(rows) != request.PageSize {
					t.Fatalf("page has %d rows, want %d", len(rows), request.PageSize)
				}
				for i := range rows {
					if rows[i].WorkloadID != want[request.offset()+i].WorkloadID {
						t.Errorf("row %d=%s, want %s", i, rows[i].WorkloadID, want[request.offset()+i].WorkloadID)
					}
				}
			})
		}
	}
}
