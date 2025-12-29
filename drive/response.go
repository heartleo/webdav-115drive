package drive

import (
	"encoding/json"
	"time"
)

type LoginResp struct {
	Code int `json:"code"`
	Data struct {
		Expire int64  `json:"expire"`
		Link   string `json:"link"`
		UserID int64  `json:"user_id"`
	} `json:"data"`
	Errno   int    `json:"errno"`
	Error   string `json:"error"`
	Message string `json:"message"`
	State   int    `json:"state"`
	Expire  int    `json:"expire"`
}

type DirIDResp struct {
	CategoryID json.Number `json:"id"`
	IsPrivate  json.Number `json:"is_private"`
	State      bool        `json:"state"`
}

type FileListResp struct {
	AreaID     string      `json:"aid"`
	CategoryID json.Number `json:"cid"`
	Count      int64       `json:"count"`
	Cur        int64       `json:"cur"`
	Data       []FileInfo  `json:"data"`
	DataSource string      `json:"data_source"`
	ErrNo      int64       `json:"errNo"`
	Error      string      `json:"error"`
	Limit      int64       `json:"limit"`
	MaxSize    int64       `json:"max_size"`
	MinSize    int64       `json:"min_size"`
	Offset     int64       `json:"offset"`
	Order      string      `json:"order"`
	PageSize   int64       `json:"page_size"`
}

type FileInfo struct {
	AreaID          json.Number `json:"aid"`
	CategoryID      json.Number `json:"cid"`
	FileID          json.Number `json:"fid"`
	ParentID        string      `json:"pid"`
	Name            string      `json:"n"`
	Type            string      `json:"ico"`
	Size            json.Number `json:"s"`
	Sha1            string      `json:"sha"`
	PickCode        string      `json:"pc"`
	CreateTime      json.Number `json:"tp"`
	UpdateTime      json.Number `json:"te"`
	MediaDuration   float64     `json:"play_long"`
	VideoFlag       int         `json:"iv"`
	VideoDefinition int         `json:"vdi"`
}

type File struct {
	IsDir      bool
	FileID     string
	ParentID   string
	Name       string
	Size       int64
	Sha1       string
	PickCode   string
	CreateTime time.Time
	UpdateTime time.Time
}

func (fi *FileInfo) ToFile() *File {
	f := &File{
		FileID:   fi.FileID.String(),
		ParentID: fi.ParentID,
		Name:     fi.Name,
		Sha1:     fi.Sha1,
		PickCode: fi.PickCode,
	}

	fileID, _ := fi.FileID.Int64()
	f.IsDir = fileID == 0

	size, _ := fi.Size.Int64()
	f.Size = size

	createTime, _ := fi.UpdateTime.Int64()
	f.CreateTime = time.Unix(createTime, 0).UTC()

	updateTime, _ := fi.UpdateTime.Int64()
	f.UpdateTime = time.Unix(updateTime, 0).UTC()

	return f
}

type DownloadInfoResp struct {
	Data json.RawMessage `json:"data,omitempty"`
}

type DownloadData struct {
	FileName string      `json:"file_name"`
	FileSize json.Number `json:"file_size"`
	PickCode string      `json:"pick_code"`
	Url      struct {
		Client float64 `json:"rc"`
		OssID  string  `json:"oss_id"`
		Url    string  `json:"url"`
	} `json:"url"`
}

type DownloadInfo map[string]*DownloadData
