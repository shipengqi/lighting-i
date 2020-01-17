package client

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"

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
	auth     struct {
		token   string
		mode    string
		server  string
		service string
	}
}

func New() *Client {
	return &Client{Client: resty.New()}
}

func (c *Client) SetUsername(username string)  {
	c.username = username
}

func (c *Client) SetPassword(password string) {
	c.password = password
}

func (c *Client) SetSecureSkip(skip bool) {
	if skip {
		c.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
}

// Ping ping registry and get authenticate info
func (c *Client) Ping() error {
	res, err := c.R().
		Get("/v2/")
	if err != nil {
		return err
	}
	authenticate := res.Header().Get("Www-Authenticate")
	fmt.Printf("ping %s ...\n", c.HostURL)
	c.auth.mode = BasicAuthType
	if strings.HasPrefix(authenticate, BearerAuthType) {
		c.auth.mode = BearerAuthType
	}
	fmt.Printf("ping %s OK\n", c.HostURL)
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
				c.auth.server = strings.Trim(ks[1], "\"")
			case "service":
				c.auth.service = strings.Trim(ks[1], "\"")
			}
		}
	}
	return nil
}

// GetAuthToken get token with scope
func (c *Client) GetAuthToken(repo string) error {
	if c.auth.mode == BearerAuthType {
		authToken := &AuthToken{}
		request := c.R()
		if c.username != "" && c.password != "" {
			request = request.SetBasicAuth(c.username, c.password)
		}
		_, err := request.
			SetResult(authToken).
			SetQueryParam("service", c.auth.service).
			SetQueryParam("scope", fmt.Sprintf("repository:%s:pull", repo)).
			Get(c.auth.server)
		if err != nil {
			return err
		}
		if authToken.Token == "" {
			return fmt.Errorf("token is null")
		}
		c.auth.token = authToken.Token
		return nil
	}

	if c.auth.mode == BasicAuthType {
		if c.username == "" || c.password == "" {
			return fmt.Errorf("bad credential")
		}
		c.auth.token = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.username, c.password)))
		return nil
	}

	return fmt.Errorf("upsupport auth type %s", c.auth.mode)
}

// GetManifest get manifest of image
func (c *Client) GetManifest(name, reference string) (*Manifest, error) {
	manifest := &Manifest{Image: ImageRepo{name, reference}}
	if err := c.GetAuthToken(name); err != nil {
		return nil, err
	}
	request := c.R()
	_, err := request.
		SetHeader("accept", "application/vnd.docker.distribution.manifest.v2+json").
		SetAuthToken(c.auth.token).
		SetResult(manifest).
		Get(fmt.Sprintf("/v2/%s/manifests/%s", name, reference))
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

// GetLayerBlobs get blobs of image layer digest
func (c *Client) GetBlobs(name, digest, output string) error {
	if err := c.GetAuthToken(name); err != nil {
		return err
	}
	request := c.R()
	_, err := request.
		SetAuthToken(c.auth.token).
		SetOutput(output).
		Get(fmt.Sprintf("/v2/%s/blobs/%s", name, digest))
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) GetLayersOfManifest(m Manifest) error {
	if len(m.Layers) < 1 {
		return nil
	}
	var wg sync.WaitGroup
	wg.Add(len(m.Layers))
	for _, l := range m.Layers {
		go func(l Layer) {
			defer wg.Done()
			o := fmt.Sprintf("%s/%s.tar.gz", ImageDateFolderPath, strings.Split(l.Digest, ":")[1])
			err := c.GetBlobs(mr.ImageName, l.Digest, o)
		}(l)
	}
	wg.Wait()
	return nil
}