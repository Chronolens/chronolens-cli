package clcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
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

	if resp.StatusCode == http.StatusForbidden {
		fmt.Println("Password or username incorrect")
		os.Exit(1)
	}

	decoder := json.NewDecoder(resp.Body)

	var tokens Tokens

	err = decoder.Decode(&tokens)
	if err != nil {
		return err
	}
	api.tokens = tokens
	return nil
}

func (api *API) Register(username, password string) error {
	endpoint := fmt.Sprintf("%v/register", api.base_url)
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

	if resp.StatusCode == http.StatusForbidden {
		fmt.Println("A User with that username already exists")
		os.Exit(1)
	}
	fmt.Printf("Created user %v successfully\n", username)
	return nil
}

func (api API) Upload(path, checksum, timestamp, mimeType string) (int, error) {
	endpoint := fmt.Sprintf("%v/image/upload", api.base_url)

	fileToUpload, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer fileToUpload.Close()

	reader, writer := io.Pipe()

	req, err := http.NewRequest("POST", endpoint, reader)
	if err != nil {
		return 0, err
	}

	formWriter := multipart.NewWriter(writer)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", api.tokens.Access_token))
	req.Header.Add("Timestamp", fmt.Sprintf("%v", timestamp))
	req.Header.Add("Content-Type", formWriter.FormDataContentType())

	go func() {
		defer writer.Close()
		defer formWriter.Close()
		filename := filepath.Base(path)

		headers := textproto.MIMEHeader{}
		headers.Add("Content-Disposition", fmt.Sprintf("form-data; name=\"%v\"; filename=\"%v\"", checksum, filename))
		headers.Add("Content-Type", mimeType)
		fieldWriter, err := formWriter.CreatePart(headers)
		if err != nil {
			writer.CloseWithError(err)
			return
		}

		if _, err := io.Copy(fieldWriter, fileToUpload); err != nil {
			writer.CloseWithError(err)
			return
		}
	}()

	resp, err := api.client.Do(req)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		err = api.RefreshToken()
		if err != nil {
			return 0, err
		}
		return api.Upload(path, checksum, timestamp, mimeType)
	}

	return resp.StatusCode, nil
}

type remoteMedia struct {
	Checksum string `json:"hash,omitempty"`
}

func (api API) SyncFull() ([]remoteMedia, error) {
	endpoint := fmt.Sprintf("%v/sync/full", api.base_url)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", api.tokens.Access_token))
	req.Header.Add("Accept", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		err = api.RefreshToken()
		if err != nil {
			return nil, err
		}
		return api.SyncFull()
	}

	var syncFull []remoteMedia

	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(&syncFull)
	if err != nil {
		return nil, err
	}

	return syncFull, nil
}

func (api *API) RefreshToken() error {
	endpoint := fmt.Sprintf("%v/refresh", api.base_url)

	payload := struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}{
		AccessToken:  api.tokens.Access_token,
		RefreshToken: api.tokens.Refresh_token,
	}

	payload_json, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payload_json))
	if err != nil {
		return err
	}

	resp, err := api.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Failed to refresh token, please run the tool again")
		os.Exit(1)
	}

	decoder := json.NewDecoder(resp.Body)

	var tokens Tokens

	err = decoder.Decode(&tokens)
	if err != nil {
		return err
	}
	api.tokens = tokens
	return nil

}
