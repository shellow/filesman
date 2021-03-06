package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/prometheus/common/log"
	"github.com/shellow/filesman"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
		cli.StringFlag{
			Name:  "uppath, up",
			Value: "/files/upload",
			Usage: "upload url path",
		},
		cli.StringFlag{
			Name:  "downpath, dp",
			Value: "/files/download",
			Usage: "download url path",
		},
		cli.StringFlag{
			Name:  "head",
			Value: "",
			Usage: "add head key",
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
					Usage: "file for download",
				},
				cli.StringFlag{
					Name:  "sdir, d",
					Value: "./",
					Usage: "file dir for save",
				},
			},
		},
		{
			Name:     "imgaddpdf",
			Usage:    "image add pdf file",
			Category: "act",
			Action:   imgaddpdf,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "file, f",
					Value: "/tmp/test.pdf",
					Usage: "file for save",
				},
				cli.StringFlag{
					Name:  "pdf",
					Usage: "pdf file for upload",
				},
				cli.StringFlag{
					Name:  "image, i",
					Usage: "image file for upload",
				},
				cli.StringFlag{
					Name:  "page",
					Usage: "page in pdf file",
				},
				cli.StringFlag{
					Name:  "xpos, x",
					Usage: "xpos to image file",
				},
				cli.StringFlag{
					Name:  "ypos, y",
					Usage: "ypos to image file",
				},
				cli.StringFlag{
					Name:  "width, w",
					Usage: "width to image file",
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func head(c *cli.Context) (key, value string) {
	head := c.GlobalString("head")
	kv := strings.Split(head, ":")
	if len(kv) == 2 {
		key = kv[0]
		value = kv[1]
	} else {
		key = ""
	}
	return
}

func upload(c *cli.Context) error {
	murl := c.GlobalString("surl")
	uploadpath := c.GlobalString("up")
	murl = murl + uploadpath

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	file := c.String("file")
	// Add your image file
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	//w.WriteField()
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

	k, v := head(c)
	if !strings.EqualFold(k, "") {
		req.Header.Set(k, v)
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
	downloadpath := c.GlobalString("dp")
	murl = murl + downloadpath
	file := c.String("file")
	murl = murl + "/" + file

	req, err := http.NewRequest("GET", murl, nil)
	if err != nil {
		return err
	}
	k, v := head(c)
	if !strings.EqualFold(k, "") {
		req.Header.Set(k, v)
	}
	req.Header.Set("charset", "UTF-8")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	status := res.Header.Get("status")
	if !strings.EqualFold(status, "ok") {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		fmt.Println(string(body))
		return nil
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

func imgaddpdf(c *cli.Context) error {
	murl := c.GlobalString("surl")
	murl = murl + "/files/imgsignpdfonce"
	pdf := c.String("pdf")
	pdff, err := os.Open(pdf)
	if err != nil {
		return err
	}
	defer pdff.Close()

	file := c.String("file")

	img := c.String("image")
	imgf, err := os.Open(img)
	if err != nil {
		return err
	}
	defer imgf.Close()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile("pdf", pdf)
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, pdff); err != nil {
		return err
	}

	fw1, err := w.CreateFormFile("image", img)
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw1, imgf); err != nil {
		return err
	}

	page := c.String("page")
	iw1, err := w.CreateFormField("page")
	if err != nil {
		return err
	}
	_, err = iw1.Write([]byte(page))
	if err != nil {
		return err
	}

	xpos := c.String("xpos")
	iw2, err := w.CreateFormField("xpos")
	if err != nil {
		return err
	}
	_, err = iw2.Write([]byte(xpos))
	if err != nil {
		return err
	}

	ypos := c.String("ypos")
	iw3, err := w.CreateFormField("ypos")
	if err != nil {
		return err
	}
	_, err = iw3.Write([]byte(ypos))
	if err != nil {
		return err
	}

	width := c.String("width")
	iw4, err := w.CreateFormField("width")
	if err != nil {
		return err
	}
	_, err = iw4.Write([]byte(width))
	if err != nil {
		return err
	}

	w.Close()

	req, err := http.NewRequest("POST", murl, &b)
	if err != nil {
		return err
	}

	k, v := head(c)
	if !strings.EqualFold(k, "") {
		req.Header.Set(k, v)
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("charset", "UTF-8")
	//req.Header.Set("token", token)

	// Submit the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	//fmt.Println(string(body))
	if gjson.Get(string(body), "file").Exists() {
		files := gjson.Get(string(body), "file").String()
		fileb, err := base64.StdEncoding.DecodeString(files)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(file, fileb, 0666)
		if err != nil {
			return err
		}
		fmt.Println("success")
		return nil
	} else {
		fmt.Println("failed")
		return nil
	}

}
