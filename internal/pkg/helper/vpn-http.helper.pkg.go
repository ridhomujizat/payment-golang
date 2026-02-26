package helper

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"go-boilerplate/internal/pkg/logger"
	"time"
)

// VPNConfig represents VPN configuration
type VPNConfig struct {
	UseVPN         bool
	VPNInterface   string
	ProxyURL       string
	SkipTLSVerify  bool
	RequestTimeout int
}

// VPNHTTPClient creates an HTTP client configured for VPN usage
type VPNHTTPClient struct {
	Client *http.Client
	Config *VPNConfig
}

// NewVPNHTTPClient creates a new VPN-aware HTTP client
func NewVPNHTTPClient(cfg *VPNConfig) *VPNHTTPClient {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.SkipTLSVerify,
		},
	}

	// Configure proxy if provided
	if cfg.ProxyURL != "" {
		proxyURL, err := url.Parse(cfg.ProxyURL)
		if err != nil {
			logger.Error.Printf("Invalid proxy URL: %v", err)
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
			logger.Debug.Printf("Using proxy: %s", cfg.ProxyURL)
		}
	}

	// Configure VPN interface binding if specified
	if cfg.UseVPN && cfg.VPNInterface != "" {
		logger.Debug.Printf("VPN enabled with interface: %s", cfg.VPNInterface)
		// Note: Interface binding would require platform-specific implementation
		// This is a placeholder for VPN interface configuration
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(cfg.RequestTimeout) * time.Second,
	}

	return &VPNHTTPClient{
		Client: client,
		Config: cfg,
	}
}

// VPNHTTPRequest performs an HTTP request using VPN-configured client
func (v *VPNHTTPClient) VPNHTTPRequest(
	payload *HTTPRequestPayload,
	config *HTTPRequestConfig,
) (*HTTPAPIResponse, error) {
	requestBody, err := handleRequestBody(payload, config)
	if err != nil {
		logger.Debug.Println("Error handling request body:", err.Error())
		return nil, err
	}

	req, err := v.prepareVPNRequest(payload, requestBody, config)
	if err != nil {
		logger.Debug.Println("Error preparing VPN request:", err.Error())
		return nil, err
	}

	return v.executeVPNRequest(req)
}

// prepareVPNRequest prepares the HTTP request with VPN client
func (v *VPNHTTPClient) prepareVPNRequest(payload *HTTPRequestPayload, body io.Reader, config *HTTPRequestConfig) (*http.Request, error) {
	req, err := http.NewRequestWithContext(config.Ctx, payload.Method.ToString(), payload.URL, body)
	if err != nil {
		return nil, err
	}

	// Add custom headers (Content-Type is already set in the headers)
	for key, values := range config.Headers {
		req.Header[key] = append(req.Header[key], values...)
	}

	// Add basic auth if provided
	if config.Auth != nil {
		req.SetBasicAuth(config.Auth.Username, config.Auth.Password)
	}

	// Add query parameters
	if len(payload.Params) > 0 {
		q := req.URL.Query()
		for key, value := range payload.Params {
			q.Add(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}

	return req, nil
}

// executeVPNRequest executes the HTTP request using VPN client
func (v *VPNHTTPClient) executeVPNRequest(req *http.Request) (*HTTPAPIResponse, error) {
	logger.Debug.Printf("Making VPN request to: %s", req.URL.String())

	resp, err := v.Client.Do(req)
	if err != nil {
		logger.Error.Printf("VPN request failed: %v", err)
		return nil, fmt.Errorf("VPN request failed: %w", err)
	}
	defer resp.Body.Close()

	result, err := parseResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	logger.Debug.Printf("VPN request completed with status: %d", resp.StatusCode)

	return &HTTPAPIResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Data:       result,
	}, nil
}
