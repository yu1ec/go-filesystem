package qiniu

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Suggestion string

const (
	SuggestionPass   Suggestion = "pass"
	SuggestionReview Suggestion = "review"
	SuggestionBlock  Suggestion = "block"
)

const (
	ImageSensorAPI = "https://ai.qiniuapi.com/v3/image/censor"
)

type Censor struct {
	qn *QiniuFilesystem
}

func NewCensor(qn *QiniuFilesystem) *Censor {
	return &Censor{
		qn: qn,
	}
}

// CheckImageByURI 检测图片
// 参数:
// uri: 图片URI 支持qiniu:///和http和data:application/octet-stream;开头的base64(建议用CheckImageData方法)
// scenes: 检测场景
func (c *Censor) CheckImageByURI(uri string, scenes ...string) (Suggestion, []string, error) {
	if uri == "" {
		return SuggestionBlock, nil, fmt.Errorf("参数为空")
	}

	if !strings.HasPrefix(uri, "qiniu:///") && !strings.HasPrefix(uri, "http") && !strings.HasPrefix(uri, "data:application/octet-stream;base64,") {
		return SuggestionBlock, nil, fmt.Errorf("图片地址协议非法")
	}

	if len(scenes) == 0 {
		scenes = []string{"pulp", "terror", "politician"}
	}

	requestBody := map[string]any{
		"data": map[string]any{
			"uri": uri,
		},
		"params": map[string]any{
			"scenes": scenes,
		},
	}

	bodyJson, _ := json.Marshal(requestBody)
	contentReader := bytes.NewBuffer(bodyJson)

	req, err := http.NewRequest("POST", ImageSensorAPI, contentReader)
	if err != nil {
		return SuggestionBlock, nil, fmt.Errorf("构建请求失败:%w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 构建鉴权
	accessToken, err := c.qn.mac.SignRequestV2(req)
	if err != nil {
		return SuggestionBlock, nil, fmt.Errorf("参数签名失败:%w", err)
	}
	req.Header.Set("Authorization", "Qiniu "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, http.ErrHandlerTimeout) {
			// 超时容错，允许通过
			return SuggestionPass, nil, nil
		}
		return SuggestionBlock, nil, fmt.Errorf("请求失败:%w", err)
	}
	defer resp.Body.Close()

	resData, err := io.ReadAll(resp.Body)
	if err != nil {
		return SuggestionBlock, nil, fmt.Errorf("读取返回数据失败:%w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorRet struct {
			Code  int    `json:"code"`
			Error string `json:"error"`
		}

		// 读取错误内容
		err = json.Unmarshal(resData, &errorRet)
		if err != nil {
			return SuggestionBlock, nil, fmt.Errorf("解析错误内容失败:%w", err)
		}
		return SuggestionBlock, nil, fmt.Errorf("请求失败，状态码:%d,返回数据:%s", resp.StatusCode, errorRet.Error)
	}

	var censorRet struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		EntryID string `json:"entry_id"`
		Result  struct {
			Suggestion Suggestion `json:"suggestion"`
			Scenes     map[string]struct {
				Suggestion Suggestion `json:"suggestion"`
				Details    []struct {
					Label    string   `json:"label"`
					Score    float64  `json:"score"`
					Desc     string   `json:"desc"`
					Sublabel []string `json:"sublabel,omitempty"`
				} `json:"details,omitempty"`
			} `json:"scenes"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resData, &censorRet); err != nil {
		return SuggestionBlock, nil, fmt.Errorf("解析返回数据失败:%w", err)
	}

	if censorRet.Code != 200 {
		return SuggestionBlock, nil, fmt.Errorf("坚定接口请求失败，错误信息:%s", censorRet.Message)
	}

	var suggestion Suggestion
	switch censorRet.Result.Suggestion {
	case "pass":
		suggestion = SuggestionPass
	case "review":
		suggestion = SuggestionReview
	case "block":
		suggestion = SuggestionBlock
	default:
		suggestion = SuggestionBlock
	}

	// 记录非法原因
	illegalReasons := []string{}
	if suggestion == SuggestionBlock {
		// 取出scenses中非pass的数据
		for k, v := range censorRet.Result.Scenes {
			if v.Suggestion == SuggestionPass {
				continue
			}

			// Find the detail with highest score
			var maxScore float64
			var maxScoreDesc string
			for _, detail := range v.Details {
				if detail.Score > maxScore {
					maxScore = detail.Score
					maxScoreDesc = fmt.Sprintf("场景:%s,描述:%s", k, detail.Desc)
				}
			}
			if maxScoreDesc != "" {
				illegalReasons = append(illegalReasons, maxScoreDesc)
			}
		}

		return suggestion, illegalReasons, nil
	}

	return suggestion, nil, nil
}

// CheckImageData 检测图片
// 参数:
// data: 图片数据
// scenes: 检测场景
func (c *Censor) CheckImageData(data []byte, scenes ...string) (Suggestion, []string, error) {
	base64Data := base64.StdEncoding.EncodeToString(data)
	return c.CheckImageByURI("data:application/octet-stream;base64,"+base64Data, scenes...)
}
