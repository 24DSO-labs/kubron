package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
)

func (m *Manager) QueryPrometheus(query string) (*PromQueryResult, error) {
	return m.promQuery("/api/v1/query", map[string]string{
		"query": query,
	})
}

func (m *Manager) QueryPrometheusRange(query string, start, end time.Time, step string) (*PromQueryResult, error) {
	return m.promQuery("/api/v1/query_range", map[string]string{
		"query": query,
		"start": fmt.Sprintf("%d", start.Unix()),
		"end":   fmt.Sprintf("%d", end.Unix()),
		"step":  step,
	})
}

func (m *Manager) promQuery(path string, params map[string]string) (*PromQueryResult, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	// Build URL to prometheus service via k8s API proxy
	proxyURL := fmt.Sprintf(
		"%s/api/v1/namespaces/%s/services/prometheus:web/proxy%s",
		conn.Config.Host,
		MetricsNamespace,
		path,
	)

	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	// Create HTTP client with k8s auth
	httpClient, err := buildK8sHTTPClient(conn.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus returned %d: %s", resp.StatusCode, string(body))
	}

	var result PromQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func buildK8sHTTPClient(config *rest.Config) (*http.Client, error) {
	transportConfig, err := config.TransportConfig()
	if err != nil {
		return nil, err
	}

	rt := http.DefaultTransport.(*http.Transport).Clone()

	// Apply TLS config
	tlsConfig, err := transport.TLSConfigFor(transportConfig)
	if err != nil {
		return nil, err
	}
	if tlsConfig != nil {
		rt.TLSClientConfig = tlsConfig
	}

	var roundTripper http.RoundTripper = rt

	// Always wrap with auth - this ensures bearer tokens and certs are added
	roundTripper, err = transport.HTTPWrappersForConfig(transportConfig, roundTripper)
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Transport: roundTripper,
		Timeout:   30 * time.Second,
	}, nil
}
