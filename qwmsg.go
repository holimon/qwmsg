package qwmsg

import (
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/holimon/requests"
)

type Config struct {
	Corpid         string
	Corpsecret     string
	Agentid        int
	TokenExpiresin int64
	Retry          int
}

type acToken struct {
	Content  string
	stop     chan bool
	Interval time.Duration
}

type Qwmsg struct {
	Token       *acToken
	Configs     Config
	CommonField map[string]interface{}
}

type CommonField struct {
	ToUser      string
	ToParty     string
	ToTag       string
	AgentId     int
	Enidtrans   bool
	Endupcheck  bool
	Dupinterval int
}

func IF(condition bool, trueval, falseval interface{}) interface{} {
	if condition {
		return trueval
	} else {
		return falseval
	}
}

type MediaType string

const (
	MediaImage MediaType = "image"
	MediaVideo MediaType = "video"
	MediaVoice MediaType = "voice"
	MediaFile  MediaType = "file"
)

type errmsg error

var (
	ErrorJsonUnmarshal errmsg = errors.New("exception occurs when the response message body is decoded as json")
	ErrorDefault       errmsg = errors.New("exception has occurred")
	ErrorStill         errmsg = errors.New("exception still occurs after the request is retried")
)

func mergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}

func (token *acToken) tokenRun(qwmsg *Qwmsg) {
	ticker := time.NewTicker(token.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			qwmsg.wxGetAccessToken()
		case <-token.stop:
			fmt.Println("GC stoped.")
			return
		}
	}
}

func New(configs Config) *Qwmsg {
	qwmsg := &Qwmsg{Configs: configs}
	qwmsg.CommonField = make(map[string]interface{})
	qwmsg.Token = &acToken{stop: make(chan bool), Interval: time.Duration(configs.TokenExpiresin) * time.Second}
	qwmsg.wxGetAccessToken()
	qwmsg.SetCommonField(CommonField{ToUser: "@all", AgentId: configs.Agentid})
	go qwmsg.Token.tokenRun(qwmsg)
	runtime.SetFinalizer(qwmsg, (*Qwmsg).tokenStop)
	return qwmsg
}

func (qwmsg *Qwmsg) tokenStop() {
	qwmsg.Token.stop <- true
}

func (qwmsg *Qwmsg) wxGetAccessToken() error {
	requrl := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", qwmsg.Configs.Corpid, qwmsg.Configs.Corpsecret)
	for try := qwmsg.Configs.Retry; try >= 0; try-- {
		if response, err := requests.Get(requrl); err == nil {
			v := make(map[string]interface{})
			if response.Json(&v) == nil {
				if v["errcode"].(float64) == 0 {
					qwmsg.Token.Content = v["access_token"].(string)
					fmt.Println(qwmsg.Token.Content)
					return nil
				} else {
					return errors.New(v["errmsg"].(string))
				}
			}
		}
	}
	return ErrorStill
}

func (qwmsg *Qwmsg) AccessToken() string {
	return qwmsg.Token.Content
}

func (qwmsg *Qwmsg) SetCommonField(common CommonField) {
	qwmsg.CommonField["touser"] = common.ToUser
	qwmsg.CommonField["toparty"] = common.ToParty
	qwmsg.CommonField["totag"] = common.ToTag
	qwmsg.CommonField["agentid"] = common.AgentId
	// qwmsg.CommonField["enable_id_trans"] = IF(common.Enidtrans, 1, 0)
	qwmsg.CommonField["enable_duplicate_check"] = IF(common.Endupcheck, 1, 0)
	qwmsg.CommonField["duplicate_check_interval"] = common.Dupinterval
}

func (qwmsg *Qwmsg) SendTextMsg(content string, safe bool) error {
	reqdata := mergeMaps(qwmsg.CommonField, map[string]interface{}{
		"msgtype": "text",
		"text":    map[string]string{"content": content},
	})
	if safe {
		reqdata = mergeMaps(reqdata, map[string]interface{}{
			"safe": 1,
		})
	}
	requrl := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", qwmsg.AccessToken())
	for try := qwmsg.Configs.Retry; try >= 0; try-- {
		if response, err := requests.PostJson(requrl, reqdata); err == nil {
			v := make(map[string]interface{})
			if e := response.Json(&v); e == nil {
				if v["errcode"].(float64) == 0 {
					return nil
				} else {
					return errors.New(v["errmsg"].(string))
				}
			} else {
				return e
			}
		}
	}
	return ErrorStill
}

