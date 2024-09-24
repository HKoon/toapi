package adaptor

import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "github.com/gin-gonic/gin"
    "github.com/songquanpeng/one-api/common/client"
    "github.com/songquanpeng/one-api/relay/meta"
    "io"
    "net/http"
    "os"
)

// 从环境变量中获取 guidelines 和 preprompt
var guidelines = os.Getenv("GUIDELINES")
var preprompt = os.Getenv("PREPROMPT")

func SetupCommonRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) {
    req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
    req.Header.Set("Accept", c.Request.Header.Get("Accept"))
    if meta.IsStream && c.Request.Header.Get("Accept") == "" {
        req.Header.Set("Accept", "text/event-stream")
    }
}

func modifyRequestBody(requestBody io.Reader) (io.Reader, error) {
    bodyBytes, err := io.ReadAll(requestBody)
    if err != nil {
        return nil, err
    }

    var requestBodyMap map[string]interface{}
    err = json.Unmarshal(bodyBytes, &requestBodyMap)
    if err != nil {
        return nil, err
    }

    messages, ok := requestBodyMap["messages"].([]interface{})
    if !ok {
        return nil, errors.New("messages field is not a valid array")
    }

    for _, msg := range messages {
        msgMap, ok := msg.(map[string]interface{})
        if !ok {
            continue
        }
        if msgMap["role"] == "system" {
            content, ok := msgMap["content"].(string)
            if !ok {
                continue
            }
            msgMap["content"] = preprompt + content + guidelines
        }
    }

    modifiedBodyBytes, err := json.Marshal(requestBodyMap)
    if err != nil {
        return nil, err
    }

    return bytes.NewReader(modifiedBodyBytes), nil
}

func DoRequestHelper(a Adaptor, c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
    modifiedRequestBody, err := modifyRequestBody(requestBody)
    if err != nil {
        return nil, fmt.Errorf("modify request body failed")
    }

    fullRequestURL, err := a.GetRequestURL(meta)
    if err != nil {
        return nil, fmt.Errorf("get request url failed")
    }

    req, err := http.NewRequest(c.Request.Method, fullRequestURL, modifiedRequestBody)
    if err != nil {
        return nil, fmt.Errorf("new request failed")
    }
    err = a.SetupRequestHeader(c, req, meta)
    if err != nil {
        return nil, fmt.Errorf("setup request header failed")
    }
    resp, err := DoRequest(c, req)
    if err != nil {
        return nil, fmt.Errorf("do request failed")
    }
    return resp, nil
}

func DoRequest(c *gin.Context, req *http.Request) (*http.Response, error) {
    resp, err := client.HTTPClient.Do(req)
    if err != nil {
        return nil, err
    }
    if resp == nil {
        return nil, errors.New("resp is nil")
    }
    _ = req.Body.Close()
    _ = c.Request.Body.Close()
    return resp, nil
}
