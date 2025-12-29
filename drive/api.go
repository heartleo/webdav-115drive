package drive

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	UAKey        = "User-Agent"
	UA115Browser = "Mozilla/5.0 115Browser/23.9.3.2"
	Referer      = "https://115.com/"
)

const (
	CookieDomain = "www.115.com"
	CookieUid    = "UID"
	CookieCid    = "CID"
	CookieSeid   = "SEID"
	CookieKid    = "KID"
)

const (
	ApiLoginCheck  = "https://passportapi.115.com/app/1.0/web/1.0/check/sso"
	ApiGetDirID    = "https://webapi.115.com/files/getid"
	ApiFileList    = "https://webapi.115.com/files"
	ApiDownloadUrl = "https://proapi.115.com/app/chrome/downurl"
)

func (c *Client) LoginCheck() error {
	result := LoginResp{}

	resp, err := c.NewRequest().
		SetQueryParam("_", strconv.Itoa(int(time.Now().UnixMilli()))).
		SetResult(&result).
		Get(ApiLoginCheck)
	if err != nil {
		return err
	}

	if result.State != 0 {
		slog.Warn("api login check failed", slog.Any("code", result.Code), slog.Any("error", result.Error))
		return errors.New("api login check failed")
	}

	if resp.StatusCode() >= http.StatusBadRequest {
		slog.Warn("api login check failed", slog.Any("status", resp.Status()))
		return errors.New("api login check failed")
	}

	slog.Debug("api login check success", slog.Any("status", resp.Status()),
		slog.Any("user_id", result.Data.UserID))

	return nil
}

func (c *Client) DirID(dirPath string) (string, error) {
	result := DirIDResp{}

	path := strings.TrimPrefix(dirPath, "/")
	if path == "" {
		path = "0"
	}

	resp, err := c.NewRequest().
		SetQueryParam("path", path).
		SetResult(&result).
		ForceContentType("application/json;charset=UTF-8").
		Get(ApiGetDirID)
	if err != nil {
		return "", err
	}

	if resp.StatusCode() >= http.StatusBadRequest {
		slog.Warn("api dir id failed", slog.Any("status", resp.Status()))
		return "", errors.New("api dir id failed")
	}

	slog.Debug("api dir id success", slog.String("dirPath", dirPath), slog.Any("result", result))

	return result.CategoryID.String(), nil
}

func (c *Client) FileList(dirID string) ([]*File, error) {
	var files []*File

	pageSize := int64(64)
	offset := int64(0)

	for {
		resp, err := apiFiles(c.rc, dirID, pageSize, offset)
		if err != nil {
			return nil, err
		}

		for i := range resp.Data {
			files = append(files, resp.Data[i].ToFile())
		}

		offset = resp.Offset + pageSize
		if offset >= resp.Count {
			break
		}

		time.Sleep(200 * time.Millisecond)
	}

	slog.Debug("file list success", slog.Any("dirID", dirID), slog.Any("files", len(files)))

	return files, nil
}

func apiFiles(rc *resty.Client, dirID string, pageSize int64, offset int64) (*FileListResp, error) {
	result := FileListResp{}

	params := map[string]string{
		"aid":              "1",
		"cid":              dirID,
		"o":                "user_ptime",
		"asc":              "0",
		"offset":           strconv.Itoa(int(offset)),
		"limit":            strconv.Itoa(int(pageSize)),
		"show_dir":         "1",
		"snap":             "0",
		"record_open_time": "1",
		"format":           "json",
		"fc_mix":           "0",
	}

	resp, err := rc.R().SetQueryParams(params).
		SetResult(&result).
		ForceContentType("application/json;charset=UTF-8").
		Get(ApiFileList)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() >= http.StatusBadRequest {
		slog.Warn("api files failed", slog.Any("status", resp.Status()))
		return nil, errors.New("api files failed")
	}

	slog.Debug("api files success", slog.Any("dirID", dirID), slog.Any("offset", offset),
		slog.Any("pageSize", pageSize),
		slog.Any("result", result))

	return &result, err
}

func (c *Client) DownloadInfo(pickCode string) (*DownloadData, error) {
	result := DownloadInfoResp{}

	params, err := json.Marshal(map[string]string{"pickcode": pickCode})
	if err != nil {
		return nil, err
	}

	key := EncryptKey()
	data := Encrypt(params, key)

	var resp *resty.Response
	resp, err = c.NewRequest().
		SetQueryParam("t", strconv.Itoa(int(time.Now().Unix()))).
		SetFormData(map[string]string{"data": data}).
		ForceContentType("application/json;charset=UTF-8").
		SetResult(&result).
		Post(ApiDownloadUrl)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() >= http.StatusBadRequest {
		slog.Warn("download info failed", slog.Any("status", resp.Status()))
		return nil, errors.New("download info failed")
	}

	slog.Debug("api download info success", slog.Any("pickCode", pickCode))

	var dataString string
	err = json.Unmarshal(result.Data, &dataString)
	if err != nil {
		slog.Debug("download data json.Unmarshal failed", slog.Any("error", err))
		return nil, err
	}

	var bs []byte
	bs, err = Decrypt(dataString, key)
	if err != nil {
		slog.Debug("download result data decode failed", slog.Any("error", err))
		return nil, err
	}

	downloadInfo := DownloadInfo{}
	if err = json.Unmarshal(bs, &downloadInfo); err != nil {
		slog.Debug("download info encrypt.Decrypt failed", slog.Any("error", err))
		return nil, err
	}

	for _, downloadData := range downloadInfo {
		fileSize, _ := downloadData.FileSize.Int64()
		if fileSize <= 0 {
			return nil, errors.New("download file empty")
		}
		return downloadData, nil
	}

	return nil, errors.New("download file not found")
}
