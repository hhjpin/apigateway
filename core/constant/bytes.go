package constant

var (
	SlashBytes             = []byte("/")
	NodeKeyBytes           = []byte("Node")
	ServiceKeyBytes        = []byte("Service")
	ServicePrefixBytes     = []byte("Service-")
	RouterPrefixBytes      = []byte("Router-")
	HealthCheckKeyBytes    = []byte("HealthCheck")
	IdKeyBytes             = []byte("ID")
	NameKeyBytes           = []byte("Name")
	HostKeyBytes           = []byte("Host")
	PortKeyBytes           = []byte("Port")
	StatusKeyBytes         = []byte("Status")
	FrontendApiKeyBytes    = []byte("FrontendApi")
	BackendApiKeyBytes     = []byte("BackendApi")
	PathKeyBytes           = []byte("Path")
	TimeoutKeyBytes        = []byte("Timeout")
	IntervalKeyBytes       = []byte("Interval")
	RetryKeyBytes          = []byte("Retry")
	RetryTimeKeyBytes      = []byte("RetryTime")
	RouterDefinitionBytes  = []byte("/Router/")
	ServiceDefinitionBytes = []byte("/Service/")

	//StrSlash            = []byte("/")
	//StrSlashSlash       = []byte("//")
	//StrSlashDotDot      = []byte("/..")
	//StrSlashDotSlash    = []byte("/./")
	//StrSlashDotDotSlash = []byte("/../")
	//StrCRLF             = []byte("\r\n")
	//StrHTTP             = []byte("http")
	//StrHTTPS            = []byte("https")
	//StrHTTP11           = []byte("HTTP/1.1")
	//StrColonSlashSlash  = []byte("://")
	//StrColonSpace       = []byte(": ")
	//StrGMT              = []byte("GMT")

	//StrResponseContinue = []byte("HTTP/1.1 100 Continue\r\n\r\n")

	StrGet    = []byte("GET")
	//StrHead   = []byte("HEAD")
	//StrPost   = []byte("POST")
	//StrPut    = []byte("PUT")
	//StrDelete = []byte("DELETE")

	//StrExpect           = []byte("Expect")
	//StrConnection       = []byte("Connection")
	//StrContentLength    = []byte("Content-Length")
	//StrContentType      = []byte("Content-Type")
	//StrDate             = []byte("Date")
	StrHost             = []byte("Host")
	//StrReferer          = []byte("Referer")
	//StrServer           = []byte("Server")
	//StrTransferEncoding = []byte("Transfer-Encoding")
	//StrContentEncoding  = []byte("Content-Encoding")
	//StrAcceptEncoding   = []byte("Accept-Encoding")
	//StrUserAgent        = []byte("User-Agent")
	//StrCookie           = []byte("Cookie")
	//StrSetCookie        = []byte("Set-Cookie")
	//StrLocation         = []byte("Location")
	//StrIfModifiedSince  = []byte("If-Modified-Since")
	//StrLastModified     = []byte("Last-Modified")
	//StrAcceptRanges     = []byte("Accept-Ranges")
	//StrRange            = []byte("Range")
	//StrContentRange     = []byte("Content-Range")

	//StrCookieExpires  = []byte("expires")
	//StrCookieDomain   = []byte("domain")
	//StrCookiePath     = []byte("path")
	//StrCookieHTTPOnly = []byte("HttpOnly")
	//StrCookieSecure   = []byte("secure")

	//StrClose               = []byte("close")
	//StrGzip                = []byte("gzip")
	//StrDeflate             = []byte("deflate")
	//StrKeepAlive           = []byte("keep-alive")
	//StrKeepAliveCamelCase  = []byte("Keep-Alive")
	//StrUpgrade             = []byte("Upgrade")
	//StrChunked             = []byte("chunked")
	//StrIdentity            = []byte("identity")
	//Str100Continue         = []byte("100-continue")
	//StrPostArgsContentType = []byte("application/x-www-form-urlencoded")
	//StrMultipartFormData   = []byte("multipart/form-data")
	//StrBoundary            = []byte("boundary")
	//StrBytes               = []byte("bytes")
	//StrTextSlash           = []byte("text/")
	//StrApplicationSlash    = []byte("application/")
	StrApplicationJson     = []byte("application/json")
)