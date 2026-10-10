package wopan

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/OpenListTeam/wopan-sdk-go"
)

const fixtureAccessToken = "0123456789abcdef-fixture-only"

type listTestTransport func(*http.Request) (*http.Response, error)

func (transport listTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return transport(req)
}

func listTestClient(transport listTestTransport) *Client {
	upstream := sdk.DefaultWithAccessToken(fixtureAccessToken)
	upstream.SetHttpClient(&http.Client{Transport: transport})
	return &Client{metaClient: upstream, accessToken: fixtureAccessToken}
}

func decodeListRequest(t *testing.T, req *http.Request) map[string]interface{} {
	t.Helper()
	var envelope struct {
		Header struct{ Key string }
		Body   struct{ Param string }
	}
	if err := json.NewDecoder(req.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Header.Key != sdk.KeyQueryAllFiles {
		t.Fatalf("key = %q", envelope.Header.Key)
	}
	crypto := sdk.NewCrypto()
	if err := crypto.SetAccessToken(fixtureAccessToken); err != nil {
		t.Fatal(err)
	}
	plain, err := crypto.WoHomeDecrypt(envelope.Body.Param)
	if err != nil {
		t.Fatal(err)
	}
	var param map[string]interface{}
	if err := json.Unmarshal([]byte(plain), &param); err != nil {
		t.Fatal(err)
	}
	return param
}

func TestListFilesPreservesSyncNamesAndSeparateIDs(t *testing.T) {
	for _, name := range []string{"我的同步盘", "同步盘"} {
		t.Run(name, func(t *testing.T) {
			client := listTestClient(func(req *http.Request) (*http.Response, error) {
				param := decodeListRequest(t, req)
				if param["parentDirectoryId"] != "account-root" || param["spaceType"] != "0" {
					t.Fatalf("unexpected directory query: %+v", param)
				}
				if param["pageNum"] != float64(1) || param["pageSize"] != float64(100) {
					t.Fatalf("unexpected pagination: %+v", param)
				}
				file := sdk.File{Id: "operation-id", Fid: "content-fid", Name: name, Type: 0}
				data, err := json.Marshal(map[string]interface{}{
					"STATUS": "200", "RSP": map[string]interface{}{
						"RSP_CODE": "0000", "DATA": sdk.QueryAllFilesData{Files: []*sdk.File{&file}},
					},
				})
				if err != nil {
					t.Fatal(err)
				}
				return &http.Response{
					StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}},
					Body: io.NopCloser(strings.NewReader(string(data))), Request: req,
				}, nil
			})
			client.SetRootFolderID("account-root")
			files, err := client.ListFiles("", 2, 100)
			if err != nil || len(files) != 1 {
				t.Fatalf("files=%v error=%v", files, err)
			}
			file := files[0]
			if file.Name != name || file.ID != "operation-id" || file.FID != "content-fid" {
				t.Fatalf("upstream fields changed: %+v", file)
			}
			if file.ParentID != "account-root" {
				t.Fatalf("parent must be queried directory, got %q", file.ParentID)
			}
		})
	}
}

func TestListFilesCtxPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client := listTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Context().Done() != ctx.Done() {
			t.Error("upstream request does not carry caller context")
		}
		cancel()
		return nil, req.Context().Err()
	})
	_, err := client.ListFilesCtx(ctx, ListFilesOptions{DirID: "0", Page: 1, PageSize: 100})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
