//go:build mini
// +build mini

package SunnyNet

import (
	"io"

	"github.com/qtgolang/SunnyNet/src/http"
)

// 是否是用户自定义脚本编辑请求
func (s *proxyRequest) isUserScriptCodeEditRequest(request *http.Request) bool {
	return false
}

// SetScriptCode 设置脚本代码
func (s *Sunny) SetScriptCode(code string) string {
	return "no"
}

// SetScriptPage 设置脚本页面
func (s *Sunny) SetScriptPage(Page string) string {
	return "no"
}

func allowOrigin(t string, w io.Writer) {
	w.Write([]byte(t + "\r\n"))
	w.Write([]byte("Access-Control-Allow-Origin: *\r\n"))
	w.Write([]byte("Access-Control-Allow-Methods: *\r\n"))
	w.Write([]byte("Access-Control-Allow-Headers: *\r\n"))
	w.Write([]byte("Access-Control-Expose-Headers: *\r\n"))
}
