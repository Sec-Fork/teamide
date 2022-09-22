package module_terminal

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"
	"teamide/internal/base"
	"teamide/internal/module/module_node"
	"teamide/internal/module/module_toolbox"
	"teamide/pkg/terminal"
	"teamide/pkg/util"
)

type api struct {
	*worker
}

func NewApi(toolboxService_ *module_toolbox.ToolboxService, nodeService_ *module_node.NodeService) *api {
	return &api{
		worker: NewWorker(toolboxService_, nodeService_),
	}
}

var (
	// Terminal 权限

	// Power 文件管理器 基本 权限
	Power                = base.AppendPower(&base.PowerAction{Action: "terminal", Text: "工具", ShouldLogin: false, StandAlone: true})
	PowerWebsocket       = base.AppendPower(&base.PowerAction{Action: "terminal_websocket", Text: "工具", ShouldLogin: true, StandAlone: true})
	PowerClose           = base.AppendPower(&base.PowerAction{Action: "terminal_close", Text: "工具", ShouldLogin: true, StandAlone: true})
	PowerKet             = base.AppendPower(&base.PowerAction{Action: "terminal_key", Text: "工具", ShouldLogin: true, StandAlone: true})
	PowerChangeSize      = base.AppendPower(&base.PowerAction{Action: "terminal_change_size", Text: "工具", ShouldLogin: true, StandAlone: true})
	PowerUploadWebsocket = base.AppendPower(&base.PowerAction{Action: "terminal_upload_websocket", Text: "工具", ShouldLogin: true, StandAlone: true})
)

func (this_ *api) GetApis() (apis []*base.ApiWorker) {
	apis = append(apis, &base.ApiWorker{Apis: []string{"terminal"}, Power: Power, Do: this_.index})
	apis = append(apis, &base.ApiWorker{Apis: []string{"terminal/key"}, Power: PowerKet, Do: this_.key})
	apis = append(apis, &base.ApiWorker{Apis: []string{"terminal/websocket"}, Power: PowerWebsocket, Do: this_.websocket, IsWebSocket: true})
	apis = append(apis, &base.ApiWorker{Apis: []string{"terminal/changeSize"}, Power: PowerChangeSize, Do: this_.changeSize})
	apis = append(apis, &base.ApiWorker{Apis: []string{"terminal/close"}, Power: PowerClose, Do: this_.close})
	apis = append(apis, &base.ApiWorker{Apis: []string{"terminal/uploadWebsocket"}, Power: PowerUploadWebsocket, Do: this_.uploadWebsocket, IsWebSocket: true})

	return
}

func (this_ *api) index(_ *base.RequestBean, _ *gin.Context) (res interface{}, err error) {
	return
}

func (this_ *api) key(_ *base.RequestBean, c *gin.Context) (res interface{}, err error) {
	request := &Request{}
	if !base.RequestJSON(request, c) {
		return
	}

	service, err := this_.createService(request.Place, request.PlaceId)
	if err != nil {
		return
	}

	data := make(map[string]interface{})

	data["isWindows"], err = service.IsWindows()
	if err != nil {
		return
	}
	data["key"] = util.UUID()
	res = data
	return
}

var upGrader = websocket.Upgrader{
	ReadBufferSize:  32 * 1024,
	WriteBufferSize: 32 * 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (this_ *api) websocket(request *base.RequestBean, c *gin.Context) (res interface{}, err error) {

	if request.JWT == nil || request.JWT.UserId == 0 {
		err = errors.New("登录用户获取失败")
		return
	}
	key := c.Query("key")
	if key == "" {
		err = errors.New("key获取失败")
		return
	}
	place := c.Query("place")
	if place == "" {
		err = errors.New("place获取失败")
		return
	}
	placeId := c.Query("placeId")
	if placeId == "" {
		err = errors.New("placeId获取失败")
		return
	}
	cols, _ := strconv.Atoi(c.Query("cols"))
	rows, _ := strconv.Atoi(c.Query("rows"))
	//升级get请求为webSocket协议
	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	service := this_.GetService(key)
	if service != nil {
		err = errors.New("会话[" + key + "]已存在")

		_ = ws.WriteMessage(websocket.BinaryMessage, []byte("service create error:"+err.Error()))
		this_.Logger.Error("websocket start error", zap.Error(err))
		_ = ws.Close()
		return
	}

	err = this_.Start(key, place, placeId, &terminal.Size{
		Cols: cols,
		Rows: rows,
	}, ws)
	if err != nil {
		_ = ws.WriteMessage(websocket.BinaryMessage, []byte("start error:"+err.Error()))
		this_.Logger.Error("websocket start error", zap.Error(err))
		_ = ws.Close()
		return
	}

	res = base.HttpNotResponse
	return
}

func (this_ *api) uploadWebsocket(request *base.RequestBean, c *gin.Context) (res interface{}, err error) {

	if request.JWT == nil || request.JWT.UserId == 0 {
		err = errors.New("登录用户获取失败")
		return
	}
	key := c.Query("key")
	if key == "" {
		err = errors.New("key获取失败")
		return
	}
	//升级get请求为webSocket协议
	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	service := this_.GetService(key)
	if service == nil {
		err = errors.New("会话[" + key + "]不存在")

		this_.Logger.Error("uploadWebsocket start error", zap.Error(err))
		_ = ws.Close()
		return
	}

	go func() {
		defer func() {
			if e := recover(); e != nil {
				this_.Logger.Error("uploadWebsocket error", zap.Any("error", e))
			}
		}()

		var buf []byte
		var readErr error
		var writeErr error
		for {
			_, buf, readErr = ws.ReadMessage()
			if readErr != nil && readErr != io.EOF {
				break
			}
			//this_.Logger.Info("ws on read", zap.Any("bs", string(buf)))
			_, writeErr = service.Write(buf)
			if writeErr != nil {
				break
			}
			writeErr = ws.WriteMessage(websocket.BinaryMessage, []byte{0})
			if writeErr != nil {
				break
			}
			if readErr == io.EOF {
				readErr = nil
				break
			}
		}

		if readErr != nil {
			this_.Logger.Error("uploadWebsocket read error", zap.Error(readErr))
		}

		if writeErr != nil {
			this_.Logger.Error("uploadWebsocket write error", zap.Error(writeErr))
		}
	}()

	res = base.HttpNotResponse
	return
}

type Request struct {
	Place   string `json:"place,omitempty"`
	PlaceId string `json:"placeId,omitempty"`
	Key     string `json:"key,omitempty"`
	*terminal.Size
}

func (this_ *api) close(_ *base.RequestBean, c *gin.Context) (res interface{}, err error) {
	request := &Request{}
	if !base.RequestJSON(request, c) {
		return
	}
	this_.stopService(request.Key)
	return
}

func (this_ *api) changeSize(_ *base.RequestBean, c *gin.Context) (res interface{}, err error) {
	request := &Request{}
	if !base.RequestJSON(request, c) {
		return
	}
	service := this_.GetService(request.Key)
	if service != nil {
		err = service.ChangeSize(request.Size)
	}
	return
}
