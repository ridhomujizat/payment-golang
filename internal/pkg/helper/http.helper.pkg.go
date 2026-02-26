package helper

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"go-boilerplate/internal/common/enum"
	_type "go-boilerplate/internal/common/type"
	"go-boilerplate/internal/pkg/logger"
	"strings"
	"time"
)

type HTTPAPIResponse struct {
	StatusCode int         `json:"status_code"`
	Headers    http.Header `json:"headers"`
	Data       interface{} `json:"data"`
}

type HTTPRequestPayload struct {
	Method enum.HTTPMethodEnum
	URL    string
	Body   interface{}
	Params map[string]string
}

type HTTPRequestConfig struct {
	Ctx       context.Context
	Headers   http.Header
	Auth      *BasicAuthConfig
	HTTPAgent *http.Transport
}

type BasicAuthConfig struct {
	Username string
	Password string
}

func HTTPRequest(
	payload *HTTPRequestPayload,
	config *HTTPRequestConfig,
) (*HTTPAPIResponse, error) {
	requestBody, err := handleRequestBody(payload, config)
	if err != nil {
		logger.Debug.Println("Error handling request body:", err.Error())
		return nil, err
	}

	req, client, err := prepareRequest(payload, requestBody, config)
	if err != nil {
		logger.Debug.Println("Error preparing request:", err.Error())
		return nil, err
	}
	return executeRequest(req, client)
}

func handleRequestBody(payload *HTTPRequestPayload, config *HTTPRequestConfig) (io.Reader, error) {
	var requestBody io.Reader
	var err error

	if payload.Method == enum.GET {
		return nil, nil
	} else {
		contentType := config.Headers.Get("Content-Type")
		logger.Error.Printf("[DEBUG] handleRequestBody - Content-Type: '%s'", contentType)
		logger.Error.Printf("[DEBUG] handleRequestBody - Expected MultipartForm: '%s'", enum.MultipartForm.ToString())

		switch contentType {
		case enum.ApplicationXform.ToString():
			if payload.Body != nil {
				requestBody, err = createFormURLEncodedBody(payload.Body)
			}
		case enum.MultipartForm.ToString():
			if payload.Body != nil {
				var ct string
				requestBody, ct, err = createMultipartBody(payload.Body)
				config.Headers.Set("Content-Type", ct)
				logger.Error.Printf("[DEBUG] handleRequestBody - New Content-Type after multipart: '%s'", ct)
			}
		case enum.ApplicationJSON.ToString():
			requestBody, err = createJSONBody(payload.Body)
		case "":
			config.Headers.Set("Content-Type", enum.ApplicationJSON.ToString())
			requestBody, err = createJSONBody(payload.Body)
		default:
			logger.Error.Printf("[DEBUG] handleRequestBody - Unsupported content type: '%s'", contentType)
			return nil, errors.New("unsupported content type")
		}
	}

	return requestBody, err
}

func prepareRequest(payload *HTTPRequestPayload, body io.Reader, config *HTTPRequestConfig) (*http.Request, *http.Client, error) {
	req, err := http.NewRequestWithContext(config.Ctx, payload.Method.ToString(), payload.URL, body)
	if err != nil {
		return nil, nil, err
	}

	for key, values := range config.Headers {
		req.Header[key] = append(req.Header[key], values...)
	}

	if config.Auth != nil {
		req.SetBasicAuth(config.Auth.Username, config.Auth.Password)
	}

	client := &http.Client{Timeout: 1 * time.Minute}

	if config.HTTPAgent != nil {
		client.Transport = config.HTTPAgent
	}

	return req, client, nil
}

func executeRequest(req *http.Request, client *http.Client) (*HTTPAPIResponse, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result, err := parseResponseBody(resp)
	if err != nil {
		return nil, err
	}

	return &HTTPAPIResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Data:       result,
	}, nil
}

