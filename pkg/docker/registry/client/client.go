package client

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
)

var (
	BasicAuthType  = "Basic"
	BearerAuthType = "Bearer"
)

var (
	// Common errors
	OK                = &Errno{Code: 200, Message: "OK"}
	BadRequestErr     = &Errno{Code: 400, Message: "Bad Request"}
	UnauthorizedErr   = &Errno{Code: 401, Message: "Unauthorized."}
	ForbiddenErr      = &Errno{Code: 403, Message: "Forbidden."}
	NotFoundErr       = &Errno{Code: 404, Message: "Not Found."}
	TooManyRequestErr = &Errno{Code: 429, Message: "Too Many Requests"}
	InternalServerErr = &Errno{Code: 500, Message: "Internal server error"}
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

func (c *Client) SetUsername(username string) {
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

// FetchManifest get manifest of image
func (c *Client) FetchManifest(name, reference string) (*Manifest, *Errno) {
	manifest := &Manifest{Image: ImageRepo{Name: name, Tag: reference}}
	if err := c.GetAuthToken(name); err != nil {
		return manifest, &Errno{InternalServerErr.Code, err.Error()}
	}
	request := c.R()
	res, err := request.
		SetHeader("accept", "application/vnd.docker.distribution.manifest.v2+json").
		SetAuthToken(c.auth.token).
		SetResult(manifest).
		Get(fmt.Sprintf("/v2/%s/manifests/%s", name, reference))
	if err != nil {
		return manifest, &Errno{InternalServerErr.Code, err.Error()}
	}
	status := handleResponseStatus(res)
	return manifest, status
}

// FetchBlobs get blobs of image layer digest
func (c *Client) FetchBlobs(name, digest, output string) *Errno {
	if err := c.GetAuthToken(name); err != nil {
		return &Errno{InternalServerErr.Code, err.Error()}
	}
	request := c.R()
	res, err := request.
		SetAuthToken(c.auth.token).
		SetOutput(output).
		Get(fmt.Sprintf("/v2/%s/blobs/%s", name, digest))
	if err != nil {
		return &Errno{InternalServerErr.Code, err.Error()}
	}
	status := handleResponseStatus(res)
	return status
}


func handleResponseStatus(res *resty.Response) *Errno {
	if res == nil {
		return InternalServerErr
	}
	switch res.StatusCode() {
	case BadRequestErr.Code:
		return BadRequestErr
	case UnauthorizedErr.Code:
		return BadRequestErr
	case ForbiddenErr.Code:
		return BadRequestErr
	case NotFoundErr.Code:
		return BadRequestErr
	case TooManyRequestErr.Code:
		return BadRequestErr
	}
	return OK
}
