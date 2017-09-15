/**
 * Created by I. Navrotskyj on 29.08.17.
 */

package call

import (
	"bytes"
	"encoding/json"
	"github.com/tidwall/gjson"
	"github.com/webitel/acr/src/pkg/logger"
	"github.com/webitel/acr/src/pkg/models"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func HttpRequest(c *Call, args interface{}) error {
	var props map[string]interface{}
	var ok bool
	var uri string
	var err error
	var urlParam *url.URL
	var str, k, method string
	var v interface{}
	var body []byte
	var req *http.Request
	var res *http.Response
	headers := make(map[string]string)

	if props, ok = args.(map[string]interface{}); !ok {
		logger.Error("Call %s httpRequest bad arguments %s", c.Uuid, args)
		return nil
	}

	if uri = getStringValueFromMap("url", props, ""); uri == "" {
		logger.Error("Call %s httpRequest url is required", c.Uuid)
		return nil
	}

	urlParam, err = url.Parse(uri)
	if err != nil {
		logger.Error("Call %s httpRequest parse url error: %s", c.Uuid, err.Error())
		return nil
	}

	if _, ok = props["path"]; ok {
		if _, ok = props["path"].(models.Application); ok {
			for k, v = range props["path"].(models.Application) {
				str = parseMapValue(c, v)
				urlParam.Path = strings.Replace(urlParam.Path, "${"+k+"}", str, -1)
				urlParam.RawQuery = strings.Replace(urlParam.RawQuery, "${"+k+"}", str, -1)
			}
		}
	}

	if _, ok = props["headers"]; ok {
		if _, ok = props["headers"].(models.Application); ok {
			for k, v = range props["headers"].(models.Application) {
				headers[strings.ToLower(k)] = parseMapValue(c, v)
			}
		}
	}

	if _, ok = headers["content-type"]; !ok {
		headers["content-type"] = "application/json"
	}

	if _, ok = props["data"]; ok {
		switch strings.ToLower(headers["content-type"]) {
		case "application/x-www-form-urlencoded":
			str = ""
			switch props["data"].(type) {
			case models.Application:
				for k, v = range props["data"].(models.Application) {
					str += "&" + k + "=" + parseMapValue(c, v)
				}
				str = str[1:]
			case string:
				str = props["data"].(string)
			}

			if len(str) > 0 {
				body = []byte(strings.Replace(c.ParseString(str), " ", "+", -1))
			}

			//case "application/json":
		default:

			body, err = json.Marshal(props["data"])
			if err != nil {
				logger.Error("Call %s httpRequest marshal data error: %s", c.Uuid, err.Error())
			} else {
				body = []byte(c.ParseString(string(body)))
			}
		}

	}

	method = strings.ToUpper(getStringValueFromMap("method", props, "POST"))

	req, err = http.NewRequest(method, urlParam.String(), bytes.NewBuffer(body))
	if err != nil {
		logger.Error("Call %s httpRequest create request error: %s", c.Uuid, err.Error())
		return nil
	}

	for k, str = range headers {
		req.Header.Set(k, str)
	}

	client := &http.Client{
		Timeout: time.Duration(getIntValueFromMap("timeout", props, 1000)) * time.Millisecond,
	}
	res, err = client.Do(req)
	if err != nil {
		logger.Error("Call %s httpRequest response error: %s", c.Uuid, err.Error())
		return nil
	}
	defer res.Body.Close()

	if str = getStringValueFromMap("responseCode", props, ""); str != "" {
		SetVar(c, str+"="+strconv.Itoa(res.StatusCode))
	}

	if str = getStringValueFromMap("exportCookie", props, ""); str != "" {
		if _, ok = res.Header["Set-Cookie"]; ok {
			err = SetVar(c, str+"="+strings.Join(res.Header["Set-Cookie"], ";"))
			if err != nil {
				logger.Error("Call %s httpRequest set cookie variable error: %s", c.Uuid, err.Error())
			}
		}
	}

	if res.ContentLength == 0 {
		logger.Debug("Call %s httpRequest response from %s code %v no response", c.Uuid, urlParam.String(), res.StatusCode)
		return nil
	} else {
		logger.Debug("Call %s httpRequest response from %s code %v content length %v", c.Uuid, urlParam.String(), res.StatusCode, res.ContentLength)
	}

	str = res.Header.Get("content-type")
	if strings.Index(str, "application/json") > -1 {
		if _, ok = props["exportVariables"]; ok {
			if _, ok = props["exportVariables"].(models.Application); ok {
				body, err = ioutil.ReadAll(res.Body)
				if err != nil {
					logger.Error("Call %s httpRequest read response error: %s", c.Uuid, err.Error())
					return nil
				}
				for k, v = range props["exportVariables"].(models.Application) {
					if str, ok = v.(string); ok {
						err = SetVar(c, "all:"+k+"="+gjson.GetBytes(body, str).String())
						if err != nil {
							logger.Error("Call %s httpRequest setVat error: %s", c.Uuid, err.Error())
						}
					}
				}
			}
		}
	} else {
		logger.Warning("Call %s httpRequest no support parse content-type %s", c.Uuid, str)
	}

	return nil
}

func parseMapValue(c *Call, v interface{}) (str string) {
	str = parseInterfaceToString(v)
	if strings.HasPrefix(str, "${") && strings.HasSuffix(str, "}") {
		str = c.GetChannelVar(str[2 : len(str)-1])
	}
	return str
}