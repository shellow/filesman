package main

import (
	"bytes"
	"fmt"
	"github.com/prometheus/common/log"
	"github.com/shellow/filesman"
	"github.com/urfave/cli"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	app := cli.NewApp()
	app.Name = "filesman client"
	app.Usage = "file client"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		//cli.IntFlag{
		//	Name:  "port, p",
		//	Value: 8000,
		//	Usage: "listening port",
		//},
		cli.StringFlag{
			Name:  "surl, s",
			Value: "http://127.0.0.1",
			Usage: "server url",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:     "upload",
			Usage:    "upload file",
			Category: "act",
			Action:   upload,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "file, f",
					Usage: "file for upload",
				},
			},
		},
		{
			Name:     "download",
			Usage:    "download file",
			Category: "act",
			Action:   download,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "file, f",
					Usage: "file for upload",
				},
				cli.StringFlag{
					Name:  "sdir, d",
					Value: "./",
					Usage: "file dir for save",
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func upload(c *cli.Context) error {
	murl := c.GlobalString("surl")
	murl = murl + "/filesm/upload"

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	file := c.String("file")
	// Add your image file
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	fw, err := w.CreateFormFile(filesman.FILEKEY, file)
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, f); err != nil {
		return err
	}
	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", murl, &b)
	if err != nil {
		return err
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("charset", "UTF-8")
	//req.Header.Set("token", token)

	// Submit the request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	var buf = make([]byte, 1000)

	_, _ = io.ReadFull(res.Body, buf)
	fmt.Println(string(buf))

	// Check the response
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", res.Status)
	}
	return nil
}

func download(c *cli.Context) error {
	murl := c.GlobalString("surl")
	murl = murl + "/filesm/download"
	file := c.String("file")
	murl = murl + "/" + file
	res, err := http.Get(murl)
	if err != nil {
		return err
	}
	sdir := c.String("sdir")
	// 	f, err := os.Create(filePath)
	fpath := filepath.Join(sdir, file)
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	io.Copy(f, res.Body)
	fmt.Println("success")
	return nil
}
