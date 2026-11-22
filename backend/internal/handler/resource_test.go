package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

func TestSyncResourceRetiresAndRestoresGPUClassification(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{IgnoreRelationshipsWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err = db.AutoMigrate(&model.Resource{}); err != nil {
		t.Fatal(err)
	}
	nodeClient := fake.NewSimpleClientset()
	mgr := &ResourceMgr{kubeClient: nodeClient}
	store := query.Use(db)
	originalQuery, originalResource := query.Q, query.Resource
	query.Q, query.Resource = store, &store.Resource
	t.Cleanup(func() { query.Q, query.Resource = originalQuery, originalResource })

	gpuType := model.ResourceTypeGPU
	gpu := model.Resource{ResourceName: "vendor.example/retired", Type: &gpuType, Amount: 2}
	if err = db.Create(&gpu).Error; err != nil {
		t.Fatal(err)
	}
	sync := func() {
		t.Helper()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/admin/resources/sync", http.NoBody)
		mgr.SyncResource(c)
		if w.Code != http.StatusOK {
			t.Fatalf("sync status=%d, body=%s", w.Code, w.Body.String())
		}
	}
	sync()
	var retired model.Resource
	if err = db.Unscoped().First(&retired, gpu.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !retired.DeletedAt.Valid || retired.Type == nil || *retired.Type != gpuType {
		t.Fatalf("sync lost retired GPU classification: %+v", retired)
	}

	_, err = nodeClient.CoreV1().Nodes().Create(t.Context(), &v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "restored-node"},
		Status:     v1.NodeStatus{Allocatable: v1.ResourceList{v1.ResourceName(gpu.ResourceName): resource.MustParse("4")}},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	sync()
	var active model.Resource
	if err = db.First(&active, gpu.ID).Error; err != nil {
		t.Fatal(err)
	}
	if active.Type == nil || *active.Type != gpuType || active.Amount != 4 {
		t.Fatalf("restored GPU lost identity or classification: %+v", active)
	}
}

func TestGetGPUVGPUResourcesExcludesRetiredVGPUAfterSync(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{IgnoreRelationshipsWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err = db.AutoMigrate(&model.Resource{}, &model.ResourceVGPU{}); err != nil {
		t.Fatal(err)
	}

	gpuType := model.ResourceTypeGPU
	gpu := model.Resource{ResourceName: "nvidia.com/gpu", Type: &gpuType, Amount: 4}
	vgpu := model.Resource{ResourceName: "nvidia.com/vgpu-retired", Amount: 2}
	if err = db.Create(&gpu).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&vgpu).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&model.ResourceVGPU{GPUResourceID: gpu.ID, VGPUResourceID: vgpu.ID}).Error; err != nil {
		t.Fatal(err)
	}

	// Keep the physical GPU on the node while removing the retired vGPU.
	nodeClient := fake.NewSimpleClientset(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "gpu-node"},
		Status: v1.NodeStatus{Allocatable: v1.ResourceList{
			v1.ResourceName(gpu.ResourceName): resource.MustParse("4"),
		}},
	})
	mgr := &ResourceMgr{kubeClient: nodeClient}
	store := query.Use(db)
	originalQuery, originalResource, originalVGPU := query.Q, query.Resource, query.ResourceVGPU
	query.Q, query.Resource, query.ResourceVGPU = store, &store.Resource, &store.ResourceVGPU
	t.Cleanup(func() {
		query.Q, query.Resource, query.ResourceVGPU = originalQuery, originalResource, originalVGPU
	})

	syncRec := httptest.NewRecorder()
	syncCtx, _ := gin.CreateTestContext(syncRec)
	syncCtx.Request = httptest.NewRequest(http.MethodPost, "/v1/admin/resources/sync", http.NoBody)
	mgr.SyncResource(syncCtx)
	if syncRec.Code != http.StatusOK {
		t.Fatalf("sync status=%d, body=%s", syncRec.Code, syncRec.Body.String())
	}

	getRec := httptest.NewRecorder()
	getCtx, _ := gin.CreateTestContext(getRec)
	getCtx.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(gpu.ID), 10)}}
	getCtx.Request = httptest.NewRequest(http.MethodGet, "/v1/admin/resources/"+strconv.FormatUint(uint64(gpu.ID), 10)+"/vgpu", http.NoBody)
	mgr.GetGPUVGPUResources(getCtx)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get vgpu status=%d, body=%s", getRec.Code, getRec.Body.String())
	}

	var resp struct {
		Data []model.ResourceVGPU `json:"data"`
	}
	if err = json.Unmarshal(getRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v (body=%s)", err, getRec.Body.String())
	}
	if len(resp.Data) != 0 {
		t.Fatalf("expected 0 active vgpu links after retiring vgpu, got %+v", resp.Data)
	}
}
