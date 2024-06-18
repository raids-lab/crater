package service

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"webdav/dao/model"
	"webdav/response"

	"github.com/gin-gonic/gin"
)

func CompressDir(srcDir, destZip string) error {
	zipFile, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	err = filepath.Walk(srcDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(fi)
		if err != nil {
			return err
		}

		// 设置文件的相对路径
		header.Name, err = filepath.Rel(srcDir, file)
		if err != nil {
			return err
		}

		if header.Name == "." {
			return nil
		}

		if fi.IsDir() {
			header.Name += "/"
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if !fi.IsDir() {
			fileData, err := os.ReadFile(file)
			if err != nil {
				return err
			}

			_, err = writer.Write(fileData)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func UncompressDir(srcZip, destDir string) error {
	zipReader, err := zip.OpenReader(srcZip)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	err = os.MkdirAll(destDir, os.FileMode(defaultFolderPerm))
	if err != nil {
		return err
	}

	for _, file := range zipReader.File {
		filePath := filepath.Join(filepath.Clean(destDir), filepath.Clean(file.Name))

		if file.FileInfo().IsDir() {
			err = os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}
		err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		inFile, err := file.Open()
		if err != nil {
			return err
		}

		_, err = io.CopyN(outFile, inFile, int64(file.UncompressedSize64))
		outFile.Close()
		inFile.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

type DatasetName struct {
	Name string `json:"name"`
}

func CopyFile(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	param := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/dataset/create")
	permission := GetPermission(param, jwttoken, c)
	if permission == model.NotAllowed {
		c.String(http.StatusUnauthorized, "Unauthorized 1")
		return
	}

	if param == "" || param == "/" {
		response.Error(c, "can't create dataset", response.NotSpecified)
		return
	}
	var datasetname DatasetName
	err = c.ShouldBind(&datasetname)
	if err != nil {
		response.BadRequestError(c, err.Error())
		return
	}
	pathPart := strings.FieldsFunc(datasetname.Name, func(s rune) bool { return s == '/' })
	if !strings.HasPrefix(datasetname.Name, "/") || len(pathPart) <= 1 {
		response.BadRequestError(c, "bad filepath")
		return
	}
	sourceDir := "/crater" + param
	destDir := "/crater" + datasetname.Name
	zipFilePath := sourceDir + ".zip"

	err = CompressDir(sourceDir, zipFilePath)
	if err != nil {
		fmt.Println("Error compressing directory:", err)
		return
	}
	fmt.Println(sourceDir, "Directory compressed successfully!")
	err = UncompressDir(zipFilePath, destDir)
	if err != nil {
		fmt.Println("Error uncompressing directory:", err)
		return
	}
	err = os.Remove(zipFilePath)
	if err != nil {
		fmt.Println("删除zip文件时出错:", err)
		return
	}
	fmt.Println(destDir, "Directory uncompressed successfully!")
}

func RegisterDataset(r *gin.Engine) {
	r.Handle("POST", "/api/ss/dataset/create/*path", CopyFile)
}
