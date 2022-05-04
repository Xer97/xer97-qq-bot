package bot

import (
	"encoding/json"
	"example/xer97-qq-bot/bill"
	"example/xer97-qq-bot/netutil"
	"fmt"
	"golang.org/x/net/websocket"
	"log"
	"time"
)

type Author struct {
	Avatar   string `json:"avatar"`
	Bot      bool   `json:"bot"`
	Id       string `json:"id"`
	Username string `json:"username"`
}

type Data struct {
	Id                string `json:"id"`
	Author            Author `json:"author"`
	ChannelId         string `json:"channel_id"`
	Content           string `json:"content"`
	GuildId           string `json:"guild_id"`
	Seq               int64  `json:"seq"`
	HeartbeatInterval int64  `json:"heartbeat_interval"`
	SessionId         string `json:"session_id"`
}

type Payload struct {
	Operate int64  `json:"op"`
	Seq     int64  `json:"s"`
	Type    string `json:"t"`
	Data    Data   `json:"d"`
}

const (
	origin      = "http://localhost"
	api         = "https://sandbox.api.sgroup.qq.com"
	pathGateway = "/gateway"
	pathSendMsg = "/channels/%v/messages"
	appId       = "102003833"
	token       = "5erUjRcZRiESnn08mlrAb9SKXJxIu7KL"

	// 事件标记
	ecGuilds              = 1 << 0
	ecGuildsMembers       = 1 << 1
	ecPublicGuildMessages = 1 << 30
	ecs                   = ecGuilds | ecGuildsMembers | ecPublicGuildMessages

	// 事件名称
	enReady           = "READY"
	enGuildCreate     = "GUILD_CREATE"
	enAtMessageCreate = "AT_MESSAGE_CREATE"
	enGuildMemberAdd  = "GUILD_MEMBER_ADD"

	// 操作枚举
	opDispatch        = 0
	opHeartbeat       = 1
	opIdentify        = 2
	opResume          = 6
	opReconnect       = 7
	opInvalidSession  = 9
	opHello           = 10
	opHeartbeatACK    = 11
	opHTTPCallbackACK = 12
)

var heartbeatInterval int64
var seq int64
var sessionId string

var ws *websocket.Conn

func Start() {
	// ws连接
	connWs()
	// 启动心跳
	go heartbeat()
	// 启动ws监听
	go listen()
	// 认证
	auth()
	// 阻塞
	select {}
}

// getUrl 拼接http请求url
func getUrl(api string, path string) string {
	return api + path
}

// getToken 拼接访问token
func getToken() string {
	return "Bot " + appId + "." + token
}

// getWsUrl 获取ws地址
func getWsUrl() string {
	header := make(map[string]string)
	header["authorization"] = getToken()

	ret := netutil.GetReq(getUrl(api, pathGateway), header)

	var result map[string]string
	err := json.Unmarshal(ret, &result)
	if err != nil {
		log.Fatalf("getWsUrl json.Unmarshal error. %v", err)
	}

	return result["url"]
}

func connWs() {
	// 初始化连接信息
	url := getWsUrl()
	config, err := websocket.NewConfig(url, origin)

	// 执行ws连接
	ws, err = websocket.DialConfig(config)
	if err != nil {
		log.Fatalf("ws conn error. %v", err)
	}

	// 获取连接结果
	var result Payload
	if err = websocket.JSON.Receive(ws, &result); err != nil {
		log.Fatalf("conn receive error. %v", err)
	}

	msg, _ := json.Marshal(result)
	log.Printf("conn Received: %v.\n", string(msg))
	// 保存心跳间隔信息
	heartbeatInterval = result.Data.HeartbeatInterval
}

// reConn 重新连接
func reConn() {
	connWs()
	auth()
}

// auth 鉴权
func auth() {
	payload := make(map[string]interface{})
	data := make(map[string]interface{})
	data["token"] = getToken()
	data["intents"] = ecs
	if sessionId != "" && seq != 0 {
		data["session_id"] = sessionId
		data["seq"] = seq
	}
	payload["op"] = 2
	payload["d"] = data
	log.Println(payload)

	if err := websocket.JSON.Send(ws, &payload); err != nil {
		log.Fatalf("auth send error. %v", err)
	}
}

// heartbeat 维持心跳
func heartbeat() {
	ticker := time.NewTicker(time.Duration(heartbeatInterval) * time.Millisecond)
	defer ticker.Stop()

	data := make(map[string]int64)
	data["op"] = 1
	for range ticker.C {
		data["d"] = seq
		log.Printf("ticker ticker ticker ... send heartbeat:[%v]\n", data)

		if err := websocket.JSON.Send(ws, data); err != nil {
			log.Fatalf("heartbeat send error. %v", err)
		}
	}
}

// listen websocket监听
func listen() {
	for true {
		var payload Payload
		if err := websocket.JSON.Receive(ws, &payload); err != nil {
			log.Printf("listen error. %v", err)
			// 重新连接
			reConn()
		}
		msg, _ := json.Marshal(payload)
		log.Printf("event Received: %v.\n", string(msg))

		opSelect(payload)
	}
}

// opSelect 操作类型分发
func opSelect(payload Payload) {
	switch payload.Operate {
	case opDispatch:
		// 记录消息序列号，心跳用
		seq = payload.Seq
		eventSelect(payload)
		break
	case opReconnect:
		log.Println("重新连接")
		reConn()
		break
	case opHeartbeatACK:
		log.Println("接收到心跳响应")
		break
	default:
		break
	}
}

// eventSelect 事件分发
func eventSelect(payload Payload) {
	switch payload.Type {
	case enReady:
		// 鉴权成功
		sessionId = payload.Data.SessionId
		break
	case enGuildCreate:
		break
	case enAtMessageCreate:
		// 有@信息，进入记账功能
		content := bill.Enter(payload.Data.Author.Id, payload.Data.Content)
		reply(payload.Data, content)
		break
	default:
		break
	}
}

// reply 回复信息
func reply(data Data, content string) {
	body := make(map[string]string)
	body["content"] = fmt.Sprintf("<@!%v>\n", data.Author.Id) + content
	body["msg_id"] = data.Id
	msg, _ := json.Marshal(body)

	header := make(map[string]string)
	header["authorization"] = getToken()
	header["Content-Type"] = "application/json; charset=utf-8"

	resp := netutil.PostReq(getUrl(api, fmt.Sprintf(pathSendMsg, data.ChannelId)), msg, header)
	log.Println(string(resp))
}
