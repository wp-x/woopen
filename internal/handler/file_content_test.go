package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestFetchRemoteFileContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("联通云盘文本内容\n第二行"))
	}))
	defer server.Close()

	content, err := fetchRemoteFileContent(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatalf("读取远程文本失败: %v", err)
	}
	if string(content) != "联通云盘文本内容\n第二行" {
		t.Fatalf("文本内容不匹配: %q", content)
	}
}

func TestFetchRemoteFileContentRejectsUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}))
	defer server.Close()

	if _, err := fetchRemoteFileContent(context.Background(), server.Client(), server.URL); err == nil {
		t.Fatal("上游错误响应应返回错误")
	}
}

func TestFetchRemoteFileContentRejectsOversizedChunkedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Transfer-Encoding", "chunked")
		_, _ = w.Write(bytes.Repeat([]byte("x"), int(maxTextPreviewBytes)+1))
	}))
	defer server.Close()

	_, err := fetchRemoteFileContent(context.Background(), server.Client(), server.URL)
	if !errors.Is(err, errTextPreviewTooLarge) {
		t.Fatalf("超大文本应返回预览上限错误，实际为: %v", err)
	}
}

func TestWriteFileContentResponsePreservesArbitraryBytes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	original := []byte{0xff, 0xfe, 0x41, 0x00}

	writeFileContentResponse(ctx, original)
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("文本响应必须禁用缓存")
	}

	var response struct {
		Data struct {
			ContentBase64 string `json:"content_base64"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(response.Data.ContentBase64)
	if err != nil {
		t.Fatalf("Base64 解码失败: %v", err)
	}
	if string(decoded) != string(original) {
		t.Fatalf("字节内容不匹配: %v", decoded)
	}
}
