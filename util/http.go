package util

import (
	"net"
	"net/http"
	"time"
)

// SingleConnNoKeepAliveTransporter returns a transporter with no keep alives and a max of 1 idle connection
func SingleConnNoKeepAliveTransporter() *http.Transport {
	return &http.Transport{
		Dial:                (&net.Dialer{Timeout: 5 * time.Second}).Dial,
		DisableKeepAlives:   true,
		MaxIdleConns:        1,
		IdleConnTimeout:     5 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	}
}

// HTTPClient generates an http client
func HttpClient() (*http.Client, error) {
	client := &http.Client{
		Transport: SingleConnNoKeepAliveTransporter(),
		Timeout:   5 * time.Second,
	}
	return client, nil
}
