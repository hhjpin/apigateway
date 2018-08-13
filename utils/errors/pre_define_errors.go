package errors

type BaseErrMsg struct {
	ErrMsg   string `json:"err_msg"`
	ErrMsgEn string `json:"err_msg_en"`
}

var (
	BaseErrors = map[int]BaseErrMsg{}
)

func init() {
	BaseErrors[0] = BaseErrMsg{ErrMsg: "成功", ErrMsgEn: "success"}
	BaseErrors[1] = BaseErrMsg{ErrMsg: "未知错误", ErrMsgEn: "unknown error"}
	BaseErrors[2] = BaseErrMsg{ErrMsg: "网络波动, 请重新尝试", ErrMsgEn: "unstable network connection, please retry"}
	BaseErrors[3] = BaseErrMsg{ErrMsg: "权限不足", ErrMsgEn: "permission denied"}
	BaseErrors[4] = BaseErrMsg{ErrMsg: "服务维护中, 请稍后", ErrMsgEn: "service maintaining, please wait"}
	BaseErrors[5] = BaseErrMsg{ErrMsg: "访问量过大, 请稍后重试", ErrMsgEn: "router under flow control"}
	BaseErrors[6] = BaseErrMsg{ErrMsg: "请求的服务不存在", ErrMsgEn: "service not found"}
	BaseErrors[7] = BaseErrMsg{ErrMsg: "路由无效", ErrMsgEn: "routing table not exists"}

	// router operating error
	BaseErrors[20] = BaseErrMsg{ErrMsg: "服务已存在", ErrMsgEn: "service already exists"}
	BaseErrors[21] = BaseErrMsg{ErrMsg: "服务不存在", ErrMsgEn: "service not exists"}
	BaseErrors[22] = BaseErrMsg{ErrMsg: "终端服务已存在", ErrMsgEn: "endpoint already exists"}
	BaseErrors[23] = BaseErrMsg{ErrMsg: "终端服务不存在", ErrMsgEn: "endpoint not exists"}

	BaseErrors[24] = BaseErrMsg{ErrMsg: "路由规则使用中, 不可移除外层API", ErrMsgEn: "router is online, can not remove frontend api"}
	BaseErrors[25] = BaseErrMsg{ErrMsg: "路由规则不存在", ErrMsgEn: "router not exist"}
	BaseErrors[26] = BaseErrMsg{ErrMsg: "缺少外层API定义, 无法创建路由规则", ErrMsgEn: "api obj not completed"}
	BaseErrors[27] = BaseErrMsg{ErrMsg: "缺少服务定义, 无法创建路由规则", ErrMsgEn: "service obj not completed"}
	BaseErrors[28] = BaseErrMsg{ErrMsg: "路由规则已存在", ErrMsgEn: "router already exists"}
	BaseErrors[29] = BaseErrMsg{ErrMsg: "绑定终端服务过程中出现错误, 无法创建路由规则", ErrMsgEn: "error raised when add endpoint to endpoint-table"}
	BaseErrors[30] = BaseErrMsg{ErrMsg: "绑定的终端服务无一在线, 无法创建路由规则", ErrMsgEn: "no server online, can not create router"}
	BaseErrors[31] = BaseErrMsg{ErrMsg: "无法找到路由规则", ErrMsgEn: "can not find router by name"}

	BaseErrors[32] = BaseErrMsg{ErrMsg: "路由规则在线, 无法被注销", ErrMsgEn: "router is online, can not unregister it"}
	BaseErrors[33] = BaseErrMsg{ErrMsg: "绑定的终端服务无一在线, 路由无法被设置为在线", ErrMsgEn: "all server are not online, this router should not be set to online"}
	BaseErrors[34] = BaseErrMsg{ErrMsg: "绑定终端服务过程中出现错误, 路由无法被设置为在线", ErrMsgEn: "error raised when add server to server-table"}
	BaseErrors[35] = BaseErrMsg{ErrMsg: "路由规则与在线路由表中的规则不符", ErrMsgEn: "data mapping error"}
	BaseErrors[36] = BaseErrMsg{ErrMsg: "路由规则已在线", ErrMsgEn: "router already online"}
	BaseErrors[37] = BaseErrMsg{ErrMsg: "无法找到服务", ErrMsgEn: "can not find service by name"}
	BaseErrors[38] = BaseErrMsg{ErrMsg: "服务链接到某个已存在的路由规则, 无法被移除", ErrMsgEn: "service is linked to an online router, can not be removed"}
	BaseErrors[39] = BaseErrMsg{ErrMsg: "无法找到终端服务", ErrMsgEn: "can not find endpoint by name"}
	BaseErrors[40] = BaseErrMsg{ErrMsg: "轮询终端服务队列时发生异常: 节点数据错误", ErrMsgEn: "the Ring of endpoints contains an invalid value"}
	BaseErrors[41] = BaseErrMsg{ErrMsg: "轮询终端服务队列时发生异常: 无一节点在线", ErrMsgEn: "all node of rings is not online"}
	BaseErrors[42] = BaseErrMsg{ErrMsg: "外层API不存在", ErrMsgEn: "frontend api not found"}
	BaseErrors[43] = BaseErrMsg{ErrMsg: "路由规则不在线", ErrMsgEn: "router is not online"}
	BaseErrors[44] = BaseErrMsg{ErrMsg: "服务对应的终端节点无一在线", ErrMsgEn: "all node of rings is not online"}

	BaseErrors[50] = BaseErrMsg{ErrMsg: "不正确的状态数值", ErrMsgEn: "invalid status code"}
	BaseErrors[51] = BaseErrMsg{ErrMsg: "终端节点属性值不全", ErrMsgEn: "uncompleted attribute assigned"}

	BaseErrors[60] = BaseErrMsg{ErrMsg: "缺少健康检查配置", ErrMsgEn: "health check not defined"}
	BaseErrors[61] = BaseErrMsg{ErrMsg: "健康检查失败", ErrMsgEn: "health check failed"}

	BaseErrors[70] = BaseErrMsg{ErrMsg: "无法生成Token令牌", ErrMsgEn: "can not generate token bucket"}
}
