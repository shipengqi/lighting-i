package client

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/go-resty/resty/v2"
)

var (
	BasicAuthType  = "Basic"
	BearerAuthType = "Bearer"
	DockerUuidKey  =  "Docker-Upload-Uuid"
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
func (c *Client) GetAuthToken(repo string) (error, string) {
	if c.auth.mode == BearerAuthType {
		authToken := &AuthToken{}
		request := c.R()
		if c.username != "" && c.password != "" {
			request = request.SetBasicAuth(c.username, c.password)
		}
		_, err := request.
			SetResult(authToken).
			SetQueryParam("service", c.auth.service).
			SetQueryParam("scope", fmt.Sprintf("repository:%s:push,pull", repo)).
			Get(c.auth.server)
		if err != nil {
			return err, ""
		}
		if authToken.Token == "" {
			return fmt.Errorf("token is null"), ""
		}
		return nil, authToken.Token
	}

	if c.auth.mode == BasicAuthType {
		if c.username == "" || c.password == "" {
			return fmt.Errorf("bad credential"), ""
		}
		return nil, base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.username, c.password)))
	}

	return fmt.Errorf("upsupport auth type %s", c.auth.mode), ""
}

// ListImageTags listing image tags
func (c *Client) ListImageTags(name string) (*Tags, *Errno) {
	tags := &Tags{}
	err, token := c.GetAuthToken(name)
	if err != nil {
		return tags, &Errno{InternalServerErr.Code, err.Error()}
	}
	request := c.R()
	res, err := request.
		SetAuthToken(token).
		SetResult(tags).
		Get(fmt.Sprintf("/v2/%s/tags/list", name))
	if err != nil {
		return tags, &Errno{InternalServerErr.Code, err.Error()}
	}
	status := handleResponseStatus(res)
	return tags, status
}

// FetchManifest get manifest of image
func (c *Client) FetchManifest(name, reference string) (*Manifest, *Errno) {
	manifest := &Manifest{Image: ImageRepo{Name: name, Tag: reference}}
	err, token := c.GetAuthToken(name)
	if err != nil {
		return manifest, &Errno{InternalServerErr.Code, err.Error()}
	}
	request := c.R()
	res, err := request.
		SetHeader("accept", "application/vnd.docker.distribution.manifest.v2+json").
		SetAuthToken(token).
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
	err, token := c.GetAuthToken(name)
	if err != nil {
		return &Errno{InternalServerErr.Code, err.Error()}
	}
	request := c.R()
	res, err := request.
		SetAuthToken(token).
		SetOutput(output).
		Get(fmt.Sprintf("/v2/%s/blobs/%s", name, digest))
	if err != nil {
		return &Errno{InternalServerErr.Code, err.Error()}
	}
	status := handleResponseStatus(res)
	return status
}

// CheckBlobs check the existence of a layer
func (c *Client) CheckBlobs(name, digest string) *Errno {
	err, token := c.GetAuthToken(name)
	if err != nil {
		return &Errno{InternalServerErr.Code, err.Error()}
	}
	request := c.R()
	res, err := request.
		SetAuthToken(token).
		Head(fmt.Sprintf("/v2/%s/blobs/%s", name, digest))
	if err != nil {
		return &Errno{InternalServerErr.Code, err.Error()}
	}
	status := handleResponseStatus(res)
	return status
}

// StartUpload starting an upload
func (c *Client) StartUpload(name string) *Errno {
	err, token := c.GetAuthToken(name)
	if err != nil {
		return &Errno{InternalServerErr.Code, err.Error()}
	}
	request := c.R()
	res, err := request.
		SetAuthToken(token).
		Post(fmt.Sprintf("/v2/%s/blobs/uploads", name))
	if err != nil {
		return &Errno{InternalServerErr.Code, err.Error()}
	}
	status := handleResponseStatus(res)
	// Set docker uuid
	if status.Code == OK.Code {
		status.Message = res.Header().Get(DockerUuidKey)
	}
	return status
}

// PushBlobs upload a layer
func (c *Client) PushBlobs(name, uuid, digest, path string) *Errno {
	err, token := c.GetAuthToken(name)
	if err != nil {
		return &Errno{InternalServerErr.Code, err.Error()}
	}
	fileBytes, _ := ioutil.ReadFile(path)
	request := c.R()
	res, err := request.
		SetBody(fileBytes).
		SetHeader("Content-Type", "application/octet-stream").
		SetAuthToken(token).
		SetContentLength(true).
		Put(fmt.Sprintf("/v2/%s/blobs/uploads/%s?digest=%s", name, uuid, digest))
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
