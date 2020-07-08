package cli

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/progrhyme/binq-gh/binqgh"
	"github.com/progrhyme/binq-gh/internal/erron"
	"github.com/progrhyme/go-lv"
)

func newHTTPClient(timeout time.Duration) (hc *http.Client) {
	hc = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext,
			IdleConnTimeout:       10 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: timeout,
	}
	return hc
}

func newHTTPGetRequest(url string, headers map[string]string) (req *http.Request, err error) {
	req, _err := http.NewRequest(http.MethodGet, url, nil)
	if _err != nil {
		return req, erron.Errorwf(_err, "Failed to create HTTP request")
	}
	req.Header.Set("User-Agent", fmt.Sprintf("binq-gh/%s", binqgh.Version))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req, nil
}

func doHTTPGetRequest(
	uri string, headers map[string]string, timeout time.Duration) (res *http.Response, err error) {

	req, err := newHTTPGetRequest(uri, headers)
	if err != nil {
		lv.Errorf("Failed to create HTTP request. %v\n", err)
		return res, err
	}

	hc := newHTTPClient(timeout)
	res, err = hc.Do(req)
	if err != nil {
		lv.Errorf("Failed to execute HTTP request. %v\n", err)
		return res, err
	}

	return res, err
}
