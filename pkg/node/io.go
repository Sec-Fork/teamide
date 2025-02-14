package node

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/team-ide/go-tool/util"
	"go.uber.org/zap"
	"io"
	"net"
	"sync"
	"teamide/pkg/filework"
	"teamide/pkg/system"
	"teamide/pkg/terminal"
)

var (
	LengthError     = errors.New("读取流长度错误")
	ConnClosedError = errors.New("连接已关闭")
)

type Message struct {
	Id                 string            `json:"id,omitempty"`
	Method             MethodType        `json:"method,omitempty"`
	Error              string            `json:"error,omitempty"`
	NotifiedNodeIdList []string          `json:"notifiedNodeIdList,omitempty"`
	LineNodeIdList     []string          `json:"lineNodeIdList,omitempty"`
	ConnData           *ConnData         `json:"connData,omitempty"`
	NodeWorkData       *WorkData         `json:"nodeWorkData,omitempty"`
	NetProxyWorkData   *NetProxyWorkData `json:"netProxyWorkData,omitempty"`
	FileWorkData       *FileWorkData     `json:"fileWorkData,omitempty"`
	TerminalWorkData   *TerminalWorkData `json:"terminalWorkData,omitempty"`
	SystemData         *SystemData       `json:"systemData,omitempty"`
	HasBytes           bool              `json:"hasBytes,omitempty"`
	SendKey            string            `json:"sendKey,omitempty"`
	Bytes              []byte            `json:"-"`
	listener           *MessageListener
}

type ConnData struct {
	ConnIndex  int      `json:"connIndex,omitempty"`
	NodeId     string   `json:"nodeId,omitempty"`
	NodeToken  string   `json:"nodeToken,omitempty"`
	NodeIdList []string `json:"nodeIdList,omitempty"`
}

type SystemData struct {
	NodeId        string                `json:"nodeId,omitempty"`
	QueryRequest  *system.QueryRequest  `json:"queryRequest,omitempty"`
	QueryResponse *system.QueryResponse `json:"queryResponse,omitempty"`
	Info          *system.Info          `json:"info,omitempty"`
}

type WorkData struct {
	NodeId       string    `json:"nodeId,omitempty"`
	ToNodeList   []*ToNode `json:"toNodeList,omitempty"`
	ToNodeIdList []string  `json:"toNodeIdList,omitempty"`

	Version     string       `json:"version,omitempty"`
	MonitorData *MonitorData `json:"monitorData,omitempty"`
	Status      int8         `json:"status,omitempty"`
}

type NetProxyWorkData struct {
	NetProxyId        string           `json:"netProxyId,omitempty"`
	ConnId            string           `json:"connId,omitempty"`
	IsReverse         bool             `json:"isReverse,omitempty"`
	MonitorData       *MonitorData     `json:"monitorData,omitempty"`
	NetProxyInnerList []*NetProxyInner `json:"netProxyInnerList,omitempty"`
	NetProxyOuterList []*NetProxyOuter `json:"netProxyOuterList,omitempty"`
	NetProxyIdList    []string         `json:"netProxyIdList,omitempty"`
	Status            int8             `json:"status,omitempty"`
}

type FileWorkData struct {
	File        *filework.FileInfo   `json:"file,omitempty"`
	FileList    []*filework.FileInfo `json:"fileList,omitempty"`
	Dir         string               `json:"dir,omitempty"`
	Path        string               `json:"path,omitempty"`
	OldPath     string               `json:"oldPath,omitempty"`
	NewPath     string               `json:"newPath,omitempty"`
	IsDir       bool                 `json:"isDir,omitempty"`
	Exist       bool                 `json:"exist,omitempty"`
	FileCount   int                  `json:"fileCount,omitempty"`
	RemoveCount int                  `json:"removeCount,omitempty"`
}

type TerminalWorkData struct {
	Key       string         `json:"key,omitempty"`
	ReadKey   string         `json:"readKey,omitempty"`
	Size      *terminal.Size `json:"size,omitempty"`
	IsWindows bool           `json:"isWindows,omitempty"`
}

type StatusChange struct {
	Id          string `json:"id,omitempty"`
	Status      int8   `json:"status,omitempty"`
	StatusError string `json:"statusError,omitempty"`
}

func (this_ *Message) ReturnError(error string, MonitorData *MonitorData) (err error) {
	err = this_.Return(&Message{
		Error: error,
	}, MonitorData)
	if err != nil {
		return
	}

	return
}

