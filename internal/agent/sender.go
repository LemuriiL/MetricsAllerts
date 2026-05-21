package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/LemuriiL/MetricsAllerts/internal/cryptoutil"
	models "github.com/LemuriiL/MetricsAllerts/internal/model"
	"github.com/go-resty/resty/v2"
)

type endpointNotSupportedError struct {
	status int
}

func (e endpointNotSupportedError) Error() string {
	return fmt.Sprintf("endpoint not supported: %d", e.status)
}

type Sender struct {
	serverAddr    string
	key           string
	cryptoKeyPath string
	client        *resty.Client

	pubKey     *rsa.PublicKey
	pubKeyErr  error
	pubKeyOnce sync.Once
}

func NewSender(serverAddr string) *Sender {
	return NewSenderWithKeyAndCryptoKey(serverAddr, "", "")
}

func NewSenderWithKey(serverAddr string, key string) *Sender {
	return NewSenderWithKeyAndCryptoKey(serverAddr, key, "")
}

func NewSenderWithKeyAndCryptoKey(serverAddr string, key string, cryptoKeyPath string) *Sender {
	client := resty.New()

	client.SetTimeout(5 * time.Second)
	client.SetRetryCount(3)
	client.SetRetryWaitTime(time.Second)
	client.SetRetryMaxWaitTime(5 * time.Second)

	client.AddRetryCondition(func(r *resty.Response, err error) bool {
		if err != nil {
			return isRetryableHTTPError(err)
		}

		if r == nil {
			return false
		}

		status := r.StatusCode()

		if status == httpStatusNotFound || status == httpStatusMethodNotAllowed {
			return false
		}

		return status >= 500
	})

	return &Sender{
		serverAddr:    serverAddr,
		key:           key,
		cryptoKeyPath: cryptoKeyPath,
		client:        client,
	}
}

func (s *Sender) Send(metric models.Metrics) error {
	body, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	u, err := url.JoinPath(s.serverAddr, "/update")
	if err != nil {
		return err
	}

	return s.postJSONWithRetry(context.Background(), u, body)
}

func (s *Sender) SendBatch(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	body, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	u, err := url.JoinPath(s.serverAddr, "/updates")
	if err != nil {
		return err
	}

	err = s.postJSONWithRetry(context.Background(), u, body)
	if err == nil {
		return nil
	}

	if isEndpointNotSupported(err) {
		for i := range metrics {
			if err = s.Send(metrics[i]); err != nil {
				return err
			}
		}

		return nil
	}

	return err
}

func (s *Sender) postJSONWithRetry(ctx context.Context, u string, body []byte) error {
	return s.postJSON(ctx, u, body)
}

func (s *Sender) postJSON(ctx context.Context, u string, body []byte) error {
	payload, err := gzipBody(body)
	if err != nil {
		return err
	}

	requestBody := payload
	encryptedKey := ""
	nonce := ""

	if s.cryptoKeyPath != "" {
		publicKey, err := s.getPublicKey()
		if err != nil {
			return err
		}

		requestBody, encryptedKey, nonce, err = cryptoutil.Encrypt(publicKey, payload)
		if err != nil {
			return err
		}
	}

	realIP, err := resolveOutboundIP(u)
	if err != nil {
		return err
	}

	request := s.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip").
		SetHeader("X-Real-IP", realIP).
		SetBody(requestBody)

	if encryptedKey != "" && nonce != "" {
		request.SetHeader(cryptoutil.HeaderEncryptedKey, encryptedKey)
		request.SetHeader(cryptoutil.HeaderNonce, nonce)
	}

	if s.key != "" {
		sum := sha256.Sum256(append(body, []byte(s.key)...))
		request.SetHeader("HashSHA256", hex.EncodeToString(sum[:]))
	}

	response, err := request.Post(u)
	if err != nil {
		return err
	}

	switch response.StatusCode() {
	case httpStatusOK:
		return nil
	case httpStatusNotFound, httpStatusMethodNotAllowed:
		return endpointNotSupportedError{status: response.StatusCode()}
	default:
		return fmt.Errorf("server returned status: %d body=%s", response.StatusCode(), response.String())
	}
}

func (s *Sender) getPublicKey() (*rsa.PublicKey, error) {
	s.pubKeyOnce.Do(func() {
		s.pubKey, s.pubKeyErr = cryptoutil.LoadPublicKey(s.cryptoKeyPath)
	})

	return s.pubKey, s.pubKeyErr
}

func gzipBody(body []byte) ([]byte, error) {
	var buf bytes.Buffer

	writer := gzip.NewWriter(&buf)

	if _, err := writer.Write(body); err != nil {
		_ = writer.Close()
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func resolveOutboundIP(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	hostPort := parsed.Host
	if hostPort == "" {
		return "", errors.New("empty server host")
	}

	if !strings.Contains(hostPort, ":") {
		switch parsed.Scheme {
		case "https":
			hostPort = net.JoinHostPort(hostPort, "443")
		default:
			hostPort = net.JoinHostPort(hostPort, "80")
		}
	}

	conn, err := net.Dial("udp", hostPort)
	if err == nil {
		defer conn.Close()

		addr, ok := conn.LocalAddr().(*net.UDPAddr)
		if ok && addr.IP != nil {
			return addr.IP.String(), nil
		}
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP == nil || ipNet.IP.IsLoopback() {
			continue
		}

		ip4 := ipNet.IP.To4()
		if ip4 != nil {
			return ip4.String(), nil
		}
	}

	return "", errors.New("failed to resolve outbound ip")
}

func isEndpointNotSupported(err error) bool {
	var e endpointNotSupportedError
	return errors.As(err, &e)
}

func isRetryableHTTPError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	if errors.Is(err, io.EOF) {
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}

	msg := err.Error()

	return bytes.Contains([]byte(msg), []byte("connection refused")) ||
		bytes.Contains([]byte(msg), []byte("EOF"))
}

const (
	httpStatusOK               = 200
	httpStatusNotFound         = 404
	httpStatusMethodNotAllowed = 405
)
