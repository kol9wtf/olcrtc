package lksdk

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/owenewans/owenlivekit/v2/signalling"
	"github.com/pion/webrtc/v4"
)

func TestNetworkConnectOptions(t *testing.T) {
	httpClient := &http.Client{}
	wsDialer := &websocket.Dialer{}
	settingEngineFunc := func(*webrtc.SettingEngine) {}
	params := &connParams{ConnectParams: &signalling.ConnectParams{}}

	WithConnectHTTPClient(httpClient)(params)
	WithWebSocketDialer(wsDialer)(params)
	WithSettingEngineFunc(settingEngineFunc)(params)

	if params.HTTPClient != httpClient {
		t.Fatal("WithConnectHTTPClient did not retain the client")
	}
	if params.WebSocketDialer != wsDialer {
		t.Fatal("WithWebSocketDialer did not retain the dialer")
	}
	if params.SettingEngineFunc == nil {
		t.Fatal("WithSettingEngineFunc did not retain the callback")
	}
}

func TestRegionURLProviderHTTPClient(t *testing.T) {
	provider := newRegionURLProvider()
	client := &http.Client{}
	provider.SetHTTPClient(client)
	if provider.cache.client != client {
		t.Fatal("SetHTTPClient did not retain the client")
	}
}

func TestPrepareConnectionUsesHTTPClient(t *testing.T) {
	originalURL := regionSettingsURL
	regionSettingsURL = func(string) string { return "https://regions.test/settings/regions" }
	t.Cleanup(func() { regionSettingsURL = originalURL })
	called := false
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		called = true
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"regions":[]}`)),
		}, nil
	})}
	room := NewRoom(nil)
	if err := room.PrepareConnection(
		"https://project.livekit.cloud", "token", WithConnectHTTPClient(client),
	); err != nil {
		t.Fatalf("PrepareConnection() error = %v", err)
	}
	if !called {
		t.Fatal("PrepareConnection did not use the injected HTTP client")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
