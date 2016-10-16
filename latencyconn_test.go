package connectors

import (
	"context"
	"log"
	"net"
	"testing"
	"time"

	"net/http"

	"crypto/md5"
	"encoding/hex"
	"io"

	humanize "github.com/dustin/go-humanize"
	"github.com/levigross/grequests"
)

type dialer struct {
	internalDialer net.Dialer
	lineRate       int64
	quantum        time.Duration
}

func (d *dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	myConn, err := d.internalDialer.DialContext(ctx, network, address)
	if err != nil {
		return myConn, err
	}
	return NewLatencyConn(d.lineRate, d.quantum, myConn), nil
}

func TestLatencyConn(t *testing.T) {
	lineRate, err := humanize.ParseBytes("1MB")
	if err != nil {
		t.Error("Unable to parse line rate", err)
	}

	myDialerContext := &dialer{
		internalDialer: net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
		lineRate: int64(lineRate),
		quantum:  time.Second,
	}

	var myTransport http.RoundTripper = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           myDialerContext.DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{Transport: myTransport}
	start := time.Now().UTC()
	resp, err := client.Get("http://speedtest-sfo1.digitalocean.com/10mb.test")
	if err != nil {
		t.Error("Unable to fetch DO page speed asset", err)
	}
	myMD5 := md5.New()
	if _, err := io.Copy(myMD5, resp.Body); err != nil {
		t.Error("Unable to copy speed test file into MD5 buffer", err)
	}
	end := time.Now().UTC()

	if hex.EncodeToString(myMD5.Sum(nil)) != "ef137d7e786535c992d1ccd5f7fff5be" {
		t.Errorf("Bad MD5 hash should be ef137d7e786535c992d1ccd5f7fff5be and is %v", hex.EncodeToString(myMD5.Sum(nil)))
	}

	if err := resp.Body.Close(); err != nil {
		t.Error("Unable to close response", err)
	}

	tooLong := time.Second * 10
	if end.Sub(start) < tooLong {
		t.Error("Download was too short", end.Sub(start))
	}

	derResp, err := grequests.Post("http://httpbin.org/post",
		&grequests.RequestOptions{Data: map[string]string{"One": "Two"}, HTTPClient: client})

	if err != nil {
		log.Println("Cannot post: ", err)
	}

	if derResp.Ok != true {
		log.Println("Request did not return OK")
	}

	if err := derResp.Close(); err != nil {
		t.Error("Unable to close response", err)
	}

}
