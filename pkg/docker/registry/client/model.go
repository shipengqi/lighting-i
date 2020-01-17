package client

type AuthToken struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	IssuedAt    string `json:"issued_at"`
}

type Manifest struct {
	SchemaVersion int            `json:"schemaVersion"`
	MediaType     string         `json:"mediaType"`
	Config        ManifestConfig `json:"config"`
	Layers        []Layer        `json:"layers"`
}

type Layer struct {
	MediaType string `json:"mediaType"`
	Size      int    `json:"size"`
	Digest    string `json:"digest"`
}

type ManifestConfig Layer