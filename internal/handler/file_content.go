package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"woopen/internal/model"

	"github.com/gin-gonic/gin"
)

const maxTextPreviewBytes int64 = 5 * 1024 * 1024

var errTextPreviewTooLarge = errors.New("文件超过 5 MiB 文本预览上限")

// HTTPDoer 是文本代理所需的最小 HTTP 客户端接口，便于替换和测试。
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// GetFileContent 通过服务端读取文件内容，避免浏览器直接请求云盘直链触发 CORS。
func (h *Handler) GetFileContent(c *gin.Context) {
	fid := c.Query("fid")
	if fid == "" {
		writeFileContentError(c, http.StatusBadRequest, "缺少 fid")
		return
	}

	content, err := h.loadFileContent(c.Request.Context(), fid)
	if err != nil {
		writeFileContentLoadError(c, err)
		return
	}

	writeFileContentResponse(c, content)
}

// GetShareFileContent 读取公开分享中的文本文件内容，保持分享密码校验逻辑。
func (h *Handler) GetShareFileContent(c *gin.Context) {
	code := c.Param("code")
	fileID := shareFileID(c)
	share, err := h.shareRepo.GetByCode(code)
	if err != nil || !share.IsActive {
		writeFileContentError(c, http.StatusNotFound, "分享不存在或已失效")
		return
	}

	if share.ExpireAt != nil && share.ExpireAt.Before(time.Now()) {
		writeFileContentError(c, http.StatusGone, "分享已过期")
		return
	}

	if share.Password != "" {
		if c.Query("pwd") != share.Password {
			if !h.checkSharePasswordAttempt(c) {
				return
			}
			h.recordSharePasswordFailure(c)
			writeFileContentError(c, http.StatusUnauthorized, "需要密码验证")
			return
		}
		h.resetSharePasswordFailures(c)
	}

	if fileID == "" || fileID == "0" {
		fileID = share.TargetID
	}

	content, err := h.loadFileContent(c.Request.Context(), fileID)
	if err != nil {
		writeFileContentLoadError(c, err)
		return
	}

	writeFileContentResponse(c, content)
}

func (h *Handler) loadFileContent(ctx context.Context, fileID string) ([]byte, error) {
	downloadURL, err := h.wopanClient.GetDownloadURL(fileID)
	if err != nil {
		return nil, fmt.Errorf("获取下载链接失败: %w", err)
	}
	return fetchRemoteFileContent(ctx, h.httpClient, downloadURL)
}

func fetchRemoteFileContent(ctx context.Context, client HTTPDoer, downloadURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建文件请求失败: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求云盘文件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("云盘返回异常状态: %s", resp.Status)
	}
	if resp.ContentLength > maxTextPreviewBytes {
		return nil, errTextPreviewTooLarge
	}

	content, err := io.ReadAll(io.LimitReader(resp.Body, maxTextPreviewBytes+1))
	if err != nil {
		return nil, fmt.Errorf("读取云盘响应失败: %w", err)
	}
	if int64(len(content)) > maxTextPreviewBytes {
		return nil, errTextPreviewTooLarge
	}
	return content, nil
}

func writeFileContentLoadError(c *gin.Context, err error) {
	if errors.Is(err, errTextPreviewTooLarge) {
		writeFileContentError(c, http.StatusRequestEntityTooLarge, err.Error())
		return
	}
	writeFileContentError(c, http.StatusBadGateway, "读取文件内容失败: "+err.Error())
}

func writeFileContentResponse(c *gin.Context, content []byte) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data:    gin.H{"content_base64": base64.StdEncoding.EncodeToString(content)},
	})
}

func writeFileContentError(c *gin.Context, status int, message string) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, model.APIResponse{Code: status, Message: message})
}
