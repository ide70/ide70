package api

import (
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"time"
)

type FileUpload struct {
	fileName    string
	mimeType    string
	totalLength int64
	file        multipart.File
	buf         []byte
	err         error
	ready       bool
	percent     int
	mu          sync.Mutex
	stop        chan bool
}

type BinaryData struct {
	data         *[]byte
}

func (bd *BinaryData) GetData() *[]byte {
	return bd.data
}

type UploadCtx struct {
	U *FileUpload
}

func NewFileUpload(r *http.Request) *FileUpload {
	if r.Method != http.MethodPost {
		logger.Error("upload invalid method:", r.Method)
		return nil
	}

	file, handle, err := r.FormFile("file")
	if err != nil {
		logger.Error("%v", err)
		return nil
	}

	fileUpload := &FileUpload{}
	fileUpload.fileName = handle.Filename
	fileUpload.mimeType = handle.Header.Get("Content-Type")
	fileUpload.totalLength = handle.Size
	fileUpload.file = file

	return fileUpload
}

func (u *FileUpload) Stop() {
	logger.Info("send stop")
	if !u.ready && u.err == nil {
		u.stop <- true
	}
	logger.Info("send stop done")
}

func (u *FileUpload) Start() {
	//logger.Info("Upload launch")
	go func() {
		u.ready = false
		u.err = nil
		u.setPercent(0)
		u.buf = make([]byte, u.totalLength+1)
		u.stop = make(chan bool)
		n := int64(0)
		step := int64(1048576)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer func() { ticker.Stop(); close(u.stop) }()
		for {
			select {
			case <-u.stop:
				return
			case <-ticker.C:
				u.setPercent(int(n * 100 / u.totalLength))
				//logger.Info("set pct:", n, u.totalLength)
			default:
				max := n + step
				if max > u.totalLength+1 {
					max = u.totalLength + 1
				}
				an, e := u.file.Read(u.buf[n:max])
				//logger.Info("rd:", an, e)
				n += int64(an)
				if e == io.EOF {
					u.setPercent(100)
					u.buf = u.buf[:len(u.buf)-1]
					u.setReady()
					logger.Info("Upload finished")
					return
				}
				if e != nil {
					logger.Info("Upload error", e.Error())
					u.err = e
				}
			}
		}
	}()
	//logger.Info("Upload launch done")
}

func (u *FileUpload) setReady() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.ready = true
}

func (u *FileUpload) setPercent(percent int) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.percent = percent
}

func (u *UploadCtx) GetFileName() string {
	return u.U.fileName
}

func (u *UploadCtx) GetMimeType() string {
	return u.U.mimeType
}

func (u *UploadCtx) GetPercent() int {
	u.U.mu.Lock()
	defer u.U.mu.Unlock()
	return u.U.percent
}

func (u *UploadCtx) GetData() *BinaryData {
	u.U.mu.Lock()
	defer u.U.mu.Unlock()
	if u.U.ready {
		return &BinaryData{data: &u.U.buf}
	}
	buf := make([]byte, 0)
	return &BinaryData{data: &buf}
}

func (u *UploadCtx) Finished() bool {
	u.U.mu.Lock()
	defer u.U.mu.Unlock()
	return u.U.ready || u.U.err != nil
}

func (u *UploadCtx) HasError() bool {
	u.U.mu.Lock()
	defer u.U.mu.Unlock()
	return u.U.err != nil
}
