package download

import (
	"fmt"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
)

type Downloader struct {
	CpuNumbers int
	Pro        *progressbar.ProgressBar
}

func NewDownloader(cpuNumbers int) *Downloader {
	return &Downloader{CpuNumbers: cpuNumbers}
}

func (d *Downloader) Download(url, filename string) error {
	/**
	 *	确定文件的最后的名字,没有就用url最后的名字
	 **/
	if len(filename) == 0 {
		filename = path.Base(url)
	}
	// 查看文件是否可以支持多次下载
	resp, err := http.Head(url)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK && resp.Header.Get("Accept-Ranges") == "bytes" {
		return d.MultiDownload(url, filename, int(resp.ContentLength))
	}
	return d.SingleDownload(url, filename)
}

// 并发下载文件
func (d *Downloader) MultiDownload(url, filename string, contentLen int) error {
	// 返回的文件长度/cpu数量得到单次获取文件长度
	partSize := contentLen / d.CpuNumbers

	// 创建文件目录（根据文件名截取的，也可以其他方式)
	partDir := d.getPartDir(filename)
	// 创建目录
	os.Mkdir(partDir, 0777)
	defer os.Remove(partDir)

	// 开启协程
	var wg sync.WaitGroup
	// 开cpu逻辑数个goroutine
	wg.Add(d.CpuNumbers)

	// 初始化进度条
	bar := progressbar.NewOptions(contentLen,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(100),
		progressbar.OptionSetDescription("downloading file..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	d.Pro = bar
	// 初始化从文件头开始
	rangeStart := 0
	// 开始开goroutine
	for i := 0; i < d.CpuNumbers; i++ {
		go func(i, rangeStart int) {
			defer wg.Done()

			// 结束处是文件头加长度
			rangeEnd := rangeStart + partSize

			// 最后一个是最大的len
			if i == d.CpuNumbers-1 {
				rangeEnd = contentLen
			}
			d.downloadPartial(url, filename, rangeStart, rangeEnd, i, d.Pro)
		}(i, rangeStart)
		rangeStart += partSize + 1
	}
	wg.Wait()
	err := d.merge(filename)
	if err != nil {
		return err
	}
	return nil
}

func (d *Downloader) merge(filename string) error {
	// 打开目标文件
	destfile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer destfile.Close()

	for i := 0; i < d.CpuNumbers; i++ {
		srcFileName := d.getPartFileName(filename, i)
		srcFile, err := os.Open(srcFileName)
		if err != nil {
			return err
		}
		io.Copy(destfile, srcFile)
		srcFile.Close()
		os.Remove(srcFileName)
	}
	return nil
}

func (d *Downloader) SingleDownload(url, filename string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	d.Pro = progressbar.NewOptions(
		int(resp.ContentLength),
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetDescription("downloading..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(io.MultiWriter(f, d.Pro), resp.Body, buf)
	return err
}

// 下载方法
func (d *Downloader) downloadPartial(url, filename string, rangeStart, rangeEnd, i int, bar *progressbar.ProgressBar) {
	if rangeStart >= rangeEnd {
		return
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	flags := os.O_CREATE | os.O_WRONLY
	partFile, err := os.OpenFile(d.getPartFileName(filename, i), flags, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer partFile.Close()
	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(io.MultiWriter(partFile, bar), resp.Body, buf)
	if err != nil {
		if err == io.EOF {
			return
		}
		log.Fatal(err)
	}
}

// 获取目录名
func (d *Downloader) getPartDir(filename string) string {
	return strings.SplitN(filename, ".", 2)[0]
}

// 获取文件名
func (d *Downloader) getPartFileName(filename string, partNumber int) string {
	partDir := d.getPartDir(filename)
	return fmt.Sprintf("%s%s-%d", partDir, filename, partNumber)
}
