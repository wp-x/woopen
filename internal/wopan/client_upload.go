package wopan

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"
	"woopen/internal/model"

	"github.com/OpenListTeam/wopan-sdk-go"
)

// UploadFile 上传文件到指定目录
func (c *Client) UploadFile(ctx context.Context, parentID, fileName, contentType string, size int64, content io.Reader, onProgress func(uploaded, total int64)) (*model.FileInfo, error) {
	c.uploadMu.Lock()
	defer c.uploadMu.Unlock()

	client, err := c.getUploadClient()
	if err != nil {
		return nil, err
	}
	c.syncClientTokens(client)

	// 如果没有指定父目录ID，使用根目录
	if parentID == "" {
		parentID = c.getRootFolderID()
	}
	if parentID == "" {
		parentID = "0"
	}

	// 调用SDK的Upload2C方法
	result, err := c.uploadFileLocked(client, uploadParams{
		ctx:         ctx,
		parentID:    parentID,
		fileName:    fileName,
		contentType: contentType,
		size:        size,
		content:     content,
		onProgress:  onProgress,
	})
	if err != nil && isLoginFailed(err) {
		// Token过期，刷新后重试
		if refreshErr := client.RefreshToken(); refreshErr != nil {
			return nil, fmt.Errorf("刷新Token失败: %w", refreshErr)
		}
		result, err = c.uploadFileLocked(client, uploadParams{
			ctx:         ctx,
			parentID:    parentID,
			fileName:    fileName,
			contentType: contentType,
			size:        size,
			content:     content,
			onProgress:  onProgress,
		})
	}
	return result, err
}

type uploadParams struct {
	ctx         context.Context
	parentID    string
	fileName    string
	contentType string
	size        int64
	content     io.Reader
	onProgress  func(uploaded, total int64)
}

// uploadFileLocked 内部上传文件方法（需要持有锁）
func (c *Client) uploadFileLocked(client *wopan.WoClient, params uploadParams) (*model.FileInfo, error) {
	contentType := params.contentType
	size := params.size

	// 将 io.Reader 转换为 io.ReadSeeker
	var seeker io.ReadSeeker
	if rs, ok := params.content.(io.ReadSeeker); ok {
		seeker = rs
	} else {
		// 如果不是 ReadSeeker，读取全部内容到内存
		// 说明：Web/API 上传通常是 multipart.File（实现 Seek），不会走到这里。
		data, err := io.ReadAll(params.content)
		if err != nil {
			return nil, fmt.Errorf("读取文件内容失败: %w", err)
		}
		seeker = bytes.NewReader(data)
		size = int64(len(data))
	}

	// SDK 期望从头开始读取
	if _, err := seeker.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("重置上传流失败: %w", err)
	}

	// ContentType 允许为空；为空时让服务器/SDK 兜底
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 构建上传文件参数
	uploadFile := wopan.Upload2CFile{
		Name:        params.fileName,
		Size:        size,
		Content:     seeker,
		ContentType: contentType,
	}

	// 构建上传选项（RetryTimes 让 SDK 在分片级别自动重试，解决首次连接 EOF 问题）
	uploadOption := wopan.Upload2COption{
		Ctx:        params.ctx,
		RetryTimes: 2,
		OnProgress: func(uploaded, total int64) {
			if params.onProgress != nil {
				params.onProgress(uploaded, total)
			}
		},
	}

	fileID, err := client.Upload2C(wopan.SpaceTypePersonal, uploadFile, params.parentID, "", uploadOption)
	if err != nil {
		return nil, fmt.Errorf("上传文件失败: %w", err)
	}

	// 转换为内部文件信息格式
	// Upload2C 返回的就是文件的 fid（下载/直连用它），不是普通 id
	fileInfo := &model.FileInfo{
		ID:       fileID,
		FID:      fileID,
		Name:     params.fileName,
		Size:     size,
		IsDir:    false,
		ModTime:  time.Now(),
		ParentID: params.parentID,
	}

	return fileInfo, nil
}
