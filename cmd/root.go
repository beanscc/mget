package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/beanscc/fetch"
	"github.com/spf13/cobra"
)

var (
	downloadFile string // 需要下载的文件url地址
	gn           int    // goroutine 数量
)

var RootCmd = &cobra.Command{
	Use:     "mget",
	Short:   "mget short text info",
	Long:    "mget long text info",
	Example: "mget --file=http://xxx.xx.xx/a.iso --g=50",
	Run:     rootCmdRun,
	// RunE: func(cmd *cobra.Command, args []string) error {
	//
	//
	// 	return nil
	// },
}

func init() {
	// http://iij.dl.osdn.jp/storage/g/m/ma/manjaro/gnome/18.1.0/manjaro-gnome-18.1.0-stable-x86_64.iso
	// https://vlc.letterboxdelivery.org/vlc/3.0.8/win32/vlc-3.0.8-win32.exe
	// https://forum.manjaro.org/uploads/default/original/3X/e/9/e96048fcca8e097ade7d260c8e71381d9a5ae27a.png
	RootCmd.PersistentFlags().StringVar(&downloadFile, "file", "", "需要下载文件的远程地址")
	RootCmd.PersistentFlags().IntVar(&gn, "g", 1, "并发协程数")
}

func rootCmdRun(cmd *cobra.Command, args []string) {
	remoteURL := strings.TrimSpace(downloadFile)
	if remoteURL == "" {
		showUsage(cmd)
	}

	accept, start, end, total, err := contentRange(remoteURL)
	if err != nil {
		log.Printf("mget: parse content-range failed. err=%v", err)
		showUsage(cmd)
	}

	log.Printf("mget: content-range accpet=%v, start=%v, end=%v, total=%v", accept, start, end, total)

	if gn < 0 {
		log.Printf("mget: invalid gn")
		showUsage(cmd)
	}

	avgRange := total / int64(gn)

	fileRanges := make([]fileRange, 0, gn)
	// 准备 goroutine, 计算每个 goroutine 要下载的分片
	for i := 1; i <= gn; i++ {
		var start, end int64
		if i == 1 {
			start, end = 0, avgRange
		} else {
			start, end = int64(i-1)*avgRange+1, int64(i)*avgRange
		}
		if i == gn {
			end = total - 1
		}

		fileRanges = append(fileRanges, fileRange{start, end})
	}

	// 创建文件
	fileName := getRemoteFileName(remoteURL)
	f, err := os.Create(fileName)
	defer f.Close()
	if err != nil {
		log.Printf("mget: create faile failed. err=%v", err)
		showUsage(cmd)
	}

	var wg sync.WaitGroup
	// 合并文件
	for _, v := range fileRanges {
		wg.Add(1)
		go func(start, end int64) {
			defer func() {
				log.Printf("mget: download range done. [%v-%v]", start, end)
				wg.Done()
			}()
			if err := downloadRange(remoteURL, start, end, f); err != nil {
				log.Printf("mget: download failed. start=%v, end=%v, err=%v", start, end, err)
			}
		}(v.start, v.end)
	}
	wg.Wait()
}

func showUsage(cmd *cobra.Command) {
	cmd.Usage()
	// todo show usage
	os.Exit(-1)
}

type fileRange struct {
	start int64
	end   int64
}

func buildRange(start, end int64) string {
	return fmt.Sprintf("bytes=%d-%d", start, end)
}

// raw: "bytes 0-2341451775/2341451776"
func parseContentRange(raw string) (accept string, start, end, total int64, err error) {
	if raw == "" {
		panic("mget: empty content-range")
	}

	s := strings.Split(raw, " ")
	accept = s[0]

	l := strings.Split(s[1], "/")
	cRange := strings.Split(l[0], "-")

	start, err = strconv.ParseInt(cRange[0], 10, 64)
	if err != nil {
		return
	}

	end, err = strconv.ParseInt(cRange[1], 10, 64)
	if err != nil {
		return
	}

	total, err = strconv.ParseInt(l[1], 10, 64)

	return
}

func contentRange(path string) (accept string, start, end, total int64, err error) {
	resp, err := fetch.Head(context.Background(), path).
		Debug(true).
		SetHeader("Range", "bytes=0-").
		Resp()
	if err != nil {
		log.Printf("mget: download file failed. err=%v", err)
		return
	}

	if resp.StatusCode != http.StatusPartialContent {
		log.Printf("mget: unsupport muitl download")
		return
	}

	contentRange := resp.Header.Get("Content-Range")
	log.Printf("content-range:%v", contentRange)

	accept, start, end, total, err = parseContentRange(contentRange)
	if err != nil {
		log.Printf("mget: parse content-range failed. origin content-range: %q, err=%v", contentRange, err)
		return
	}

	return
}

func getRemoteFileName(path string) string {
	fs := strings.Split(path, "/")
	l := len(fs)
	if l < 1 {
		panic("mget: invalid path")
	}
	return fs[l-1]
}

func downloadRange(remoteURL string, rangeStart, rangeEnd int64, f *os.File) error {
	bs, err := fetch.Get(context.Background(), remoteURL).
		// Debug(true).
		SetHeader("Range", buildRange(rangeStart, rangeEnd)).
		Bytes()
	if err != nil {
		log.Printf("mget: download range failed. err=%v", err)
		return err
	}

	log.Printf("mget: http.Get range succeed. %s", buildRange(rangeStart, rangeEnd))

	n, err := f.WriteAt(bs, rangeStart)
	if n < len(bs) {
		return errors.New("short write to file")
	}
	//f.Close()
	if err != nil {
		return err
	}

	return nil
}
