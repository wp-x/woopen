package wopan

import (
	"context"
	"fmt"

	"github.com/OpenListTeam/wopan-sdk-go"
	"resty.dev/v3"
)

// Delete 删除文件/目录（个人云）
func (c *Client) Delete(ctx context.Context, isDir bool, id string) error {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()

	client, err := c.getMetaClient()
	if err != nil {
		return err
	}
	c.syncClientTokens(client)

	dirList := make([]string, 0, 1)
	fileList := make([]string, 0, 1)
	if isDir {
		dirList = append(dirList, id)
	} else {
		fileList = append(fileList, id)
	}

	err = c.deleteLocked(client, deleteParams{
		ctx:      ctx,
		dirList:  dirList,
		fileList: fileList,
	})
	if err != nil && isLoginFailed(err) {
		if refreshErr := client.RefreshToken(); refreshErr != nil {
			return fmt.Errorf("刷新Token失败: %w", refreshErr)
		}
		err = c.deleteLocked(client, deleteParams{
			ctx:      ctx,
			dirList:  dirList,
			fileList: fileList,
		})
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
		if refreshErr := client.RefreshToken(); refreshErr != nil {
			return fmt.Errorf("刷新Token失败: %w", refreshErr)
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
	c.metaMu.Lock()
	defer c.metaMu.Unlock()

	client, err := c.getMetaClient()
	if err != nil {
		return err
	}
	c.syncClientTokens(client)

	dirList := make([]string, 0, 1)
	fileList := make([]string, 0, 1)
	if isDir {
		dirList = append(dirList, id)
	} else {
		fileList = append(fileList, id)
	}

	err = c.moveLocked(client, moveParams{
		ctx:       ctx,
		dirList:   dirList,
		fileList:  fileList,
		targetDir: targetDirID,
	})
	if err != nil && isLoginFailed(err) {
		if refreshErr := client.RefreshToken(); refreshErr != nil {
			return fmt.Errorf("刷新Token失败: %w", refreshErr)
		}
		err = c.moveLocked(client, moveParams{
			ctx:       ctx,
			dirList:   dirList,
			fileList:  fileList,
			targetDir: targetDirID,
		})
	}
	return err
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
