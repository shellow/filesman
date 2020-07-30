package filesman

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/minio/sha256-simd"
	"github.com/shellow/keyman"
	"github.com/tjfoc/gmsm/sm3"
	"github.com/unidoc/unipdf/v3/creator"
	pdf "github.com/unidoc/unipdf/v3/model"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const FILEKEY = "uploadfile"

type Filesman struct {
	Filedir       string
	MaxUploadSize int64
}

func NewFilesman() *Filesman {
	filesman := new(Filesman)
	filesman.Filedir = "/tmp"
	filesman.MaxUploadSize = 2 * 1024 * 1024 // 2 mb
	return filesman
}

func BuildFilename(addr string, filename string) string {
	return addr + "-" + filename
}

func GenFilename(c *gin.Context, filename string) (string, error) {
	addr, err := keyman.TokenToAddrStr(c.GetHeader("token"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid token",
		})
		return "", err
	}
	filename = BuildFilename(addr, filename)
	return filename, nil
}

func (filesman *Filesman) Upload(c *gin.Context) (filename string) {
	c.Header("Access-Control-Allow-Origin", "*")
	if err := c.Request.ParseMultipartForm(filesman.MaxUploadSize); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not parse multipart form",
		})
		return
	}

	// parse and validate file and post parameters
	file, fileHeader, err := c.Request.FormFile(FILEKEY)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid file",
		})
		return
	}
	defer file.Close()
	// Get and print out file size
	fileSize := fileHeader.Size
	// validate file size
	if fileSize > filesman.MaxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "File too big",
		})
		return
	}
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid file",
		})
		return
	}

	// check file type, detectcontenttype only needs the first 512 bytes
	detectedFileType := http.DetectContentType(fileBytes)
	switch detectedFileType {
	case "image/jpeg", "image/jpg":
	case "image/gif", "image/png":
	case "application/pdf":
		break
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid file type",
		})
		return
	}
	fileName := fmt.Sprintf("%x", sha256.Sum256(fileBytes))
	fileEndings, err := mime.ExtensionsByType(detectedFileType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Can not read file type",
		})
		return
	}

	filename = fileName + fileEndings[0]
	filenameReal, err := GenFilename(c, filename)
	if err != nil {
		return
	}

	newPath := filepath.Join(filesman.Filedir, filenameReal)
	//fmt.Printf("FileType: %s, File: %s\n", detectedFileType, newPath)

	// write file
	newFile, err := os.Create(newPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Can not write file",
		})
		return
	}
	defer newFile.Close() // idempotent, okay to call twice
	if _, err := newFile.Write(fileBytes); err != nil || newFile.Close() != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Can not write file",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"file":   filename,
	})
	return filename
}

func (filesman *Filesman) Download(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")

	filename := c.Param("filename")
	filename, err := GenFilename(c, filename)
	if err != nil {
		return
	}

	filepath := filepath.Join(filesman.Filedir, filename)
	c.Header("status", "ok")
	c.File(filepath)
}

func (filesman *Filesman) Hash(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	filename := c.Param("filename")

	filename, err := GenFilename(c, filename)
	if err != nil {
		return
	}
	filepath := filepath.Join(filesman.Filedir, filename)

	file, err := os.OpenFile(filepath, os.O_RDONLY, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Can not write file",
		})
		return
	}
	defer file.Close()

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid file",
		})
		return
	}

	hashtype := c.GetHeader("hashtype")
	var hash string
	if strings.EqualFold(hashtype, "sha256") {
		hash = fmt.Sprintf("%x", sha256.Sum256(fileBytes))
	} else if strings.EqualFold(hashtype, "sm3") {
		hash = fmt.Sprintf("%x", sm3.Sm3Sum(fileBytes))
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"hashtype": hashtype,
		"hash":     hash,
	})
	return
}

func AddImageToPdf(inputPath string, outputPath string, imagePath string, pageNum int, xPos float64, yPos float64, iwidth float64) error {

	c := creator.New()

	// Prepare the image.
	img, err := c.NewImageFromFile(imagePath)
	if err != nil {
		return err
	}
	img.ScaleToWidth(iwidth)
	img.SetPos(xPos, yPos)

	// Read the input pdf file.
	f, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	pdfReader, err := pdf.NewPdfReader(f)
	if err != nil {
		return err
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return err
	}

	// Load the pages.
	for i := 0; i < numPages; i++ {
		page, err := pdfReader.GetPage(i + 1)
		if err != nil {
			return err
		}

		// Add the page.
		err = c.AddPage(page)
		if err != nil {
			return err
		}

		// If the specified page, or -1, apply the image to the page.
		if i+1 == pageNum || pageNum == -1 {
			_ = c.Draw(img)
		}
	}

	err = c.WriteToFile(outputPath)
	return err
}

func (filesman *Filesman) ImgAddPdf(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	pdffile, ok := c.GetPostForm("pdf")
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "Params pdf error",
		})
		return
	}

	pdffile, err := GenFilename(c, pdffile)
	if err != nil {
		return
	}

	pdfpath := filepath.Join(filesman.Filedir, pdffile)

	pagestr, ok := c.GetPostForm("page")
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "Params page error",
		})
		return
	}

	page, err := strconv.Atoi(pagestr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "Params page error",
		})
		return
	}

	image, ok := c.GetPostForm("image")
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "Params image error",
		})
		return
	}
	image, err = GenFilename(c, image)
	if err != nil {
		return
	}
	imagepath := filepath.Join(filesman.Filedir, image)

	xposStr, ok := c.GetPostForm("xpos")
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "Params xpos error",
		})
		return
	}
	xpos, err := strconv.ParseFloat(xposStr, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "Params xpos error",
		})
		return
	}

	yposStr, ok := c.GetPostForm("ypos")
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "Params ypos error",
		})
		return
	}
	ypos, err := strconv.ParseFloat(yposStr, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "Params ypos error",
		})
		return
	}

	widthStr, ok := c.GetPostForm("width")
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "Params width error",
		})
		return
	}
	width, err := strconv.ParseFloat(widthStr, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "Params width error",
		})
		return
	}
	hash := sha256.Sum256([]byte(pdffile + image))
	outfile := fmt.Sprintf("%x", hash) + ".pdf"
	outfilepath := filepath.Join(filesman.Filedir, outfile)

	err = AddImageToPdf(pdfpath, outfilepath, imagepath, page, xpos, ypos, width)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "Merge error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"resultfile": outfile,
	})
}
