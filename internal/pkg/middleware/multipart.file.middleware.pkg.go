package middleware

// import (
// 	"io"
// 	"net/http"
// 	"strconv"
// 	"go-boilerplate/internal/common/enum"
// 	_type "go-boilerplate/internal/common/type"
// 	"go-boilerplate/internal/pkg/helper"

// 	"github.com/gin-gonic/gin"
// )

// type FieldOpts struct {
// 	Name string
// 	Max  int
// 	Min  int
// }

// func MultipartFormMiddleware(fields []FieldOpts) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		send := c.MustGet("send").(func(r *_type.Response))

// 		form, err := c.MultipartForm()
// 		if err != nil {
// 			send(helper.ParseResponse(&_type.Response{
// 				Code:    http.StatusInternalServerError,
// 				Message: "Failed retrieving files",
// 				Error:   err,
// 			}))
// 			return
// 		}

// 		bufferedFiles := make(_type.BufferedFiles)

// 		for _, field := range fields {
// 			files := form.File[field.Name]
// 			if files == nil {
// 				continue
// 			}

// 			for _, fileHeader := range files {
// 				file, err := fileHeader.Open()
// 				if err != nil {
// 					send(helper.ParseResponse(&_type.Response{
// 						Code:    http.StatusInternalServerError,
// 						Message: "Failed reading file",
// 						Error:   err,
// 					}))
// 					return
// 				}

// 				fileBuffer, err := io.ReadAll(file)
// 				if err != nil {
// 					send(helper.ParseResponse(&_type.Response{
// 						Code:    http.StatusInternalServerError,
// 						Message: "Failed reading file content",
// 						Error:   err,
// 					}))
// 					return
// 				}
// 				err = file.Close()
// 				if err != nil {
// 					send(helper.ParseResponse(&_type.Response{
// 						Code:    http.StatusInternalServerError,
// 						Message: "Failed close file",
// 						Error:   err,
// 					}))
// 					return
// 				}

// 				bufferedFile := _type.BufferedFile{
// 					MediaType:    field.Name,
// 					OriginalName: fileHeader.Filename,
// 					Encoding:     "7bit",
// 					MimeType:     fileHeader.Header.Get("Content-Type"),
// 					Size:         int(fileHeader.Size),
// 					Buffer:       fileBuffer,
// 				}

// 				// add validate
// 				if field.Name == enum.IMAGE.ToString() {
// 					isValid := enum.IMAGE.IsValidImage(&bufferedFile)
// 					if !isValid {
// 						send(helper.ParseResponse(&_type.Response{
// 							Code:    http.StatusBadRequest,
// 							Message: "Please upload a valid image file",
// 							Error:   err,
// 						}))
// 						return
// 					}
// 				}
// 				if field.Name == enum.VIDEO.ToString() {
// 					isValid := enum.VIDEO.IsValidVideo(&bufferedFile)
// 					if !isValid {
// 						send(helper.ParseResponse(&_type.Response{
// 							Code:    http.StatusBadRequest,
// 							Message: "Please upload a valid video file",
// 							Error:   err,
// 						}))
// 						return
// 					}
// 				}

// 				bufferedFiles[field.Name] = append(bufferedFiles[field.Name], bufferedFile)
// 			}
// 		}

// 		for _, field := range fields {
// 			if len(bufferedFiles[field.Name]) < field.Min {
// 				send(helper.ParseResponse(&_type.Response{
// 					Code:    http.StatusBadRequest,
// 					Message: "Minimum " + field.Name + " is " + strconv.Itoa(field.Min),
// 				}))
// 				return
// 			}
// 			if len(bufferedFiles[field.Name]) > field.Max {
// 				send(helper.ParseResponse(&_type.Response{
// 					Code:    http.StatusBadRequest,
// 					Message: "Maximum " + field.Name + " is " + strconv.Itoa(field.Max),
// 				}))
// 				return
// 			}
// 		}

// 		c.Set("bufferedFiles", bufferedFiles)
// 		c.Next()
// 	}
// }
