package proxy

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

// TransformFunc is a function that transforms data
type TransformFunc func([]byte) ([]byte, error)

// Transformer transforms HTTP requests and responses
type Transformer struct {
	requestHeaders  map[string]string
	responseHeaders map[string]string
	requestTransform  TransformFunc
	responseTransform TransformFunc
}

// NewTransformer creates a new transformer
func NewTransformer() *Transformer {
	return &Transformer{
		requestHeaders:  make(map[string]string),
		responseHeaders: make(map[string]string),
	}
}

// SetRequestHeader sets a request header
func (t *Transformer) SetRequestHeader(key, value string) {
	t.requestHeaders[key] = value
}

// SetResponseHeader sets a response header
func (t *Transformer) SetResponseHeader(key, value string) {
	t.responseHeaders[key] = value
}

// SetRequestTransform sets the request body transform function
func (t *Transformer) SetRequestTransform(f TransformFunc) {
	t.requestTransform = f
}

// SetResponseTransform sets the response body transform function
func (t *Transformer) SetResponseTransform(f TransformFunc) {
	t.responseTransform = f
}

// TransformRequest transforms an HTTP request
func (t *Transformer) TransformRequest(req *http.Request) {
	// Add/modify headers
	for k, v := range t.requestHeaders {
		req.Header.Set(k, v)
	}
	
	// Transform body if needed
	if t.requestTransform != nil && req.Body != nil {
		body, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		
		if err == nil {
			transformed, err := t.requestTransform(body)
			if err == nil {
				req.Body = ioutil.NopCloser(bytes.NewBuffer(transformed))
				req.ContentLength = int64(len(transformed))
				req.Header.Set("Content-Length", string(rune(len(transformed))))
			}
		}
	}
}

// TransformResponse transforms an HTTP response
func (t *Transformer) TransformResponse(resp *http.Response) error {
	// Add/modify headers
	for k, v := range t.responseHeaders {
		resp.Header.Set(k, v)
	}
	
	// Transform body if needed
	if t.responseTransform != nil && resp.Body != nil {
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		
		if err != nil {
			return err
		}
		
		transformed, err := t.responseTransform(body)
		if err != nil {
			return err
		}
		
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(transformed))
		resp.ContentLength = int64(len(transformed))
		resp.Header.Set("Content-Length", string(rune(len(transformed))))
	}
	
	return nil
}
