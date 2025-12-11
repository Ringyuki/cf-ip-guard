package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	HTTPClient *http.Client
	APIURL     string
}

type ipResponse struct {
	Success bool `json:"success"`
	Result  struct {
		IPv4Cidrs []string `json:"ipv4_cidrs"`
		IPv6Cidrs []string `json:"ipv6_cidrs"`
		Etag      string   `json:"etag"`
	} `json:"result"`
}

func (c *Client) FetchIPs(ctx context.Context, prevETag string) (ipv4, ipv6 []string, etag string, notModified bool, err error) {
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{}
	}
	if c.APIURL == "" {
		c.APIURL = "https://api.cloudflare.com/client/v4/ips"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.APIURL, nil)
	if err != nil {
		return nil, nil, "", false, fmt.Errorf("build request: %w", err)
	}
	if prevETag != "" {
		req.Header.Set("If-None-Match", prevETag)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, "", false, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil, nil, prevETag, true, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, "", false, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var ipResp ipResponse
	if err := json.NewDecoder(resp.Body).Decode(&ipResp); err != nil {
		return nil, nil, "", false, fmt.Errorf("decode json: %w", err)
	}

	if !ipResp.Success {
		return nil, nil, "", false, fmt.Errorf("cloudflare api returned success=false")
	}

	return ipResp.Result.IPv4Cidrs, ipResp.Result.IPv6Cidrs, ipResp.Result.Etag, false, nil
}
