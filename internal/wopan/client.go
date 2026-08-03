package wopan

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/OpenListTeam/wopan-sdk-go"
)

// Client 联通云盘客户端封装
type Client struct {
	metaClient      *wopan.WoClient
	uploadClient    *wopan.WoClient
	refreshToken    string
	accessToken     string
	rootFolderID    string
	defaultFamilyID string
	mu              sync.RWMutex
	metaMu          sync.Mutex
	uploadMu        sync.Mutex

	// Token更新回调
	onTokenUpdate func(refreshToken, accessToken string)
}

// NewClient 创建联通云盘客户端
func NewClient(refreshToken string) *Client {
	return &Client{
		refreshToken: refreshToken,
	}
}

// SetTokenUpdateCallback 设置Token更新回调
func (c *Client) SetTokenUpdateCallback(callback func(refreshToken, accessToken string)) {
	c.mu.Lock()
	c.onTokenUpdate = callback
	c.mu.Unlock()
}

// SetRootFolderID 设置根文件夹ID
func (c *Client) SetRootFolderID(rootFolderID string) {
	c.mu.Lock()
	c.rootFolderID = rootFolderID
	c.mu.Unlock()
}

// SetAccessToken 设置AccessToken
func (c *Client) SetAccessToken(accessToken string) {
	c.mu.Lock()
	c.accessToken = accessToken
	c.mu.Unlock()
}

// Init 初始化个人云客户端，并加载创建目录所需的默认家庭空间 ID。
func (c *Client) Init() error {
	accessToken, refreshToken := c.getTokens()
	if refreshToken == "" && accessToken == "" {
		return fmt.Errorf("refresh_token 和 access_token 不能同时为空")
	}

	metaClient := c.buildSDKClient(accessToken, refreshToken)
	uploadClient := c.buildSDKClient(accessToken, refreshToken)

	if err := c.initSDKClient(metaClient, &c.metaMu); err != nil {
		return err
	}
	if err := c.initDefaultFamilyID(metaClient); err != nil {
		return err
	}
	if err := c.initSDKClient(uploadClient, &c.uploadMu); err != nil {
		return err
	}
	c.mu.Lock()
	c.metaClient = metaClient
	c.uploadClient = uploadClient
	c.mu.Unlock()
	return nil
}

func (c *Client) initDefaultFamilyID(client *wopan.WoClient) error {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()

	c.syncClientTokens(client)
	data, err := client.FamilyUserCurrentEncode()
	if err != nil {
		return fmt.Errorf("获取默认家庭空间ID失败: %w", err)
	}
	c.mu.Lock()
	c.defaultFamilyID = strconv.Itoa(data.DefaultHomeId)
	c.mu.Unlock()
	return nil
}

func (c *Client) buildSDKClient(accessToken, refreshToken string) *wopan.WoClient {
	var client *wopan.WoClient
	if refreshToken != "" {
		client = wopan.DefaultWithRefreshToken(refreshToken)
	} else {
		client = wopan.DefaultWithAccessToken(accessToken)
	}
	if accessToken != "" {
		client.SetAccessToken(accessToken)
	}
	client.OnRefreshToken(c.onSDKRefresh)
	return client
}

func (c *Client) initSDKClient(client *wopan.WoClient, mu *sync.Mutex) error {
	mu.Lock()
	defer mu.Unlock()

	c.syncClientTokens(client)
	if err := client.InitData(); err != nil {
		if isLoginFailed(err) {
			_, refreshToken := c.getTokens()
			if refreshToken == "" {
				return fmt.Errorf("初始化客户端数据失败: %w", err)
			}
			if refreshErr := client.RefreshToken(); refreshErr != nil {
				return c.retryRefreshValueAsAccess(client, refreshToken, refreshErr)
			}
			if retryErr := client.InitData(); retryErr != nil {
				return fmt.Errorf("初始化客户端数据失败: %w", retryErr)
			}
			return nil
		}
		return fmt.Errorf("初始化客户端数据失败: %w", err)
	}
	return nil
}

func (c *Client) retryRefreshValueAsAccess(client *wopan.WoClient, token string, refreshErr error) error {
	// 最新联通云盘页面有时只提供一个 UUID Access Token。迁移后的旧配置会
	// 把它放入 refresh_token；刷新返回 8005 后，再按 Access Token 明确验证。
	client.SetAccessToken(token)
	client.SetRefreshToken("")
	if err := client.InitData(); err != nil {
		return fmt.Errorf("初始化客户端数据失败（Refresh Token 和 Access Token 均不可用）: refresh=%w; access=%v", refreshErr, err)
	}

	c.mu.Lock()
	c.accessToken = token
	c.refreshToken = ""
	callback := c.onTokenUpdate
	c.mu.Unlock()
	if callback != nil {
		callback("", token)
	}
	return nil
}

func (c *Client) onSDKRefresh(accessToken, refreshToken string) {
	c.mu.Lock()
	c.accessToken = accessToken
	c.refreshToken = refreshToken
	callback := c.onTokenUpdate
	c.mu.Unlock()

	if callback != nil {
		callback(refreshToken, accessToken)
	}
}

func (c *Client) getTokens() (string, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken, c.refreshToken
}

func (c *Client) getRootFolderID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.rootFolderID
}

func (c *Client) getDefaultFamilyID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.defaultFamilyID
}

func (c *Client) getMetaClient() (*wopan.WoClient, error) {
	c.mu.RLock()
	client := c.metaClient
	c.mu.RUnlock()
	if client == nil {
		return nil, fmt.Errorf("客户端未初始化")
	}
	return client, nil
}

func (c *Client) getUploadClient() (*wopan.WoClient, error) {
	c.mu.RLock()
	client := c.uploadClient
	c.mu.RUnlock()
	if client == nil {
		return nil, fmt.Errorf("客户端未初始化")
	}
	return client, nil
}

func (c *Client) syncClientTokens(client *wopan.WoClient) {
	accessToken, refreshToken := c.getTokens()
	curAccess, curRefresh := client.GetToken()
	if accessToken != curAccess {
		client.SetAccessToken(accessToken)
	}
	if refreshToken != curRefresh {
		client.SetRefreshToken(refreshToken)
	}
}

// IsInitialized 检查客户端是否已初始化
func (c *Client) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metaClient != nil
}

// GetRefreshToken 获取当前的RefreshToken
func (c *Client) GetRefreshToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.refreshToken
}

// GetAccessToken 获取当前的AccessToken
func (c *Client) GetAccessToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken
}
