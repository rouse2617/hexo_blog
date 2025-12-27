package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-ops/internal/ssh"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupHostHandler() (*HostHandler, *gin.Engine) {
	pool := ssh.NewPool(ssh.Config{})
	handler := NewHostHandler(pool)

	r := gin.New()
	api := r.Group("/api")
	{
		hosts := api.Group("/hosts")
		{
			hosts.GET("", handler.ListHosts)
			hosts.POST("", handler.CreateHost)
			hosts.GET("/:id", handler.GetHost)
			hosts.PUT("/:id", handler.UpdateHost)
			hosts.DELETE("/:id", handler.DeleteHost)
			hosts.POST("/:id/test", handler.TestConnection)
			hosts.POST("/import", handler.ImportHosts)
		}
		groups := api.Group("/groups")
		{
			groups.GET("", handler.GetGroups)
			groups.POST("", handler.CreateGroup)
			groups.DELETE("/:name", handler.DeleteGroup)
		}
	}

	return handler, r
}

// TestListHosts_Empty 测试空主机列表
func TestListHosts_Empty(t *testing.T) {
	_, r := setupHostHandler()

	req, _ := http.NewRequest("GET", "/api/hosts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	data := resp.Data.(map[string]interface{})
	assert.NotNil(t, data["list"])
	assert.Equal(t, float64(0), data["total"])
}

// TestCreateHost_Success 测试创建主机成功
func TestCreateHost_Success(t *testing.T) {
	_, r := setupHostHandler()

	body := map[string]interface{}{
		"name": "test-host",
		"host": "192.168.1.100",
		"port": 22,
		"user": "root",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/hosts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "test-host", data["id"])
	assert.Equal(t, "test-host", data["name"])
	assert.Equal(t, "192.168.1.100", data["host"])
	assert.Equal(t, float64(22), data["port"])
	assert.Equal(t, "root", data["user"])
}

// TestCreateHost_MissingName 测试创建主机缺少名称
func TestCreateHost_MissingName(t *testing.T) {
	_, r := setupHostHandler()

	body := map[string]interface{}{
		"host": "192.168.1.100",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/hosts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeParamError, resp.Code)
}

// TestCreateHost_MissingHost 测试创建主机缺少地址
func TestCreateHost_MissingHost(t *testing.T) {
	_, r := setupHostHandler()

	body := map[string]interface{}{
		"name": "test-host",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/hosts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeParamError, resp.Code)
}

// TestCreateHost_DefaultValues 测试创建主机默认值
func TestCreateHost_DefaultValues(t *testing.T) {
	_, r := setupHostHandler()

	body := map[string]interface{}{
		"name": "test-host-default",
		"host": "192.168.1.101",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/hosts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, float64(22), data["port"]) // 默认端口
	assert.Equal(t, "root", data["user"])      // 默认用户
	assert.Equal(t, "key", data["auth_type"])  // 默认认证类型
}

// TestListHosts_AfterCreate 测试创建后列表
func TestListHosts_AfterCreate(t *testing.T) {
	_, r := setupHostHandler()

	// 先创建一个主机
	body := map[string]interface{}{
		"name": "list-test-host",
		"host": "192.168.1.102",
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/hosts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 获取列表
	req, _ = http.NewRequest("GET", "/api/hosts", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, float64(1), data["total"])
	list := data["list"].([]interface{})
	assert.Len(t, list, 1)
}

// TestGetHost_Success 测试获取单个主机成功
func TestGetHost_Success(t *testing.T) {
	_, r := setupHostHandler()

	// 先创建一个主机
	body := map[string]interface{}{
		"name": "get-test-host",
		"host": "192.168.1.103",
		"port": 2222,
		"user": "admin",
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/hosts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 获取主机
	req, _ = http.NewRequest("GET", "/api/hosts/get-test-host", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "get-test-host", data["name"])
	assert.Equal(t, "192.168.1.103", data["host"])
	assert.Equal(t, float64(2222), data["port"])
	assert.Equal(t, "admin", data["user"])
}

// TestGetHost_NotFound 测试获取不存在的主机
func TestGetHost_NotFound(t *testing.T) {
	_, r := setupHostHandler()

	req, _ := http.NewRequest("GET", "/api/hosts/non-existent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeNotFound, resp.Code)
}

// TestUpdateHost_Success 测试更新主机成功
func TestUpdateHost_Success(t *testing.T) {
	_, r := setupHostHandler()

	// 先创建一个主机
	body := map[string]interface{}{
		"name": "update-test-host",
		"host": "192.168.1.104",
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/hosts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 更新主机
	updateBody := map[string]interface{}{
		"name": "update-test-host",
		"host": "192.168.1.200",
		"port": 3333,
		"user": "newuser",
	}
	jsonBody, _ = json.Marshal(updateBody)
	req, _ = http.NewRequest("PUT", "/api/hosts/update-test-host", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)
}

// TestDeleteHost_Success 测试删除主机成功
func TestDeleteHost_Success(t *testing.T) {
	_, r := setupHostHandler()

	// 先创建一个主机
	body := map[string]interface{}{
		"name": "delete-test-host",
		"host": "192.168.1.105",
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/hosts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 删除主机
	req, _ = http.NewRequest("DELETE", "/api/hosts/delete-test-host", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	// 验证已删除
	req, _ = http.NewRequest("GET", "/api/hosts/delete-test-host", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeNotFound, resp.Code)
}

// TestListHosts_WithKeyword 测试关键字过滤
func TestListHosts_WithKeyword(t *testing.T) {
	_, r := setupHostHandler()

	// 创建多个主机
	hosts := []map[string]interface{}{
		{"name": "web-server-1", "host": "192.168.1.10"},
		{"name": "web-server-2", "host": "192.168.1.11"},
		{"name": "db-server-1", "host": "192.168.1.20"},
	}

	for _, h := range hosts {
		jsonBody, _ := json.Marshal(h)
		req, _ := http.NewRequest("POST", "/api/hosts", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// 搜索 web
	req, _ := http.NewRequest("GET", "/api/hosts?keyword=web", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, float64(2), data["total"])
}

// TestGetGroups 测试获取分组列表
func TestGetGroups(t *testing.T) {
	_, r := setupHostHandler()

	req, _ := http.NewRequest("GET", "/api/groups", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)
}

// TestCreateGroup_Success 测试创建分组成功
func TestCreateGroup_Success(t *testing.T) {
	_, r := setupHostHandler()

	body := map[string]interface{}{
		"name":        "production",
		"description": "生产环境服务器",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/groups", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)
}

// TestCreateGroup_MissingName 测试创建分组缺少名称
func TestCreateGroup_MissingName(t *testing.T) {
	_, r := setupHostHandler()

	body := map[string]interface{}{
		"description": "测试分组",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/groups", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeParamError, resp.Code)
}

// TestDeleteGroup_Success 测试删除分组成功
func TestDeleteGroup_Success(t *testing.T) {
	_, r := setupHostHandler()

	req, _ := http.NewRequest("DELETE", "/api/groups/test-group", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)
}

// TestImportHosts 测试批量导入
func TestImportHosts(t *testing.T) {
	_, r := setupHostHandler()

	body := map[string]interface{}{
		"format": "json",
		"data":   "[]",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/hosts/import", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)
}

// TestTestConnection_HostNotFound 测试连接不存在的主机
func TestTestConnection_HostNotFound(t *testing.T) {
	_, r := setupHostHandler()

	req, _ := http.NewRequest("POST", "/api/hosts/non-existent/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSSHError, resp.Code)
}
