package base

import "github.com/gin-gonic/gin"

type RequestBean struct {
	JWT  *JWTBean
	Path string
}

type PageBean struct {
	PageIndex int64
	PageSize  int64
	Total     int64
	TotalPage int64
	Value     interface{}
}

func (page *PageBean) Init() {
	page.TotalPage = (page.Total + page.PageSize - 1) / page.PageSize
}

type JWTBean struct {
	Sign   string `json:"sign,omitempty"`
	UserId int64  `json:"userId,omitempty"`
	Name   string `json:"name,omitempty"`
	Time   int64  `json:"time,omitempty"`
}

type ApiWorker struct {
	Apis    []string
	Power   *PowerAction
	Do      func(request *RequestBean, c *gin.Context) (res interface{}, err error)
	DoOther func(request *RequestBean, c *gin.Context)
}

type PowerAction struct {
	Action      string `json:"action,omitempty"`
	Text        string `json:"text,omitempty"`
	ShouldLogin bool   `json:"shouldLogin,omitempty"`
	AllowNative bool   `json:"allowNative,omitempty"`
	Parent      *PowerAction
}

var (
	powers []*PowerAction

	// 基础权限
	PowerRegister  = addPower(&PowerAction{Action: "register", Text: "注册", AllowNative: false})
	PowerData      = addPower(&PowerAction{Action: "data", Text: "数据", AllowNative: true})
	PowerSession   = addPower(&PowerAction{Action: "session", Text: "会话", AllowNative: true})
	PowerLogin     = addPower(&PowerAction{Action: "login", Text: "登录", AllowNative: false})
	PowerLogout    = addPower(&PowerAction{Action: "logout", Text: "登出", AllowNative: false})
	PowerAutoLogin = addPower(&PowerAction{Action: "auto_login", Text: "自动登录", AllowNative: false})
)

func addPower(power *PowerAction) *PowerAction {
	powers = append(powers, power)
	return power
}

func GetPowers() (ps []*PowerAction) {

	ps = append(ps, powers...)

	return
}