func (this_ *Message) Return(msg *Message, MonitorData *MonitorData) (err error) {
	if this_.listener == nil {
		err = errors.New("消息监听器丢失")
		return
	}
	msg.Id = this_.Id
	err = this_.listener.Send(msg, MonitorData)
	if err != nil {
		return
	}
	return
}

type MessageListener struct {
	conn      net.Conn
	onMessage func(msg *Message)
	isClose   bool
	isStop    bool
	writeMu   sync.Mutex
}

func (this_ *MessageListener) stop() {
	this_.isStop = true
	_ = this_.conn.Close()
}

func (this_ *MessageListener) listen(onClose func(), MonitorData *MonitorData) {
	var err error
	this_.isClose = false
	go func() {
		defer func() {
			this_.isClose = true
			if x := recover(); x != nil {
				Logger.Error("message listen error", zap.Error(err))
			}
			_ = this_.conn.Close()
			onClose()
		}()

		for {
			if this_.isStop {
				return
			}
			var msg *Message
			msg, err = ReadMessage(this_.conn, MonitorData)
			if err != nil {
				if this_.isStop {
					return
				}
				if err == io.EOF {
					return
				}
				return
			}
			msg.listener = this_
			go this_.onMessage(msg)
		}
	}()
}

func (this_ *MessageListener) Send(msg *Message, MonitorData *MonitorData) (err error) {
	if msg == nil {
		return
	}
	if this_.isClose {
		err = ConnClosedError
		return
	}
	this_.writeMu.Lock()
	defer this_.writeMu.Unlock()
	err = WriteMessage(this_.conn, msg, MonitorData)
	return
}

func ReadMessage(reader io.Reader, MonitorData *MonitorData) (message *Message, err error) {
	var bytes []byte

	bytes, err = ReadBytes(reader, MonitorData)
	if err != nil {
		return
	}
	message = &Message{}

	err = json.Unmarshal(bytes, &message)
	if err != nil {
		Logger.Error("ReadMessage JSON Unmarshal error", zap.Any("bytes length", len(bytes)), zap.Error(err))
		return
	}
	if message.HasBytes {
		bytes, err = ReadBytes(reader, MonitorData)
		if err != nil {
			return
		}
		message.Bytes = bytes
	}
	return
}

func WriteMessage(writer io.Writer, message *Message, MonitorData *MonitorData) (err error) {
	var bytes []byte

	bytes, err = json.Marshal(message)
	if err != nil {
		return
	}

	err = WriteBytes(writer, bytes, MonitorData)
	if err != nil {
		return
	}
	if message.HasBytes {
		err = WriteBytes(writer, message.Bytes, MonitorData)
	}
	return
}

func ReadBytes(reader io.Reader, MonitorData *MonitorData) (bytes []byte, err error) {

	start := util.GetNow().UnixNano()

	var buf []byte
	var n int

	buf = make([]byte, 4)
	n, err = reader.Read(buf)
	if err != nil {
		return
	}
	if n < 4 {
		err = LengthError
		return
	}

	length := int(binary.LittleEndian.Uint32(buf))
	if length < 0 {
		err = LengthError
		return
	}

	if length > 0 {
		var hasLen = length
		for {

			bs := make([]byte, hasLen)
			n, err = reader.Read(bs)
			if err != nil && err != io.EOF {
				return
			}
			hasLen = hasLen - n
			bytes = append(bytes, bs[0:n]...)
			//Logger.Info("ReadBytes", zap.Any("should read", length), zap.Any("has size", hasLen))
			if hasLen <= 0 {
				break
			}
			if err == io.EOF {
				break
			}
		}
	}
	end := util.GetNow().UnixNano()
	MonitorData.monitorWrite(int64(length+4), end-start)
	return
}

func WriteBytes(writer io.Writer, bytes []byte, MonitorData *MonitorData) (err error) {
	var n int
	var length = len(bytes)

	start := util.GetNow().UnixNano()

	writeBytes := []byte{0, 0, 0, 0}
	binary.LittleEndian.PutUint32(writeBytes, uint32(length))

	n, err = writer.Write(writeBytes)
	if err != nil {
		return
	}
	if n < 4 {
		err = LengthError
		return
	}

	n, err = writer.Write(bytes)
	if err != nil {
		return
	}
	//Logger.Info("WriteBytes", zap.Any("should write", length), zap.Any("write size", n))
	if n < length {
		err = LengthError
		return
	}
	end := util.GetNow().UnixNano()
	MonitorData.monitorWrite(int64(length), end-start)
	return
}
