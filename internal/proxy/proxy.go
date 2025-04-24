package proxy

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Proxy represents an HTTP proxy
type Proxy struct {
	targetURL *url.URL
	proxy     *httputil.ReverseProxy
	transform *Transformer
}

// NewProxy creates a new HTTP proxy
func NewProxy(targetURL string, transform *Transformer) (*Proxy, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}
	
	proxy := httputil.NewSingleHostReverseProxy(parsedURL)
	
	// Customize the Director function to modify the request
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		
		// Allow transformer to modify request if needed
		if transform != nil {
			transform.TransformRequest(req)
		}
	}
	
	// Add a response modifier
	proxy.ModifyResponse = func(resp *http.Response) error {
		if transform != nil {
			return transform.TransformResponse(resp)
		}
		return nil
	}
	
	return &Proxy{
		targetURL: parsedURL,
		proxy:     proxy,
		transform: transform,
	}, nil
}

// ServeHTTP implements the http.Handler interface
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.proxy.ServeHTTP(w, r)
}

// HandleRequest handles an HTTP request without using the built-in proxy
func (p *Proxy) HandleRequest(r *http.Request) (*http.Response, error) {
	// Create a new client
	client := &http.Client{}
	
	// Create a new request
	targetURL := *p.targetURL
	targetURL.Path = r.URL.Path
	targetURL.RawQuery = r.URL.RawQuery
	
	req, err := http.NewRequest(r.Method, targetURL.String(), nil)
	if err != nil {
		return nil, err
	}
	
	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	
	// Copy body if needed
	if r.Body != nil {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	}
	
	// Apply transformer if needed
	if p.transform != nil {
		p.transform.TransformRequest(req)
	}
	
	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	
	// Apply transformer to response if needed
	if p.transform != nil {
		err = p.transform.TransformResponse(resp)
		if err != nil {
			resp.Body.Close()
			return nil, err
		}
	}
	
	return resp, nil
}
