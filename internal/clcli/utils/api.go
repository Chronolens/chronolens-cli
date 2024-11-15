package clcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

type API struct {
	client   http.Client
	base_url string
	tokens   Tokens
}

type Tokens struct {
	Access_token  string `json:"access_token"`
	Refresh_token string `json:"refresh_token"`
	Expires_at    int64  `json:"expires_at"`
}

func NewAPI(base_url string) API {
	return API{
		client:   http.Client{},
		base_url: base_url,
	}
}

func (api *API) Login(username, password string) error {
	endpoint := fmt.Sprintf("%v/login", api.base_url)
	payload := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: username,
		Password: password,
	}

	payload_json, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payload_json))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	var tokens Tokens

	err = decoder.Decode(&tokens)
	if err != nil {
		return err
	}
	api.tokens = tokens
	return nil
}

func (api API) Upload(path, checksum string) (*http.Response, error) {
	endpoint := fmt.Sprintf("%v/image/upload", api.base_url)

	// Open the file to upload
	fileToUpload, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fileToUpload.Close()

	stat, err := fileToUpload.Stat()
	if err != nil {
		return nil, err
	}

	reader, writer := io.Pipe()

	req, err := http.NewRequest("POST", endpoint, reader)
	if err != nil {
		return nil, err
	}

	// Create a multipart writer for the pipe
	formWriter := multipart.NewWriter(writer)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", api.tokens.Access_token))
	req.Header.Add("Timestamp", fmt.Sprintf("%v", stat.ModTime().UnixMilli()))
	req.Header.Add("Content-Type", formWriter.FormDataContentType())

	// Write to the pipe in a separate goroutine
	go func() {
		defer writer.Close()     // Close the writer to signal EOF
		defer formWriter.Close() // Close the multipart writer

		fieldWriter, err := formWriter.CreateFormFile(checksum, fileToUpload.Name())
		if err != nil {
			writer.CloseWithError(err)
			return
		}

		if _, err := io.Copy(fieldWriter, fileToUpload); err != nil {
			writer.CloseWithError(err)
			return
		}
	}()

	return api.client.Do(req)
}

func (api API) SyncFull() error {
	endpoint := fmt.Sprintf("%v/sync/full", api.base_url)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", api.tokens.Access_token))
	req.Header.Add("Accept", "application/json")

	type remoteMedia struct {
		Checksum  string `json:"hash,omitempty"`
	}

	resp,err := api.client.Do(req)
	if err != nil {
		return err
	}

	var syncFull []remoteMedia

	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(&syncFull)
	if err != nil {
		return err
	}

	for _,media := range syncFull {
		fmt.Printf("%+v\n",media)
	}
	
	return nil
}
