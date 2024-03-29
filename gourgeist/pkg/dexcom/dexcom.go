package dexcom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"iv2/gourgeist/defs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	appID            = "d89443d2-327c-4a6f-89e5-496bbb0317db"
	baseUrl          = "https://shareous1.dexcom.com/ShareWebServices/Services"
	loginEndpoint    = "General/LoginPublisherAccountByName"
	authEndpoint     = "General/AuthenticatePublisherAccount"
	readingsEndpoint = "Publisher/ReadPublisherLatestGlucoseValues"

	// One day's worth.
	MinuteLimit = 1440
	CountLimit  = 288
)

type Client struct {
	client      *http.Client
	logger      *zap.Logger
	accountName string
	password    string
	sessionID   string
}

type Source interface {
	Readings(ctx context.Context, minutes, maxCount int) ([]*defs.TransformedReading, error)
}

type LoginRequest struct {
	AccountName   string `json:"accountName"`
	Password      string `json:"password"`
	ApplicationID string `json:"applicationId"`
}

type Reading struct {
	WT    string  `json:"WT"` // Not sure what this stands for.
	Value float64 `json:"Value"`
	Trend string  `json:"Trend"`
}

func New(accountName, password string, logger *zap.Logger) *Client {
	return &Client{
		client:      &http.Client{},
		logger:      logger,
		accountName: accountName,
		password:    password,
	}
}

// Readings fetches readings from Dexcom's Share API, and applies a transformation.
// Automatically creates a new session when it expires.
func (c *Client) Readings(ctx context.Context, minutes, maxCount int) ([]*defs.TransformedReading, error) {
	if trs, err := c.readings(ctx, minutes, maxCount); err == nil {
		return trs, nil
	}
	if _, err := c.CreateSession(ctx); err != nil {
		return nil, fmt.Errorf("unable to create dexcom session: %w", err)
	}
	return c.readings(ctx, minutes, maxCount)
}

func (c *Client) CreateSession(ctx context.Context) (string, error) {
	lreq := &LoginRequest{
		AccountName:   c.accountName,
		Password:      c.password,
		ApplicationID: appID,
	}

	b, err := json.Marshal(lreq)
	if err != nil {
		return "", fmt.Errorf("unable to marshal login request: %w", err)
	}
	c.logger.Debug("making login request for sessionID",
		zap.ByteString("request", b),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseUrl+"/"+loginEndpoint, bytes.NewBuffer(b))
	if err != nil {
		return "", fmt.Errorf("unable to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	c.sessionID = strings.Trim(string(body), "\"")

	c.logger.Debug("successfully obtained sessionID",
		zap.String("sessionID", c.sessionID),
	)

	return c.sessionID, nil
}

func (c *Client) readings(ctx context.Context, minutes, maxCount int) ([]*defs.TransformedReading, error) {
	if minutes > MinuteLimit || maxCount > CountLimit {
		return nil, fmt.Errorf("window too large: minutes %d, maxCount %d", minutes, maxCount)
	}

	params := url.Values{
		"sessionId": {c.sessionID},
		"minutes":   {strconv.Itoa(minutes)},
		"maxCount":  {strconv.Itoa(maxCount)},
	}

	c.logger.Debug("making fetch request",
		zap.String("sessionID", c.sessionID),
		zap.Int("minutes", minutes),
		zap.Int("maximum count", maxCount),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseUrl+"/"+readingsEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to send request: %w", err)
	}
	defer resp.Body.Close()

	var readings []*Reading
	err = json.NewDecoder(resp.Body).Decode(&readings)
	if err != nil {
		return nil, fmt.Errorf("unable to decode readings response: %w", err)
	}
	c.logger.Debug("received readings from share API", zap.Int("count", len(readings)))

	trs := make([]*defs.TransformedReading, len(readings))
	for i, r := range readings {
		tr, err := transform(r)
		if err != nil {
			return nil, fmt.Errorf("unable to transform glucose readings: %w", err)
		}
		trs[i] = tr
	}

	return trs, nil
}

func transform(r *Reading) (*defs.TransformedReading, error) {
	parsedTime := strings.Trim(r.WT[4:], "()")
	unix, err := strconv.Atoi(parsedTime)
	if err != nil {
		return nil, fmt.Errorf("unable to convert to int: %w", err)
	}
	return &defs.TransformedReading{
		Time:  time.Unix(int64(unix/1000), 0),
		Mmol:  r.Value / 18,
		Trend: r.Trend,
	}, nil
}
