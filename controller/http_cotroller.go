package controller

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go_demo/common"
	"go_demo/model"
	"gorm.io/gorm"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var ch = make(chan int, 1)

func Download(ctx *gin.Context) {
	startTime := time.Now()
	fileUrl := ctx.PostForm("fileUrl")

	url, err := DownloadURL(fileUrl)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		log.Printf("文件下载完成耗时: %f second", time.Now().Sub(startTime).Seconds())
		return
	}

	ctx.JSON(200, gin.H{
		"code": 200,
		"data": gin.H{"url": url},
		"msg":  "文件下载完成",
	})
}

func DownloadURL(url string) (string, error) {
	filesize, fileName, err := GetHead(url)
	if err != nil {
		return "", err
	}
	wd, err := os.Getwd()
	destPath := filepath.Join(wd, fileName)

	//存储目地的文件
	file, err := os.OpenFile(destPath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	eachSize := filesize / 10
	db := common.InitDB()
	var seek model.SeekOffset
	db.First(&seek)
	if seek.ID == 0 {
		newSeek := &model.SeekOffset{
			Num:  0,
			From: 0,
			To:   eachSize,
		}
		db.Create(&newSeek)
	}
	for i := 0; i < 10; i++ {
		go wget(i)
	}

	for i := 0; i < 10; i++ {
		//接收信号
		wput(url, eachSize, file, db)
	}
	return destPath, err
}

func wget(index int) {
	ch <- index
}

func wput(url string, eachSize int, file *os.File, db *gorm.DB) {
	index := <-ch
	request, err := GetRequest("GET", url)
	if err != nil {
		return
	}
	//创建偏移数据
	var seek model.SeekOffset
	if index != 0 {
		db.Where("num", index).First(&seek)
		if seek.ID == 0 {
			db.Model(&seek).Order("id DESC").First(&seek)
			from := seek.To + 1
			to := from + eachSize

			log.Printf("开始[%d]下载from:%d to:%d\n", index, from, to)
			request.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", from, to))
			resp, err := http.DefaultClient.Do(request)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			file.Seek(int64(from), io.SeekStart)
			io.Copy(file, resp.Body)

			saveSeek := &model.SeekOffset{
				Num:  index,
				From: from,
				To:   to,
			}
			db.Save(&saveSeek)
		} else {
			from := seek.From
			to := from + eachSize

			log.Printf("开始[%d]下载from:%d to:%d\n", seek.Num, from, to)
			request.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", from, to))
			resp, err := http.DefaultClient.Do(request)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			file.Seek(int64(from), io.SeekStart)
			io.Copy(file, resp.Body)

			db.Update("current", to)
		}
	} else {
		db.Where("num", index).First(&seek)
		if seek.ID != 0 {
			log.Printf("开始[%d]下载from:%d to:%d\n", seek.Num, seek.From, seek.To)
			request.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", seek.From, seek.To))
			resp, err := http.DefaultClient.Do(request)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			file.Seek(int64(seek.From), io.SeekStart)
			io.Copy(file, resp.Body)

			updateSeek := &model.SeekOffset{
				Num:  index,
				From: seek.From,
				To:   seek.To,
			}
			db.Updates(&updateSeek)
		}
	}
}

func GetHead(url string) (int, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return 0, "", errors.New(fmt.Sprintf("Can't process, response is %v", resp.StatusCode))
	}

	//检查是否支持断点续传
	if resp.Header.Get("Accept-Ranges") != "bytes" {
		return 0, "", errors.New("服务器不支持文件断点续传")
	}

	contentDisposition := resp.Header.Get("Content-Disposition")

	var fileName string
	if contentDisposition != "" {
		_, params, err := mime.ParseMediaType(contentDisposition)

		if err != nil {
			panic(err)
		}
		name := params["filename"]

		fileName = name
	}
	originName := filepath.Base(resp.Request.URL.Path)
	fileName = originName

	fileSize, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	return fileSize, fileName, err
}

func GetRequest(method, url string) (*http.Request, error) {
	r, err := http.NewRequest(
		method,
		url,
		nil,
	)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "mojocn")
	return r, nil
}