func (qwmsg *Qwmsg) PostMedia(filename string, filetype MediaType) (media_id string, ierr error) {
	requrl := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/media/upload?access_token=%s&type=%s", qwmsg.AccessToken(), filetype)
	for try := qwmsg.Configs.Retry; try >= 0; try-- {
		if response, err := requests.Post(requrl, requests.Files{"media": filename}); err == nil {
			v := make(map[string]interface{})
			if e := response.Json(&v); e == nil {
				if v["errcode"].(float64) == 0 {
					return v["media_id"].(string), nil
				} else {
					return "", errors.New(v["errmsg"].(string))
				}
			} else {
				return "", e
			}
		}
	}
	return "", ErrorStill
}

func (qwmsg *Qwmsg) SendImageMsg(media_id string, safe bool) error {
	reqdata := mergeMaps(qwmsg.CommonField, map[string]interface{}{
		"msgtype": "image",
		"image":   map[string]string{"media_id": media_id},
	})
	if safe {
		reqdata = mergeMaps(reqdata, map[string]interface{}{
			"safe": 1,
		})
	}
	requrl := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", qwmsg.AccessToken())
	for try := qwmsg.Configs.Retry; try >= 0; try-- {
		if response, err := requests.PostJson(requrl, reqdata); err == nil {
			v := make(map[string]interface{})
			if e := response.Json(&v); e == nil {
				if v["errcode"].(float64) == 0 {
					return nil
				} else {
					return errors.New(v["errmsg"].(string))
				}
			} else {
				return e
			}
		}
	}
	return ErrorStill
}

func (qwmsg *Qwmsg) SendFileMsg(media_id string, safe bool) error {
	reqdata := mergeMaps(qwmsg.CommonField, map[string]interface{}{
		"msgtype": "file",
		"file":    map[string]string{"media_id": media_id},
	})
	if safe {
		reqdata = mergeMaps(reqdata, map[string]interface{}{
			"safe": 1,
		})
	}
	requrl := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", qwmsg.AccessToken())
	for try := qwmsg.Configs.Retry; try >= 0; try-- {
		if response, err := requests.PostJson(requrl, reqdata); err == nil {
			v := make(map[string]interface{})
			if e := response.Json(&v); e == nil {
				if v["errcode"].(float64) == 0 {
					return nil
				} else {
					return errors.New(v["errmsg"].(string))
				}
			} else {
				return e
			}
		}
	}
	return ErrorStill
}

func (qwmsg *Qwmsg) SendTextCardMsg(title, description, url string) error {
	reqdata := mergeMaps(qwmsg.CommonField, map[string]interface{}{
		"msgtype": "textcard",
		"textcard": map[string]string{
			"title":       title,
			"description": description,
			"url":         url,
		},
	})
	requrl := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", qwmsg.AccessToken())
	for try := qwmsg.Configs.Retry; try >= 0; try-- {
		if response, err := requests.PostJson(requrl, reqdata); err == nil {
			v := make(map[string]interface{})
			if e := response.Json(&v); e == nil {
				if v["errcode"].(float64) == 0 {
					return nil
				} else {
					return errors.New(v["errmsg"].(string))
				}
			} else {
				return e
			}
		}
	}
	return ErrorStill
}

type NewsMsg struct {
	Title       string
	Description string
	Url         string
	Picurl      string
}

func (qwmsg *Qwmsg) SendNewsMsg(news []NewsMsg, safe bool) error {
	articles := make([]map[string]string, 0)
	for _, art := range news {
		temp := make(map[string]string)
		temp["title"] = art.Title
		temp["description"] = art.Description
		temp["url"] = art.Url
		temp["picurl"] = art.Picurl
		articles = append(articles, temp)
	}
	reqdata := mergeMaps(qwmsg.CommonField, map[string]interface{}{
		"msgtype": "news",
		"news": map[string]interface{}{
			"articles": articles,
		},
	})
	if safe {
		reqdata = mergeMaps(reqdata, map[string]interface{}{
			"safe": 1,
		})
	}
	requrl := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", qwmsg.AccessToken())
	for try := qwmsg.Configs.Retry; try >= 0; try-- {
		if response, err := requests.PostJson(requrl, reqdata); err == nil {
			v := make(map[string]interface{})
			if e := response.Json(&v); e == nil {
				if v["errcode"].(float64) == 0 {
					return nil
				} else {
					return errors.New(v["errmsg"].(string))
				}
			} else {
				return e
			}
		}
	}
	return ErrorStill
}

func (qwmsg *Qwmsg) SendMarkdownMsg(content string) error {
	reqdata := mergeMaps(qwmsg.CommonField, map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"content": content,
		},
	})
	requrl := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", qwmsg.AccessToken())
	for try := qwmsg.Configs.Retry; try >= 0; try-- {
		if response, err := requests.PostJson(requrl, reqdata); err == nil {
			v := make(map[string]interface{})
			if e := response.Json(&v); e == nil {
				if v["errcode"].(float64) == 0 {
					return nil
				} else {
					return errors.New(v["errmsg"].(string))
				}
			} else {
				return e
			}
		}
	}
	return ErrorStill
}
