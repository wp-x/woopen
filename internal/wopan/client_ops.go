package wopan

import (
	"context"
	"fmt"

	"github.com/OpenListTeam/wopan-sdk-go"
	"resty.dev/v3"
)

// Delete 删除文件/目录（个人云）
func (c *Client) Delete(ctx context.Context, isDir bool, id string) error {
	dirList, fileList := operationIDs(isDir, id)
	return c.DeleteBatch(ctx, dirList, fileList)
}

// DeleteBatch 批量删除文件/目录（个人云）
func (c *Client) DeleteBatch(ctx context.Context, dirList, fileList []string) error {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()

	client, err := c.getMetaClient()
	if err != nil {
		return err
	}
	c.syncClientTokens(client)

	params := deleteParams{ctx: ctx, dirList: dirList, fileList: fileList}
	err = c.deleteLocked(client, params)
	if err != nil && isLoginFailed(err) {
		if refreshErr := c.refreshClientLocked(client); refreshErr != nil {
			return refreshErr
		}
		err = c.deleteLocked(client, params)
	}
	return err
}

type deleteParams struct {
	ctx      context.Context
	dirList  []string
	fileList []string
}

func (c *Client) deleteLocked(client *wopan.WoClient, params deleteParams) error {
	return client.DeleteFile(wopan.SpaceTypePersonal, params.dirList, params.fileList, func(req *resty.Request) {
		req.SetContext(params.ctx)
	})
}

// Rename 重命名文件/目录（个人云）
func (c *Client) Rename(ctx context.Context, isDir bool, id, newName string) error {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()

	client, err := c.getMetaClient()
	if err != nil {
		return err
	}
	c.syncClientTokens(client)

	_type := 1
	if isDir {
		_type = 0
	}

	err = c.renameLocked(client, renameParams{
		ctx:  ctx,
		typ:  _type,
		id:   id,
		name: newName,
	})
	if err != nil && isLoginFailed(err) {
		if refreshErr := c.refreshClientLocked(client); refreshErr != nil {
			return refreshErr
		}
		err = c.renameLocked(client, renameParams{
			ctx:  ctx,
			typ:  _type,
			id:   id,
			name: newName,
		})
	}
	return err
}

type renameParams struct {
	ctx  context.Context
	typ  int
	id   string
	name string
}

func (c *Client) renameLocked(client *wopan.WoClient, params renameParams) error {
	return client.RenameFileOrDirectory(wopan.SpaceTypePersonal, params.typ, params.id, params.name, "", func(req *resty.Request) {
		req.SetContext(params.ctx)
	})
}

// Move 移动文件/目录到目标目录（个人云）
func (c *Client) Move(ctx context.Context, isDir bool, id, targetDirID string) error {
	dirList, fileList := operationIDs(isDir, id)
	return c.MoveBatch(ctx, dirList, fileList, targetDirID)
}

// MoveBatch 批量移动文件/目录到目标目录（个人云）
func (c *Client) MoveBatch(ctx context.Context, dirList, fileList []string, targetDirID string) error {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()

	client, err := c.getMetaClient()
	if err != nil {
		return err
	}
	c.syncClientTokens(client)

	params := moveParams{ctx: ctx, dirList: dirList, fileList: fileList, targetDir: targetDirID}
	err = c.moveLocked(client, params)
	if err != nil && isLoginFailed(err) {
		if refreshErr := c.refreshClientLocked(client); refreshErr != nil {
			return refreshErr
		}
		err = c.moveLocked(client, params)
	}
	return err
}

// Copy 复制文件/目录到目标目录（个人云）
func (c *Client) Copy(ctx context.Context, isDir bool, id, targetDirID string) error {
	dirList, fileList := operationIDs(isDir, id)
	return c.CopyBatch(ctx, dirList, fileList, targetDirID)
}

// CopyBatch 批量复制文件/目录到目标目录（个人云）
func (c *Client) CopyBatch(ctx context.Context, dirList, fileList []string, targetDirID string) error {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()

	client, err := c.getMetaClient()
	if err != nil {
		return err
	}
	c.syncClientTokens(client)
	params := copyParams{ctx: ctx, dirList: dirList, fileList: fileList, targetDir: targetDirID}
	err = c.copyLocked(client, params)
	if err != nil && isLoginFailed(err) {
		if refreshErr := c.refreshClientLocked(client); refreshErr != nil {
			return refreshErr
		}
		err = c.copyLocked(client, params)
	}
	return err
}

func operationIDs(isDir bool, id string) ([]string, []string) {
	if isDir {
		return []string{id}, []string{}
	}
	return []string{}, []string{id}
}

type copyParams struct {
	ctx       context.Context
	dirList   []string
	fileList  []string
	targetDir string
}

func (c *Client) copyLocked(client *wopan.WoClient, params copyParams) error {
	return client.CopyFile(params.dirList, params.fileList, params.targetDir,
		wopan.SpaceTypePersonal, wopan.SpaceTypePersonal, "", "", func(req *resty.Request) {
			req.SetContext(params.ctx)
		})
}

func (c *Client) refreshClientLocked(client *wopan.WoClient) error {
	accessToken, refreshToken := c.getTokens()
	if accessToken == "" && refreshToken == "" {
		return fmt.Errorf("刷新Token失败：未配置可用的 Token")
	}
	if err := client.RefreshToken(); err != nil {
		fallbackToken := accessToken
		if fallbackToken == "" {
			fallbackToken = refreshToken
		}
		if fallbackErr := c.retryRefreshValueAsAccess(client, fallbackToken, err); fallbackErr != nil {
			return fallbackErr
		}
	}
	return nil
}

type moveParams struct {
	ctx       context.Context
	dirList   []string
	fileList  []string
	targetDir string
}

func (c *Client) moveLocked(client *wopan.WoClient, params moveParams) error {
	return client.MoveFile(params.dirList, params.fileList, params.targetDir, wopan.SpaceTypePersonal, wopan.SpaceTypePersonal, "", "", func(req *resty.Request) {
		req.SetContext(params.ctx)
	})
}
