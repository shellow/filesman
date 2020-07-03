package filesman

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/minio/sha256-simd"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
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

func (filesman *Filesman) Upload(c *gin.Context) {
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
	newPath := filepath.Join(filesman.Filedir, fileName+fileEndings[0])
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
		"file":   fileName + fileEndings[0],
	})
}

func (filesman *Filesman) Download(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	filename := c.Param("filename")
	filepath := filepath.Join(filesman.Filedir, filename)
	c.File(filepath)
}
