package client

import (
	"encoding/base64"
	"fmt"
	"github.com/go-resty/resty/v2"
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
}

func New() *Client {
	return &Client{Client: resty.New()}
}

func (c *Client) GetManifest(name, reference string) (*Manifest, error) {
	manifest := &Manifest{}
	request := c.R()
	_, err := request.
		SetHeader("accept", "application/vnd.docker.distribution.manifest.v2+json").
		SetHeader("accept", "application/vnd.docker.distribution.manifest.list.v2+json").
		SetHeader("accept", "application/vnd.docker.distribution.manifest.v1+json").
		SetHeader("Authorization", c.token).
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
		SetHeader("Authorization", c.token).
		SetOutput(output).
		Post(fmt.Sprintf("/v2/%s/blobs/%s", name, digest))
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) SetClientToken() {
	if c.username != "" && c.password != "" {
		basic := fmt.Sprintf("%s:%s", c.username, c.password)
		c.token = fmt.Sprintf("%s %s", BasicAuthType, base64.StdEncoding.EncodeToString([]byte(basic)))
	}
	c.token = fmt.Sprintf("%s %s", BearerAuthType, "")
}