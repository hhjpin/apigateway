package errors

type BaseErrMsg struct {
	ErrMsg   string `json:"err_msg"`
	ErrMsgEn string `json:"err_msg_en"`
}

type ErrCode int

var (
	baseErrors = map[ErrCode]BaseErrMsg{}
)

const (
	ErrSuccess ErrCode = iota
	ErrUnknownError
	ErrUnstableNetwork
	ErrPermissionDeny
	ErrServiceUnderMaintaining
	ErrTooMuchRequest
	ErrServiceNotFound

	ErrNeedLogin    ErrCode = 40100
	ErrTokenExpired ErrCode = 41101
)

func init() {
	// err code definition rule:
	// 	0				System Reserved: Success
	//  1 - 99			System Reserved: Basic Error
	//  100 - 299		System Reserved: Api Gateway
	//  300 - 999  		System Reserved: Reserved

	//  1000 - 9999  	Business Logic
	//
	//  	1000 - 1999		Module: User
	//  	2000 - 2999		Module: Post
	//  	3000 - 3999 	Module: Demand/Quote
	//  	4000 - 4999     Module: HomePage
	//  	5000 - 5999     Module: DropShipping
	//  	6000 - 6999 	Module: WxHtml5
	//  	7000 - 7999     Module: WxMiniProgram
	//		8000 - 9999 	Reserved

	//  41101 / 40100   System Reserved: Need Login

	baseErrors[0] = BaseErrMsg{ErrMsg: "成功", ErrMsgEn: "success"}
	baseErrors[1] = BaseErrMsg{ErrMsg: "未知错误", ErrMsgEn: "unknown error"}
	baseErrors[2] = BaseErrMsg{ErrMsg: "网络波动, 请重新尝试", ErrMsgEn: "unstable network connection, please retry"}
	baseErrors[3] = BaseErrMsg{ErrMsg: "权限不足", ErrMsgEn: "permission denied"}
	baseErrors[4] = BaseErrMsg{ErrMsg: "服务维护中, 请稍后", ErrMsgEn: "service maintaining, please wait"}
	baseErrors[5] = BaseErrMsg{ErrMsg: "访问量过大, 请稍后重试", ErrMsgEn: "router under flow control"}
	baseErrors[6] = BaseErrMsg{ErrMsg: "请求的服务不存在", ErrMsgEn: "service not found"}

	// router operating error
	baseErrors[100] = BaseErrMsg{ErrMsg: "路由无效", ErrMsgEn: "routing table not exists"}
	baseErrors[120] = BaseErrMsg{ErrMsg: "服务已存在", ErrMsgEn: "service already exists"}
	baseErrors[121] = BaseErrMsg{ErrMsg: "服务不存在", ErrMsgEn: "service not exists"}
	baseErrors[122] = BaseErrMsg{ErrMsg: "终端服务已存在", ErrMsgEn: "endpoint already exists"}
	baseErrors[123] = BaseErrMsg{ErrMsg: "终端服务不存在", ErrMsgEn: "endpoint not exists"}

	baseErrors[124] = BaseErrMsg{ErrMsg: "路由规则使用中, 不可移除外层API", ErrMsgEn: "router is online, can not remove frontend api"}
	baseErrors[125] = BaseErrMsg{ErrMsg: "路由规则不存在", ErrMsgEn: "router not exist"}
	baseErrors[126] = BaseErrMsg{ErrMsg: "缺少外层API定义, 无法创建路由规则", ErrMsgEn: "api obj not completed"}
	baseErrors[127] = BaseErrMsg{ErrMsg: "缺少服务定义, 无法创建路由规则", ErrMsgEn: "service obj not completed"}
	baseErrors[128] = BaseErrMsg{ErrMsg: "路由规则已存在", ErrMsgEn: "router already exists"}
	baseErrors[129] = BaseErrMsg{ErrMsg: "绑定终端服务过程中出现错误, 无法创建路由规则", ErrMsgEn: "error raised when add endpoint to endpoint-table"}
	baseErrors[130] = BaseErrMsg{ErrMsg: "绑定的终端服务无一在线, 无法创建路由规则", ErrMsgEn: "no server online, can not create router"}
	baseErrors[131] = BaseErrMsg{ErrMsg: "无法找到路由规则", ErrMsgEn: "can not find router by name"}

	baseErrors[132] = BaseErrMsg{ErrMsg: "路由规则在线, 无法被注销", ErrMsgEn: "router is online, can not unregister it"}
	baseErrors[133] = BaseErrMsg{ErrMsg: "绑定的终端服务无一在线, 路由无法被设置为在线", ErrMsgEn: "all server are not online, this router should not be set to online"}
	baseErrors[134] = BaseErrMsg{ErrMsg: "绑定终端服务过程中出现错误, 路由无法被设置为在线", ErrMsgEn: "error raised when add server to server-table"}
	baseErrors[135] = BaseErrMsg{ErrMsg: "路由规则与在线路由表中的规则不符", ErrMsgEn: "data mapping error"}
	baseErrors[136] = BaseErrMsg{ErrMsg: "路由规则已在线", ErrMsgEn: "router already online"}
	baseErrors[137] = BaseErrMsg{ErrMsg: "无法找到服务", ErrMsgEn: "can not find service by name"}
	baseErrors[138] = BaseErrMsg{ErrMsg: "服务链接到某个已存在的路由规则, 无法被移除", ErrMsgEn: "service is linked to an online router, can not be removed"}
	baseErrors[139] = BaseErrMsg{ErrMsg: "无法找到终端服务", ErrMsgEn: "can not find endpoint by name"}
	baseErrors[140] = BaseErrMsg{ErrMsg: "轮询终端服务队列时发生异常: 节点数据错误", ErrMsgEn: "the Ring of endpoints contains an invalid value"}
	baseErrors[141] = BaseErrMsg{ErrMsg: "轮询终端服务队列时发生异常: 无一节点在线", ErrMsgEn: "all node of rings is not online"}
	baseErrors[142] = BaseErrMsg{ErrMsg: "外层API不存在", ErrMsgEn: "frontend api not found"}
	baseErrors[143] = BaseErrMsg{ErrMsg: "路由规则不在线", ErrMsgEn: "router is not online"}
	baseErrors[144] = BaseErrMsg{ErrMsg: "服务对应的终端节点无一在线", ErrMsgEn: "all node of rings is not online"}

	baseErrors[150] = BaseErrMsg{ErrMsg: "不正确的状态数值", ErrMsgEn: "invalid status code"}
	baseErrors[151] = BaseErrMsg{ErrMsg: "终端节点属性值不全", ErrMsgEn: "uncompleted attribute assigned"}

	baseErrors[160] = BaseErrMsg{ErrMsg: "缺少健康检查配置", ErrMsgEn: "health check not defined"}
	baseErrors[161] = BaseErrMsg{ErrMsg: "健康检查失败", ErrMsgEn: "health check failed"}

	baseErrors[170] = BaseErrMsg{ErrMsg: "无法生成Token令牌", ErrMsgEn: "can not generate token bucket"}

	baseErrors[40100] = BaseErrMsg{ErrMsg: "需要重新登录", ErrMsgEn: "need login"}
	baseErrors[41101] = BaseErrMsg{ErrMsg: "需要重新登录", ErrMsgEn: "need login"}
}
