package paymentprovider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type Client struct {
	publicKey  string
	secretKey  string
	apiURL     string
	httpClient *http.Client
}

func NewClient(publicKey, secretKey string) *Client {
	return &Client{
		publicKey:  publicKey,
		secretKey:  secretKey,
		apiURL:     "https://api.cloudpayments.ru",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) newRequest(method, path string, body interface{}) (*http.Request, error) {
	url := c.apiURL + path
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		return nil, err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(c.publicKey + ":" + c.secretKey))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// Метод для создания платёжного метода (tokenization)
func (c *Client) CreatePaymentMethod(req CreatePaymentMethodRequest) (*CreatePaymentMethodResponse, error) {
	httpReq, err := c.newRequest("POST", "/payments/cards/post3ds", req)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, errors.New("unexpected status: " + resp.Status)
	}
	var result CreatePaymentMethodResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}