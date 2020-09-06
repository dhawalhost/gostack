package handlers

/**
 * @author Dhawal Dyavanpalli <dhawalhost@gmail.com>
 * @file Description
 * @desc Created on 2020-08-31 4:28:02 pm
 * @copyright Crearosoft
 */

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DhawalDN/gostack/server/utility"

	"github.com/DhawalDN/gostack/server/dao"
	"github.com/DhawalDN/gostack/server/helpers"

	"github.com/DhawalDN/gostack/server/models"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// MaxUploadSize :
const MaxUploadSize = 100 * 1024 * 1024 // 2 mb
// UploadPath :
const UploadPath = "/home/dhost/cdn/data"

// UploadFileHandler :
func UploadFileHandler(c *gin.Context) {
	// validate file size
	w := c.Writer
	c.Request.Body = http.MaxBytesReader(w, c.Request.Body, MaxUploadSize)
	if err := c.Request.ParseMultipartForm(MaxUploadSize); err != nil {
		renderError(w, "FILE_TOO_BIG", http.StatusBadRequest)
		return
	}
	resultDataStr := ""
	dataToStoreInDBStr := ""
	token := c.GetHeader("Authorization")
	if strings.Trim(token, "") != "" {
		login, _ := helpers.GetUserNameFromToken(c)
		dataToStoreInDBStr, _ = sjson.Set(dataToStoreInDBStr, "username", login.UserName)
		resultDataStr, _ = sjson.Set(resultDataStr, "isAuthorized", false)
	} else {
		dataToStoreInDBStr, _ = sjson.Set(dataToStoreInDBStr, "username", "unknown")
	}
	dataToStoreInDBStr, _ = sjson.Set(dataToStoreInDBStr, "createdOn", time.Now().Unix())
	dataToStoreInDBStr, _ = sjson.Set(dataToStoreInDBStr, "isExpired", false)
	// parse and validate file and post parameters
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		renderError(w, "INVALID_FILE", http.StatusBadRequest)
		return
	}
	defer file.Close()
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		renderError(w, "INVALID_FILE", http.StatusBadRequest)
		return
	}
	// check file type, detectcontenttype only needs the first 512 bytes
	detectedFileType := http.DetectContentType(fileBytes)
	switch detectedFileType {
	case "image/jpeg", "image/jpg":
	case "image/gif", "image/png":
	case "image/svg+xml":
	case "image/webp":
	case "application/pdf":
		break
	default:
		renderError(w, "INVALID_FILE_TYPE", http.StatusBadRequest)
		return
	}
	// fileName := randToken(12)
	fileName := c.Request.FormValue("name")
	fileEndings, err := mime.ExtensionsByType(detectedFileType)
	if err != nil {
		renderError(w, "CANT_READ_FILE_TYPE", http.StatusInternalServerError)
		return
	}
	dateDir := helpers.GetDateForPath()
	newPath := filepath.Join(UploadPath, dateDir, fileName+fileEndings[0])
	fmt.Printf("FileType: %s, File: %s\n", detectedFileType, newPath)

	// write file
	_ = os.MkdirAll(UploadPath+"/"+dateDir, 0755)
	newFile, err := os.Create(newPath)

	if err != nil {
		fmt.Println(err)
		renderError(w, "Creation Error : CANT_WRITE_FILE", http.StatusInternalServerError)
		return
	}
	defer helpers.CreateThumbnail(newPath, fileName)
	if _, err := newFile.Write(fileBytes); err != nil || newFile.Close() != nil {
		renderError(w, "CANT_WRITE_FILE", http.StatusInternalServerError)
		return
	}
	imageID := utility.GetGUID()
	dataToStoreInDBStr, _ = sjson.Set(dataToStoreInDBStr, "imageId", imageID)
	dataToStoreInDBStr, _ = sjson.Set(dataToStoreInDBStr, "fileName", fileName+fileEndings[0])
	dataToStoreInDBStr, _ = sjson.Set(dataToStoreInDBStr, "relativepath", "/"+models.ProjectID+"/images/"+dateDir+"/"+imageID)
	dataToStoreInDBStr, _ = sjson.Set(dataToStoreInDBStr, "thumbnail", "/"+models.ProjectID+"/images/"+dateDir+"/"+imageID)
	dataToStoreInDB := gjson.Parse(dataToStoreInDBStr)
	models.FC.Set(imageID, dataToStoreInDB.Value())
	err = dao.CdnDAO.Insert(dataToStoreInDB.Value())
	if err != nil {
		fmt.Println(err)
	}
	resultDataStr, _ = sjson.Set(resultDataStr, "result.relativePath", "/"+models.ProjectID+"/images/"+dateDir+"/"+imageID)
	resultData := gjson.Parse(resultDataStr)
	// byt, _ := json.Marshal(resultData.Value())
	c.JSON(200, resultData.Value())
	// w.Write([]byte("SUCCESS"))

}

// GetUploadedFileData - To fetch userwise uploaded data
func GetUploadedFileData(c *gin.Context) {
	rawBytes, _ := c.GetRawData()
	rawData := gjson.ParseBytes(rawBytes)
	aggr := gjson.ParseBytes([]byte(fmt.Sprintf(`[
		{"$match":{"username":"%s"}},
		{"$sort":{"createdOn":-1}},
		{"$skip":%d}
		{"$limit":%d}
		]`, rawData.Get("username").String(), rawData.Get("skip").Int(), rawData.Get("limit").Int())))
	rs, _ := dao.CdnDAO.GetAggregateData(aggr.Value())
	c.JSON(200, rs.Value())

}

// DownloadHandler - to Display image on request
func DownloadHandler(c *gin.Context) {
	dir := c.Param("directory")
	imageID := c.Param("imageId")
	// filePath := fmt.Sprintf("%v", c.Request.URL)
	// filePath = UploadPath +strings.Replace(filePath, models.ProjectID+"/images/", "/", 1)
	imageRef, isFound := models.FC.Get(imageID)
	b, _ := json.Marshal(imageRef)
	imgRef := gjson.ParseBytes(b)
	// imageRef = imageRef.(map[string]interface{})
	if isFound {
		filePath := filepath.Join(UploadPath, dir, imgRef.Get("fileName").String())
		c.File(filePath)
		return
	}
	c.JSON(404, gin.H{
		"status": "Not Found",
		"reason": "Link may be expired or invalid",
	})
}
func renderError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(message))
}

func randToken(len int) string {
	b := make([]byte, len)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
