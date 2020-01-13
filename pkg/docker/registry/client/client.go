package client

import (
	"fmt"
	"github.com/go-resty/resty/v2"
)

type Client struct {
	*resty.Client
}

func New() *Client {
	return &Client{resty.New()}
}

func (c *Client) GetManifest(name, reference string) (*Manifest, error) {
	manifest := &Manifest{}
	request := c.R()
	_, err := request.
		SetHeader("accept", "application/vnd.docker.distribution.manifest.v2+json").
		SetHeader("accept", "application/vnd.docker.distribution.manifest.list.v2+json").
		SetHeader("accept", "application/vnd.docker.distribution.manifest.v1+json").
		SetAuthToken("").
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
		SetAuthToken("").
		SetOutput(output).
		Post(fmt.Sprintf("/v2/%s/blobs/%s", name, digest))
	if err != nil {
		return err
	}
}