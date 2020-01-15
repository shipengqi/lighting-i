package client

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"strings"
)

var (
	BasicAuthType  = "Basic"
	BearerAuthType = "Bearer"
)

type Client struct {
	*resty.Client

	username string
	password string
	token    string
	auth     struct {
		mode    string
		server  string
		service string
		scope   string
	}
}

func New() *Client {
	return &Client{Client: resty.New()}
}

func (c *Client) Login() {
	url := "?service=${AUTH_SERVICE}&scope=repository:${repo}:pull"
}

func (c *Client) GetManifest(name, reference string) (*Manifest, error) {
	manifest := &Manifest{}
	request := c.R()
	_, err := request.
		SetHeader("accept", "application/vnd.docker.distribution.manifest.v2+json").
		SetHeader("accept", "application/vnd.docker.distribution.manifest.list.v2+json").
		SetHeader("accept", "application/vnd.docker.distribution.manifest.v1+json").
		SetAuthToken(c.token).
		SetResult(manifest).
		Post(fmt.Sprintf("/v2/%s/manifests/%s", name, reference))
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (c *Client) GetLayer(name, digest, output string) error {
	request := c.R()
	_, err := request.
		SetHeader("accept", "application/vnd.docker.distribution.manifest.v2+json").
		SetAuthToken(c.token).
		SetOutput(output).
		Post(fmt.Sprintf("/v2/%s/blobs/%s", name, digest))
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Ping() error {
	res, err := c.R().Get("/v2")
	if err != nil {
		return err
	}
	authenticate := res.Header().Get("Www-Authenticate")
	c.auth.mode = BasicAuthType
	if strings.HasPrefix(authenticate, BearerAuthType) {
		c.auth.mode = BearerAuthType
	}
	authInfo := strings.Split(authenticate, " ")
	if len(authInfo) < 1 {
		return nil
	}
	c.auth.mode = authInfo[0]

	var asn []string
	if len(authInfo) > 1 {
		asn = strings.Split(authInfo[1], ",")
	}
	if len(asn) < 1 {
		return nil
	}
	for _, v := range asn {
		ks := strings.Split(v, "=")
		if len(ks) > 1 {
			if ks[0] == "realm" {
				c.auth.server = ks[1]
			}
			switch ks[0] {
			case "realm":
				c.auth.server = ks[1]
			case "service":
				c.auth.service = ks[1]
			case "scope":
				c.auth.scope = ks[1]
			}
		}
	}
	return nil
}