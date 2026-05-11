package ws1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/w4l1dcode/wsone2sentinel/config"
	"golang.org/x/oauth2/clientcredentials"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func doAuthRequest(ctx context.Context, ws1AuthLocation, clientID, secret, url, method string, payload interface{}) (respBytes []byte, err error) {
	var reqPayload []byte
	if payload != nil {
		if reqPayload, err = json.Marshal(&payload); err != nil {
			return nil, errors.Wrap(err, "coult not encode request body")
		}
	}

	oauth2Config := clientcredentials.Config{ClientID: clientID, ClientSecret: secret,
		TokenURL: fmt.Sprintf("https://%s.uemauth.vmwservices.com/connect/token", ws1AuthLocation)}
	httpClient := oauth2Config.Client(ctx)
	httpClient.Timeout = time.Second * 30

	req, err := http.NewRequest(method, url, bytes.NewReader(reqPayload))
	req = req.WithContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}

	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "http request failed")
	}

	if resp.StatusCode > 399 {
		respB, _ := io.ReadAll(resp.Body)
		logrus.WithField("response", string(respB)).Warn("invalid response")
		return nil, errors.New("invalid response code: " + strconv.Itoa(resp.StatusCode))
	}

	defer resp.Body.Close()

	if respBytes, err = io.ReadAll(resp.Body); err != nil {
		return nil, errors.New("could not read response body")
	}

	return respBytes, nil
}

func fetchDevicesPage(config *config.Config, ctx context.Context, page *int) (DevicesResponse, error) {
	endpoint := strings.TrimRight(config.WS1.Endpoint, "/") + "/mdm/devices/search"
	if page != nil {
		query := url.Values{}
		query.Set("page", strconv.Itoa(*page))
		endpoint += "?" + query.Encode()
	}

	deviceResponseB, err := doAuthRequest(
		ctx,
		config.WS1.AuthLocation, config.WS1.ClientID, config.WS1.ClientSecret,
		endpoint,
		http.MethodGet,
		nil,
	)
	if err != nil {
		return DevicesResponse{}, err
	}

	var devicesResponse DevicesResponse
	if err := json.Unmarshal(deviceResponseB, &devicesResponse); err != nil {
		return DevicesResponse{}, errors.Wrap(err, "could not deserialize getDevices call")
	}

	return devicesResponse, nil
}

func fetchAllDevices(config *config.Config, ctx context.Context) ([]Devices, error) {
	firstPage, err := fetchDevicesPage(config, ctx, nil)
	if err != nil {
		return nil, err
	}

	allDevices := append([]Devices{}, firstPage.Devices...)
	if firstPage.Total <= len(allDevices) || firstPage.PageSize == 0 {
		return allDevices, nil
	}

	seenUUIDs := make(map[string]struct{}, len(allDevices))
	for _, device := range allDevices {
		if device.UUID != "" {
			seenUUIDs[device.UUID] = struct{}{}
		}
	}

	currentPage := firstPage.Page
	for len(allDevices) < firstPage.Total {
		nextPage := currentPage + 1
		devicesPage, err := fetchDevicesPage(config, ctx, &nextPage)
		if err != nil {
			return nil, errors.Wrapf(err, "could not fetch WS1 devices page %d", nextPage)
		}
		if len(devicesPage.Devices) == 0 {
			break
		}

		added := 0
		for _, device := range devicesPage.Devices {
			if device.UUID != "" {
				if _, exists := seenUUIDs[device.UUID]; exists {
					continue
				}
				seenUUIDs[device.UUID] = struct{}{}
			}

			allDevices = append(allDevices, device)
			added++
		}
		if added == 0 {
			break
		}

		currentPage = devicesPage.Page
	}

	return allDevices, nil
}

func parseWS1Time(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}

	parsed, err := time.Parse("2006-01-02T15:04:05.999999999", raw)
	if err == nil {
		return parsed, nil
	}

	return time.Time{}, fmt.Errorf("unsupported WS1 timestamp format: %s", raw)
}

func GetLogs(config *config.Config, ctx context.Context) ([]Devices, error) {
	devices, err := fetchAllDevices(config, ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch WS1 devices")
	}

	result := make([]Devices, 0, len(devices))

	now := time.Now()
	timeGenerated := now.UTC().Format(time.RFC3339)

	for _, device := range devices {
		userEmail := strings.ToLower(strings.TrimSpace(device.UserEmailAddress))
		device.UserEmailAddress = userEmail
		device.TimeGenerated = timeGenerated

		if _, err := parseWS1Time(device.LastSeen); err != nil {
			logrus.WithError(err).WithField("timestamp", device.LastSeen).
				WithField("device", device.DeviceFriendlyName).Error("could not parse MDM host last seen")
		}

		result = append(result, device)
	}

	return result, nil
}
