package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/startvibecoding/mothx/internal/cron"
)

type cronAPIResponse struct {
	Enabled bool           `json:"enabled"`
	Running bool           `json:"running"`
	Path    string         `json:"path,omitempty"`
	Jobs    []cron.CronJob `json:"jobs"`
}

type cronJobRequest struct {
	Name      *string `json:"name,omitempty"`
	Prompt    *string `json:"prompt,omitempty"`
	Schedule  *string `json:"schedule,omitempty"`
	OneShot   *bool   `json:"oneshot,omitempty"`
	Mode      *string `json:"mode,omitempty"`
	WorkDir   *string `json:"workDir,omitempty"`
	A2ATarget *string `json:"a2aTarget,omitempty"`
	A2AToken  *string `json:"a2aToken,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
}

func (rt *channelRuntime) handleCron(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rt.writeCronStatus(w)
	case http.MethodPost:
		rt.handleCronCreate(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (rt *channelRuntime) handleCronByID(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/cron/"), "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cron job ID required"})
		return
	}

	switch r.Method {
	case http.MethodPatch, http.MethodPut:
		rt.handleCronUpdate(w, r, id)
	case http.MethodDelete:
		rt.handleCronDelete(w, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (rt *channelRuntime) writeCronStatus(w http.ResponseWriter) {
	jobs, err := rt.listCronJobs()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, cronAPIResponse{
		Enabled: rt.cronEnabled(),
		Running: rt.cronScheduler != nil && rt.cronScheduler.IsRunning(),
		Path:    rt.cronPath(),
		Jobs:    jobs,
	})
}

func (rt *channelRuntime) handleCronCreate(w http.ResponseWriter, r *http.Request) {
	store := rt.ensureCronStore()
	if store == nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "cron is disabled"})
		return
	}

	var req cronJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if req.Name == nil || strings.TrimSpace(*req.Name) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if req.Prompt == nil || strings.TrimSpace(*req.Prompt) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "prompt is required"})
		return
	}

	job := cron.CronJob{
		Name:     strings.TrimSpace(*req.Name),
		Prompt:   *req.Prompt,
		Enabled:  true,
		Mode:     "yolo",
		WorkDir:  rt.cfg.API.GetWorkDir(),
		Schedule: "",
	}
	if req.Enabled != nil {
		job.Enabled = *req.Enabled
	}
	if req.Mode != nil && strings.TrimSpace(*req.Mode) != "" {
		job.Mode = strings.TrimSpace(*req.Mode)
	}
	if req.WorkDir != nil {
		job.WorkDir = strings.TrimSpace(*req.WorkDir)
	}
	if req.Schedule != nil {
		job.Schedule = strings.TrimSpace(*req.Schedule)
	}
	if req.OneShot != nil {
		job.OneShot = *req.OneShot
	}
	if req.A2ATarget != nil {
		job.A2ATarget = strings.TrimSpace(*req.A2ATarget)
	}
	if req.A2AToken != nil {
		job.A2AToken = strings.TrimSpace(*req.A2AToken)
	}
	if err := normalizeCronJobSchedule(&job); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	created, err := store.Create(job)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"job": created})
}

func (rt *channelRuntime) handleCronUpdate(w http.ResponseWriter, r *http.Request, id string) {
	store := rt.ensureCronStore()
	if store == nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "cron is disabled"})
		return
	}
	job, err := store.Get(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	var req cronJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if req.Name != nil {
		job.Name = strings.TrimSpace(*req.Name)
	}
	if req.Prompt != nil {
		job.Prompt = *req.Prompt
	}
	if req.Schedule != nil {
		job.Schedule = strings.TrimSpace(*req.Schedule)
	}
	if req.OneShot != nil {
		job.OneShot = *req.OneShot
	}
	if req.Mode != nil {
		job.Mode = strings.TrimSpace(*req.Mode)
	}
	if req.WorkDir != nil {
		job.WorkDir = strings.TrimSpace(*req.WorkDir)
	}
	if req.A2ATarget != nil {
		job.A2ATarget = strings.TrimSpace(*req.A2ATarget)
	}
	if req.A2AToken != nil {
		job.A2AToken = strings.TrimSpace(*req.A2AToken)
	}
	if req.Enabled != nil {
		job.Enabled = *req.Enabled
	}
	if strings.TrimSpace(job.Name) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if strings.TrimSpace(job.Prompt) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "prompt is required"})
		return
	}
	if err := normalizeCronJobSchedule(job); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := store.Update(*job); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job": job})
}

func (rt *channelRuntime) handleCronDelete(w http.ResponseWriter, id string) {
	store := rt.ensureCronStore()
	if store == nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "cron is disabled"})
		return
	}
	if err := store.Delete(id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "deleted": true})
}

func (rt *channelRuntime) listCronJobs() ([]cron.CronJob, error) {
	store := rt.ensureCronStore()
	if store == nil {
		return []cron.CronJob{}, nil
	}
	jobs, err := store.List()
	if err != nil {
		return nil, err
	}
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].CreatedAt.Equal(jobs[j].CreatedAt) {
			return jobs[i].ID < jobs[j].ID
		}
		return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
	})
	return jobs, nil
}

func (rt *channelRuntime) ensureCronStore() cron.CronStore {
	if rt == nil || rt.cfg == nil || !rt.cronEnabled() {
		return nil
	}
	hCfg := buildConfigFromServeConfig(rt.cfg)
	nextPath := cronStorePath(hCfg)
	if rt.cronStore == nil || rt.cronStorePath != nextPath {
		rt.stopCronScheduler()
		rt.cronStorePath = nextPath
		rt.cronStore = buildCronStore(hCfg)
	}
	return rt.cronStore
}

func (rt *channelRuntime) cronEnabled() bool {
	return rt != nil && rt.cfg != nil && rt.cfg.Features.Cron
}

func (rt *channelRuntime) cronPath() string {
	if rt == nil {
		return ""
	}
	if rt.cronStorePath != "" {
		return rt.cronStorePath
	}
	if rt.cfg == nil {
		return ""
	}
	return cronStorePath(buildConfigFromServeConfig(rt.cfg))
}

func normalizeCronJobSchedule(job *cron.CronJob) error {
	if job == nil {
		return fmt.Errorf("cron job required")
	}
	if job.Mode == "" {
		job.Mode = "yolo"
	}
	if job.Mode != "agent" && job.Mode != "yolo" {
		return fmt.Errorf("mode must be agent or yolo")
	}

	next, isOneShot, err := cron.ParseSchedule(job.Schedule, time.Now())
	if err != nil {
		return err
	}
	if job.OneShot || isOneShot {
		job.OneShot = true
		job.NextRun = time.Time{}
		return nil
	}
	job.NextRun = next
	return nil
}
