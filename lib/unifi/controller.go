package unifi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"strings"
)

var (
	loginError = fmt.Errorf("not logged in")
	siteError  = fmt.Errorf("invalid/missing site")
)

func readWithSize(r io.Reader, maxSize int) ([]byte, error) {
	buf := make([]byte, maxSize+1)
	n, err := r.Read(buf)

	if n > maxSize {
		return nil, fmt.Errorf("response too large (>%d bytes)", maxSize)
	}

	if n > 0 {
		return buf[0:n], nil
	}

	if err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("empty read")
}

func unmarshalBody(r io.Reader, d interface{}) error {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(buf, d); err != nil {
		return err
	}
	return nil
}

func decodeControllerError(resp *http.Response) error {
	errorFromStatus := func() error {
		if resp.Status != "" {
			return fmt.Errorf("%s", resp.Status)
		}
		return fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	body, err := readWithSize(resp.Body, 1000)
	if err != nil {
		return errorFromStatus()
	}

	type ErrorResp struct {
		Errors []string
	}

	errorResp := &ErrorResp{}
	if err := json.Unmarshal(body, errorResp); err != nil {
		return errorFromStatus()
	}

	if len(errorResp.Errors) > 0 {
		return fmt.Errorf("error from controller: %v",
			errorResp.Errors[0])
	}

	return errorFromStatus()
}

type Controller struct {
	username string
	site     string
	baseURL  url.URL

	loggedIn bool
	client   *http.Client
}

func NewController(username, site string, baseURL *url.URL) *Controller {
	jar, _ := cookiejar.New(nil)

	return &Controller{
		username: username,
		site:     site,
		baseURL:  *baseURL,
		loggedIn: false,
		client: &http.Client{
			Jar: jar,
		},
	}
}

func (c *Controller) siteNetworkAPI(api string) string {
	return path.Join("/proxy/network/api/s", c.site, api)
}

func (c *Controller) requireLoginAndSite() error {
	if !c.loggedIn {
		return loginError
	}
	if c.site == "" {
		return siteError
	}

	return nil
}

func (c *Controller) getCookie(name string) *http.Cookie {
	for _, cookie := range c.client.Jar.Cookies(&c.baseURL) {
		if cookie.Name == name {
			return cookie
		}
	}

	return nil
}

func (c *Controller) deleteCookie(name string) {
	cookie := c.getCookie(name)
	if cookie == nil {
		return
	}

	cookie.Value = ""
	cookie.MaxAge = -1
	c.client.Jar.SetCookies(&c.baseURL, []*http.Cookie{cookie})
}

func (c *Controller) getCSRFToken() (string, bool) {
	cookie := c.getCookie("TOKEN")
	if cookie == nil {
		return "", false
	}

	parts := strings.SplitN(cookie.Value, ".", 3)
	if len(parts) != 3 {
		return "", false
	}

	enc := parts[1]

	// The controller omits padding characters
	dec, _ := base64.RawStdEncoding.DecodeString(enc)
	if len(dec) == 0 {
		return "", false
	}

	type Parsed struct {
		CSRFToken string
	}
	parsed := &Parsed{}
	if err := json.Unmarshal(dec, parsed); err != nil {
		return "", false
	}

	return parsed.CSRFToken, parsed.CSRFToken != ""
}

func (c *Controller) sendRequest(ctx context.Context, reqType, api string, data interface{}) (respBody io.ReadCloser, err error) {
	var reqBody io.Reader
	if data != nil {
		jd, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}

		reqBody = strings.NewReader(string(jd))
	}

	ref, _ := url.Parse(api)
	url := c.baseURL.ResolveReference(ref)

	req, err := http.NewRequestWithContext(ctx, reqType, url.String(), reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	if tok, found := c.getCSRFToken(); found {
		req.Header.Add("x-csrf-token", tok)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		err := decodeControllerError(resp)
		resp.Body.Close()
		return nil, err
	}

	return resp.Body, nil
}

func (c *Controller) Login(ctx context.Context, password string) error {
	type LoginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	loginData := &LoginData{
		Username: c.username,
		Password: password,
	}

	body, err := c.sendRequest(ctx, "POST", "/api/auth/login", loginData)
	if err != nil {
		return err
	}

	body.Close()
	c.loggedIn = true
	return nil
}

type Site struct {
	Desc string
	Name string
}

func (c *Controller) Sites(ctx context.Context) ([]*Site, error) {
	if !c.loggedIn {
		return nil, loginError
	}

	body, err := c.sendRequest(ctx, "GET", "/proxy/network/api/stat/sites", nil)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	type SitesResp struct {
		Data []*Site
	}
	resp := &SitesResp{}
	if err := unmarshalBody(body, resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

type DeviceAddr struct {
	Name string
	IP   net.IP
	MAC  net.HardwareAddr
}

func parseDeviceAddr(name, ip, mac string, da *DeviceAddr) (err error) {
	da.Name = name

	da.IP = net.ParseIP(ip)
	if da.IP == nil {
		return fmt.Errorf("bad IP addr %v", ip)
	}

	da.MAC, err = net.ParseMAC(mac)
	if err != nil {
		return fmt.Errorf("bad MAC addr %v: %w", mac, err)
	}

	return nil
}

type Device struct {
	DeviceAddr
	ExtraPorts []DeviceAddr
}

func (c *Controller) Devices(ctx context.Context) ([]*Device, error) {
	if err := c.requireLoginAndSite(); err != nil {
		return nil, err
	}

	body, err := c.sendRequest(ctx, "GET", c.siteNetworkAPI("/stat/device"), nil)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	type Resp struct {
		Data []struct {
			Name         string
			IP           string
			MAC          string
			NetworkTable []struct {
				Name string
				IP   string
				MAC  string
			} `json:"network_table"`
		}
	}

	resp := &Resp{}
	if err := unmarshalBody(body, resp); err != nil {
		return nil, err
	}

	out := []*Device{}
	for _, raw := range resp.Data {
		dev := &Device{
			ExtraPorts: make([]DeviceAddr, len(raw.NetworkTable)),
		}

		if err := parseDeviceAddr(raw.Name, raw.IP, raw.MAC, &dev.DeviceAddr); err != nil {
			return nil, fmt.Errorf("bad top-level device addr for %v: %w",
				raw.Name, err)
		}

		for i, ent := range raw.NetworkTable {
			port := &dev.ExtraPorts[i]
			if err := parseDeviceAddr(ent.Name, ent.IP, ent.MAC, port); err != nil {
				return nil, fmt.Errorf("bad entry %v addr for %v: %w",
					ent.Name, raw.Name, err)
			}

		}

		out = append(out, dev)
	}

	return out, nil
}

func (c *Controller) ForceProvision(ctx context.Context, mac net.HardwareAddr) error {
	if err := c.requireLoginAndSite(); err != nil {
		return err
	}

	type Cmd struct {
		Mac string `json:"mac"`
		Cmd string `json:"cmd"`
	}

	cmd := &Cmd{
		Mac: mac.String(),
		Cmd: "force-provision",
	}

	body, err := c.sendRequest(ctx, "POST", c.siteNetworkAPI("/cmd/devmgr"), cmd)
	if err != nil {
		return err
	}
	defer body.Close()

	return nil
}