func parseResponseBody(resp *http.Response) (interface{}, error) {
	contentType := resp.Header.Get("Content-Type")
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result interface{}
	switch {
	case strings.Contains(contentType, "application/json"):
		if err := json.Unmarshal(responseBody, &result); err != nil {
			return nil, err
		}

	case strings.Contains(contentType, "text/plain"), strings.Contains(contentType, "text/html"):
		result = string(responseBody)

	case strings.Contains(contentType, "application/xml"), strings.Contains(contentType, "text/xml"):
		var xmlResult interface{}
		if err := xml.Unmarshal(responseBody, &xmlResult); err != nil {
			return nil, err
		} else {
			result = xmlResult
		}

	case strings.Contains(contentType, "application/octet-stream"), strings.Contains(contentType, "image/"):
		result = responseBody

	case strings.Contains(contentType, "application/x-www-form-urlencoded"):
		parsedForm, err := url.ParseQuery(string(responseBody))
		if err != nil {
			return nil, err
		} else {
			result = parsedForm
		}

	default:
		result = responseBody
	}

	return result, nil
}

func createJSONBody(body interface{}) (io.Reader, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(jsonData), nil
}

func createFormURLEncodedBody(body interface{}) (io.Reader, error) {
	formData, ok := body.(map[string]string)
	if !ok {
		return nil, errors.New("body must be a map[string]string for form-urlencoded content type")
	}
	values := url.Values{}
	for key, value := range formData {
		values.Set(key, value)
	}
	return strings.NewReader(values.Encode()), nil
}

func createMultipartBody(body interface{}) (io.Reader, string, error) {
	formData, ok := body.(map[string]interface{})
	if !ok {
		return nil, "", errors.New("body must be a map[string]interface{} for multipart/form-data content type")
	}
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for key, value := range formData {
		switch v := value.(type) {
		case string:
			_ = writer.WriteField(key, v)
		case []byte:
			part, err := writer.CreateFormFile(key, key)
			if err != nil {
				return nil, "", err
			}
			_, err = part.Write(v)
			if err != nil {
				return nil, "", err
			}
		case []_type.BufferedFile:
			for _, v := range v {
				fileField, err := writer.CreatePart(textproto.MIMEHeader{
					"Content-Disposition": []string{fmt.Sprintf(`form-data; name=%q; filename=%q`, key, v.OriginalName)},
					"Content-Type":        []string{v.MimeType},
					"Content-Encoding":    []string{v.Encoding},
					"Content-Length":      []string{fmt.Sprintf("%d", v.Size)},
				})
				if err != nil {
					return nil, "", err
				}
				_, err = fileField.Write(v.Buffer)
				if err != nil {
					return nil, "", err
				}
			}
		case _type.BufferedFile:
			fileField, err := writer.CreatePart(textproto.MIMEHeader{
				"Content-Disposition": []string{fmt.Sprintf(`form-data; name=%q; filename=%q`, key, v.OriginalName)},
				"Content-Type":        []string{v.MimeType},
				"Content-Encoding":    []string{v.Encoding},
				"Content-Length":      []string{fmt.Sprintf("%d", v.Size)},
			})
			if err != nil {
				return nil, "", err
			}
			_, err = fileField.Write(v.Buffer)
			if err != nil {
				return nil, "", err
			}
		default:
			return nil, "", errors.New("unsupported multipart data type")
		}
	}

	err := writer.Close()
	if err != nil {
		return nil, "", err
	}
	return &buf, writer.FormDataContentType(), nil
}

func ConvertMultipartToBufferedFile(file multipart.File, header *multipart.FileHeader) (_type.BufferedFile, error) {
	defer file.Seek(0, 0) // Reset posisi file pointer

	buffer, err := io.ReadAll(file)
	if err != nil {
		return _type.BufferedFile{}, err
	}

	return _type.BufferedFile{
		MediaType:    "file", // kamu bisa sesuaikan, misalnya "image", "document", dll
		OriginalName: header.Filename,
		Encoding:     "binary", // atau isi dengan "base64" jika nanti diencode
		MimeType:     header.Header.Get("Content-Type"),
		Size:         int(header.Size),
		Buffer:       buffer,
	}, nil
}
