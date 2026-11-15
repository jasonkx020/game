package authclient

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	appID      string
	appSecret  string
	httpClient *http.Client
}

func New(baseURL, appID, appSecret string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		appID:      appID,
		appSecret:  appSecret,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type CreateAdminAccountResult struct {
	AccountID int64  `json:"account_id"`
	Phone     string `json:"phone"`
	Nickname  string `json:"nickname"`
	Role      string `json:"role"`
	Status    string `json:"status"`
}

func (c *Client) CreateAdminAccount(ctx context.Context, adminToken, phone, nickname, role string) (*CreateAdminAccountResult, error) {
	body, _ := json.Marshal(map[string]string{
		"phone": phone, "nickname": nickname, "role": role,
	})
	var out CreateAdminAccountResult
	if err := c.do(ctx, http.MethodPost, "/v1/admin/auth/accounts", adminToken, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SetAccountRole(ctx context.Context, adminToken string, accountID int64, role string) error {
	body, _ := json.Marshal(map[string]string{"role": role})
	path := fmt.Sprintf("/v1/admin/auth/accounts/%d/role", accountID)
	return c.do(ctx, http.MethodPatch, path, adminToken, body, nil)
}

func (c *Client) SetAccountStatus(ctx context.Context, adminToken string, accountID int64, status, reason string) error {
	body, _ := json.Marshal(map[string]string{"status": status, "reason": reason})
	path := fmt.Sprintf("/v1/admin/auth/accounts/%d/status", accountID)
	return c.do(ctx, http.MethodPatch, path, adminToken, body, nil)
}

func (c *Client) RevokeSessions(ctx context.Context, adminToken string, accountID int64) error {
	path := fmt.Sprintf("/v1/admin/auth/accounts/%d/revoke-sessions", accountID)
	return c.do(ctx, http.MethodPost, path, adminToken, []byte("{}"), nil)
}

func (c *Client) do(ctx context.Context, method, path, adminToken string, body []byte, out any) error {
	if body == nil {
		body = []byte{}
	}
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	auth := ""
	if adminToken != "" {
		auth = "Bearer " + adminToken
		req.Header.Set("Authorization", auth)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	contentSHA := sha256Hex(body)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := randomNonce()
	canonical := strings.Join([]string{
		method,
		u.Path,
		canonicalQuery(u.Query()),
		ts,
		nonce,
		contentSHA,
		auth,
	}, "\n")
	mac := hmac.New(sha256.New, []byte(c.appSecret))
	mac.Write([]byte(canonical))
	sig := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set("X-App-Id", c.appID)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Content-SHA256", contentSHA)
	req.Header.Set("X-Signature", sig)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	respBody, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return fmt.Errorf("auth %s %s: %d %s", method, path, res.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if out != nil && len(respBody) > 0 {
		return json.Unmarshal(respBody, out)
	}
	return nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func randomNonce() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func canonicalQuery(q url.Values) string {
	if len(q) == 0 {
		return ""
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		vs := q[k]
		sort.Strings(vs)
		for _, v := range vs {
			parts = append(parts, k+"="+url.QueryEscape(v))
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, "&")
}
