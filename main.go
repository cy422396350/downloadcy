package main

import (
	"github.com/cy422396350/downloadcy/download"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"runtime"
	"strconv"
)

func main() {
	// 先确定机器的cpu核心数
	cpuNum := runtime.NumCPU()

	// 用cli库来定义命令行的运行
	app := &cli.App{
		Name:        "downloader",
		HelpName:    "File concurrency downloader help",
		Usage:       "File concurrency downloader",
		UsageText:   "",
		ArgsUsage:   "",
		Version:     "",
		Description: "",
		Commands:    nil,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "url",
				Aliases:  []string{"u"},
				Usage:    "Url to download",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "output filename",
			},
			&cli.StringFlag{
				Name:    "cpunumbers",
				Aliases: []string{"c"},
				Usage:   "cpu numbers",
				Value:   strconv.Itoa(cpuNum),
			},
		},
		Action: func(c *cli.Context) error {
			strURL := c.String("url")
			filename := c.String("output")
			concurrency := c.Int("cpunumbers")
			// 初始化下载器
			return download.NewDownloader(concurrency).Download(strURL, filename)
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
