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
	shopID     string
	secretKey  string
	apiURL     string
	httpClient *http.Client
}

// NewClient создаёт новый клиент ЮKassa
func NewClient(shopID, secretKey string) *Client {
	return &Client{
		shopID:     shopID,
		secretKey:  secretKey,
		apiURL:     "https://api.yookassa.ru/v3",
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
	auth := base64.StdEncoding.EncodeToString([]byte(c.shopID + ":" + c.secretKey))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// CreatePayment отправляет запрос на создание платежа с использованием payment_token
func (c *Client) CreatePayment(reqParams CreatePaymentRequest) (*CreatePaymentResponse, error) {
	req, err := c.newRequest("POST", "/payments", reqParams)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, errors.New("unexpected status: " + resp.Status)
	}

	var paymentResp CreatePaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&paymentResp); err != nil {
		return nil, err
	}
	return &paymentResp, nil
}
