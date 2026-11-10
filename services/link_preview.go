package services

import (
	"context"
	"errors"
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"gorm.io/gorm/clause"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func publicURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || len(raw) > 2000 {
		return nil, errors.New("invalid public URL")
	}
	return u, nil
}
func publicIP(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || inCGNAT(ip) {
		return false
	}
	// Documentation, benchmarking and translation ranges are not public source
	// destinations. In particular NAT64 must not provide a route to metadata.
	for _, cidr := range []string{"0.0.0.0/8", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001:db8::/32", "64:ff9b::/96", "64:ff9b:1::/48", "100::/64"} {
		_, block, _ := net.ParseCIDR(cidr)
		if block.Contains(ip) {
			return false
		}
	}
	return true
}

func inCGNAT(ip net.IP) bool {
	_, block, _ := net.ParseCIDR("100.64.0.0/10")
	return block.Contains(ip)
}

func GetPublicLinkPreview(ctx context.Context, raw string) (*models.LinkPreview, error) {
	u, err := publicURL(raw)
	if err != nil {
		return nil, err
	}
	u.Fragment = ""
	key := u.String()
	var cached models.LinkPreview
	if config.DB.Where("url = ? AND updated_at > ?", key, time.Now().Add(-24*time.Hour)).First(&cached).Error == nil {
		return &cached, nil
	}
	dialer := net.Dialer{Timeout: 4 * time.Second}
	transport := &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if !publicIP(ip.IP) {
				return nil, errors.New("private address refused")
			}
		}
		if len(ips) == 0 {
			return nil, errors.New("no address")
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}}
	defer transport.CloseIdleConnections()
	client := http.Client{Timeout: 7 * time.Second, Transport: transport, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) >= 4 {
			return errors.New("too many redirects")
		}
		_, err := publicURL(r.URL.String())
		return err
	}}
	req, err := http.NewRequestWithContext(ctx, "GET", key, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "eCOnectiva-LinkPreview/1.0")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(res.Header.Get("Content-Type"), "text/html") {
		return nil, errors.New("metadata unavailable")
	}
	doc, err := parsePreviewDocument(res.Body, res.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	preview := models.LinkPreview{URL: key, UpdatedAt: time.Now()}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" && preview.Title == "" && n.FirstChild != nil {
			preview.Title = n.FirstChild.Data
		}
		if n.Type == html.ElementNode && n.Data == "meta" {
			var property, content string
			for _, a := range n.Attr {
				if a.Key == "property" || a.Key == "name" {
					property = strings.ToLower(a.Val)
				}
				if a.Key == "content" {
					content = a.Val
				}
			}
			switch property {
			case "og:title":
				preview.Title = content
			case "description", "og:description":
				preview.Description = content
			case "og:image":
				if image, err := res.Request.URL.Parse(content); err == nil {
					if _, err := publicURL(image.String()); err == nil {
						preview.Image = image.String()
					}
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(doc)
	if len(key) <= 768 {
		_ = config.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&preview).Error
	}
	return &preview, nil
}

// External pages can still use ISO-8859-1/Windows-1252. Decode their declared
// encoding before extracting text so imported references keep their accents.
func parsePreviewDocument(body io.Reader, contentType string) (*html.Node, error) {
	reader, err := charset.NewReader(io.LimitReader(body, 1024*1024), contentType)
	if err != nil {
		return nil, err
	}
	return html.Parse(reader)
}
