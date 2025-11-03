// Usage: CRATER_DEBUG_CONFIG_PATH=${PWD}/etc/debug-config-actgpu.yaml go run hack/export_statistic.go
package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/monitor"
)

func main() {
	db := query.GetDB()

	// Get all jobs with related user and account
	var jobs []model.Job
	// InClude deleted jobs
	if err := db.Preload("User").Preload("Account").Unscoped().Order("id DESC").Find(&jobs).Error; err != nil {
		panic(fmt.Errorf("failed to fetch jobs: %w", err))
	}

	// Create CSV file
	file, err := os.Create("jobs_export.csv")
	if err != nil {
		panic(fmt.Errorf("failed to create CSV file: %w", err))
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	headers := []string{
		"ID", "Name", "JobName", "UserID", "UserName", "AccountID", "AccountName",
		"JobType", "Status", "CreationTimestamp", "RunningTimestamp", "CompletedTimestamp",
		"LockedTimestamp", "Nodes", "CPU", "Memory", "GPUCount", "GPUModel",
		"KeepWhenLowResourceUsage", "Reminded",
		"AlertEnabled",
		// ProfileData fields
		"CPURequest", "CPULimit", "MemRequest", "MemLimit",
		"CPUUsageAvg", "CPUUsageMax", "CPUUsageStd",
		"CPUMemAvg", "CPUMemMax", "CPUMemStd",
		"GPUUtilAvg", "GPUUtilMax", "GPUUtilStd",
		"SMActiveAvg", "SMActiveMax", "SMActiveStd",
		"SMOccupancyAvg", "SMOccupancyMax", "SMOccupancyStd",
		"DramUtilAvg", "DramUtilMax", "DramUtilStd",
		"MemCopyUtilAvg", "MemCopyUtilMax", "MemCopyUtilStd",
		"PCIETxAvg", "PCIETxMax", "PCIERxAvg", "PCIERxMax",
		"GPUMemTotal", "GPUMemMax", "GPUMemAvg", "GPUMemStd",
		"TensorActiveAvg", "TensorActiveMax", "TensorActiveStd",
		"Fp64ActiveAvg", "Fp64ActiveMax", "Fp64ActiveStd",
		"Fp32ActiveAvg", "Fp32ActiveMax", "Fp32ActiveStd",
		"DramActiveAvg", "DramActiveMax", "DramActiveStd",
		"Fp16ActiveAvg", "Fp16ActiveMax", "Fp16ActiveStd",
	}
	if err := writer.Write(headers); err != nil {
		panic(fmt.Errorf("failed to write CSV header: %w", err))
	}

	// Write each job
	for i := range jobs {
		job := &jobs[i]
		record, err := jobToCSVRecord(job)
		if err != nil {
			fmt.Printf("Warning: failed to process job ID %d: %v\n", job.ID, err)
			continue
		}

		if err := writer.Write(record); err != nil {
			panic(fmt.Errorf("failed to write CSV record: %w", err))
		}
	}

	fmt.Println("Successfully exported jobs to jobs_export.csv")
}

func jobToCSVRecord(job *model.Job) ([]string, error) {
	// Convert JSON fields to strings
	nodes, err := json.Marshal(job.Nodes.Data())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal nodes: %w", err)
	}

	resources := job.Resources.Data()
	cpu := resources.Cpu()
	memory := resources.Memory()
	// find nvidia.com/ prefix from resources, such as nvidia.com/v100, nvidia.com/a100
	gpuCount := 0
	gpuModel := ""
	for k, v := range resources {
		if strings.Contains(k.String(), "nvidia.com/") {
			gpuCount += int(v.Value())
			gpuModel = strings.TrimPrefix(k.String(), "nvidia.com/")
			break
		}
	}

	// Format timestamps
	formatTime := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format(time.RFC3339)
	}

	// Helper function to format float pointer
	formatFloat := func(f *float32) string {
		if f == nil {
			return ""
		}
		return fmt.Sprintf("%f", *f)
	}

	// Extract ProfileData values with nil checks
	var profileData *monitor.ProfileData
	if job.ProfileData != nil && job.ProfileData.Data() != nil {
		profileData = job.ProfileData.Data()
	} else {
		profileData = &monitor.ProfileData{}
	}

	record := []string{
		fmt.Sprintf("%d", job.ID),
		job.Name,
		job.JobName,
		fmt.Sprintf("%d", job.UserID),
		job.User.Name,
		fmt.Sprintf("%d", job.AccountID),
		job.Account.Name,
		string(job.JobType),
		string(job.Status),
		formatTime(job.CreationTimestamp),
		formatTime(job.RunningTimestamp),
		formatTime(job.CompletedTimestamp),
		formatTime(job.LockedTimestamp),
		string(nodes),
		fmt.Sprintf("%d", cpu.Value()),
		fmt.Sprintf("%d", memory.Value()),
		fmt.Sprintf("%d", gpuCount),
		gpuModel,
		fmt.Sprintf("%t", job.KeepWhenLowResourceUsage),
		fmt.Sprintf("%t", job.Reminded),
		fmt.Sprintf("%t", job.AlertEnabled),
		// ProfileData fields
		formatFloat(profileData.CPURequest),
		formatFloat(profileData.CPULimit),
		formatFloat(profileData.MemRequest),
		formatFloat(profileData.MemLimit),
		formatFloat(profileData.CPUUsageAvg),
		formatFloat(profileData.CPUUsageMax),
		formatFloat(profileData.CPUUsageStd),
		formatFloat(profileData.CPUMemAvg),
		formatFloat(profileData.CPUMemMax),
		formatFloat(profileData.CPUMemStd),
		formatFloat(profileData.GPUUtilAvg),
		formatFloat(profileData.GPUUtilMax),
		formatFloat(profileData.GPUUtilStd),
		formatFloat(profileData.SMActiveAvg),
		formatFloat(profileData.SMActiveMax),
		formatFloat(profileData.SMActiveStd),
		formatFloat(profileData.SMOccupancyAvg),
		formatFloat(profileData.SMOccupancyMax),
		formatFloat(profileData.SMOccupancyStd),
		formatFloat(profileData.DramUtilAvg),
		formatFloat(profileData.DramUtilMax),
		formatFloat(profileData.DramUtilStd),
		formatFloat(profileData.MemCopyUtilAvg),
		formatFloat(profileData.MemCopyUtilMax),
		formatFloat(profileData.MemCopyUtilStd),
		formatFloat(profileData.PCIETxAvg),
		formatFloat(profileData.PCIETxMax),
		formatFloat(profileData.PCIERxAvg),
		formatFloat(profileData.PCIERxMax),
		formatFloat(profileData.GPUMemTotal),
		formatFloat(profileData.GPUMemMax),
		formatFloat(profileData.GPUMemAvg),
		formatFloat(profileData.GPUMemStd),
		formatFloat(profileData.TensorActiveAvg),
		formatFloat(profileData.TensorActiveMax),
		formatFloat(profileData.TensorActiveStd),
		formatFloat(profileData.Fp64ActiveAvg),
		formatFloat(profileData.Fp64ActiveMax),
		formatFloat(profileData.Fp64ActiveStd),
		formatFloat(profileData.Fp32ActiveAvg),
		formatFloat(profileData.Fp32ActiveMax),
		formatFloat(profileData.Fp32ActiveStd),
		formatFloat(profileData.DramActiveAvg),
		formatFloat(profileData.DramActiveMax),
		formatFloat(profileData.DramActiveStd),
		formatFloat(profileData.Fp16ActiveAvg),
		formatFloat(profileData.Fp16ActiveMax),
		formatFloat(profileData.Fp16ActiveStd),
	}
	return record, nil
}
