package handler

import (
	"net/http"
	"sync"
	"time"
	"woopen/internal/model"

	"github.com/gin-gonic/gin"
)

// uploadProgressEntry tracks server-side (server->wopan) progress for one upload.
// It is intentionally kept in-memory (best-effort) to avoid adding persistence.
type uploadProgressEntry struct {
	mu        sync.Mutex
	fileName  string
	parentID  string
	total     int64
	uploaded  int64
	status    string // uploading | success | failed
	errMsg    string
	createdAt time.Time
	updatedAt time.Time
}

func (e *uploadProgressEntry) snapshot() (fileName, parentID string, uploaded, total int64, status, errMsg string, updatedAt time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.fileName, e.parentID, e.uploaded, e.total, e.status, e.errMsg, e.updatedAt
}

func (e *uploadProgressEntry) setUploading(fileName, parentID string, total int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	now := time.Now()
	e.fileName = fileName
	e.parentID = parentID
	e.total = total
	e.uploaded = 0
	e.status = "uploading"
	e.errMsg = ""
	e.createdAt = now
	e.updatedAt = now
}

func (e *uploadProgressEntry) setProgress(uploaded, total int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Prefer SDK-provided total when present; otherwise keep the original header size.
	if total > 0 {
		e.total = total
	}
	if uploaded >= 0 {
		e.uploaded = uploaded
	}
	e.updatedAt = time.Now()
}

func (e *uploadProgressEntry) setDone(status string, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status = status
	if err != nil {
		e.errMsg = err.Error()
	}
	e.updatedAt = time.Now()
}

// GetUploadProgress returns the last known progress snapshot for an upload_id.
// It always responds 200 to avoid spamming global axios error toasts on polling.
func (h *Handler) GetUploadProgress(c *gin.Context) {
	uploadID := c.Query("upload_id")
	if uploadID == "" {
		c.JSON(http.StatusOK, model.APIResponse{
			Code:    0,
			Message: "success",
			Data: gin.H{
				"found": false,
			},
		})
		return
	}

	raw, ok := h.uploadProgress.Load(uploadID)
	if !ok {
		c.JSON(http.StatusOK, model.APIResponse{
			Code:    0,
			Message: "success",
			Data: gin.H{
				"found": false,
			},
		})
		return
	}

	entry, _ := raw.(*uploadProgressEntry)
	if entry == nil {
		c.JSON(http.StatusOK, model.APIResponse{
			Code:    0,
			Message: "success",
			Data: gin.H{
				"found": false,
			},
		})
		return
	}

	fileName, parentID, uploaded, total, status, errMsg, updatedAt := entry.snapshot()
	percent := 0
	if total > 0 && uploaded > 0 {
		percent = int((uploaded * 100) / total)
		if percent > 100 {
			percent = 100
		}
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"found":      true,
			"upload_id":  uploadID,
			"file_name":  fileName,
			"parent_id":  parentID,
			"uploaded":   uploaded,
			"total":      total,
			"percent":    percent,
			"status":     status,
			"error":      errMsg,
			"updated_at": updatedAt.Unix(),
		},
	})
}

