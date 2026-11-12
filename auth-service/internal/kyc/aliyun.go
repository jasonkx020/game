package kyc

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// AliyunRPCClient implements Id2MetaVerify via Aliyun OpenAPI RPC (signature v1.0).
type AliyunRPCClient struct {
	AccessKeyID     string
	AccessKeySecret string
	Endpoint        string
	HTTP            *http.Client
}

func NewAliyunRPC(ak, sk, endpoint string) *AliyunRPCClient {
	if endpoint == "" {
		endpoint = "https://cloudauth.aliyuncs.com"
	}
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = "https://" + endpoint
	}
	return &AliyunRPCClient{
		AccessKeyID:     ak,
		AccessKeySecret: sk,
		Endpoint:        strings.TrimRight(endpoint, "/"),
		HTTP:            &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *AliyunRPCClient) Id2MetaVerify(ctx context.Context, name, idNo string) (string, error) {
	params := map[string]string{
		"Action":           "Id2MetaVerify",
		"Version":          "2019-03-07",
		"Format":           "JSON",
		"AccessKeyId":      c.AccessKeyID,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
		"SignatureNonce":   fmt.Sprintf("%d", time.Now().UnixNano()),
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"ParamType":        "normal",
		"UserName":         name,
		"IdentifyNum":      idNo,
	}
	params["Signature"] = c.sign("POST", params)

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint+"/", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var out struct {
		Code         string `json:"Code"`
		Message      string `json:"Message"`
		ResultObject struct {
			BizCode string `json:"BizCode"`
		} `json:"ResultObject"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("aliyun decode: %w body=%s", err, string(body))
	}
	if out.Code != "200" && out.Code != "Success" {
		return "", fmt.Errorf("aliyun Id2MetaVerify: code=%s msg=%s", out.Code, out.Message)
	}
	return out.ResultObject.BizCode, nil
}

func (c *AliyunRPCClient) sign(method string, params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "Signature" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var canonical []string
	for _, k := range keys {
		canonical = append(canonical, percentEncode(k)+"="+percentEncode(params[k]))
	}
	stringToSign := method + "&" + percentEncode("/") + "&" + percentEncode(strings.Join(canonical, "&"))
	mac := hmac.New(sha1.New, []byte(c.AccessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func percentEncode(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(url.QueryEscape(s), "+", "%20"), "*", "%2A"), "%7E", "~")
}

func NewVerifier(mode, ak, sk, endpoint string) Verifier {
	if mode == "live" && ak != "" && sk != "" {
		return &AliyunVerifier{Client: NewAliyunRPC(ak, sk, endpoint)}
	}
	return NewMock()
}
