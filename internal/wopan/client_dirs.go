package wopan

import (
	"context"
	"fmt"
	"time"
	"woopen/internal/model"

	"github.com/OpenListTeam/wopan-sdk-go"
	"resty.dev/v3"
)

// CreateDirectory 创建目录
func (c *Client) CreateDirectory(parentID, dirName string) (*model.FileInfo, error) {
	return c.CreateDirectoryCtx(context.Background(), parentID, dirName)
}

// CreateDirectoryCtx 创建目录（支持 context）
func (c *Client) CreateDirectoryCtx(ctx context.Context, parentID, dirName string) (*model.FileInfo, error) {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()

	client, err := c.getMetaClient()
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

	result, err := c.createDirectoryLocked(client, createDirParams{
		ctx:      ctx,
		parentID: parentID,
		dirName:  dirName,
	})
	if err != nil && isLoginFailed(err) {
		if refreshErr := client.RefreshToken(); refreshErr != nil {
			return nil, fmt.Errorf("刷新Token失败: %w", refreshErr)
		}
		result, err = c.createDirectoryLocked(client, createDirParams{
			ctx:      ctx,
			parentID: parentID,
			dirName:  dirName,
		})
	}
	return result, err
}

type createDirParams struct {
	ctx      context.Context
	parentID string
	dirName  string
}

// createDirectoryLocked 内部创建目录方法（需要持有锁）
func (c *Client) createDirectoryLocked(client *wopan.WoClient, params createDirParams) (*model.FileInfo, error) {
	// OpenList 在个人云创建目录时也会传入初始化阶段取得的默认家庭空间 ID。
	result, err := client.CreateDirectory(wopan.SpaceTypePersonal, params.parentID, params.dirName, c.getDefaultFamilyID(), func(req *resty.Request) {
		req.SetContext(params.ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	fileInfo := &model.FileInfo{
		ID:       result.Id,
		FID:      params.parentID,
		Name:     params.dirName,
		Size:     0,
		IsDir:    true,
		ModTime:  time.Now(),
		ParentID: params.parentID,
	}

	return fileInfo, nil
}
