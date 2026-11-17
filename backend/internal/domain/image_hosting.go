package domain

// 图床服务商标识。空字符串视为 "see"（兼容历史上只有 S.EE 时的旧配置文件）。
const (
	ImageHostingProviderSEE   = "see"
	ImageHostingProviderGitee = "gitee"
	ImageHostingProviderOSS   = "oss"
)

type ImageHostingSetting struct {
	Enabled bool   `json:"enabled"`
	APIKey  string `json:"apiKey"` // S.EE 的 API Key
	// Provider 图床服务商：see / gitee / oss；空值按 see 处理
	Provider string                    `json:"provider,omitempty"`
	Gitee    *ImageHostingGiteeSetting `json:"gitee,omitempty"`
	OSS      *ImageHostingOSSSetting    `json:"oss,omitempty"`
}

// ImageHostingGiteeSetting Gitee 图床配置（基于仓库 Contents API，单文件 ≤ 1MB）
type ImageHostingGiteeSetting struct {
	AccessToken   string `json:"accessToken"`   // 私人令牌（需 projects 权限）
	Owner         string `json:"owner"`         // 个人空间地址名，如 gitee.com/<owner>
	Repo          string `json:"repo"`          // 图床仓库名
	Branch        string `json:"branch"`        // 分支名，留空默认 main
	PathPrefix    string `json:"pathPrefix"`    // 仓库内目录前缀，如 images/，可留空
	PublicBaseURL string `json:"publicBaseUrl"` // 自定义访问基地址（如 CDN/Pages 域名），留空用 Gitee raw 地址
}

// ImageHostingOSSSetting 阿里云 OSS 图床配置（REST V1 签名直连，不依赖 SDK）
type ImageHostingOSSSetting struct {
	AccessKeyID     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
	Bucket          string `json:"bucket"`
	Endpoint        string `json:"endpoint"`     // 如 oss-cn-shanghai.aliyuncs.com
	PathPrefix      string `json:"pathPrefix"`  // 对象名前缀，如 blog/，可留空
	CustomDomain    string `json:"customDomain"` // 自定义访问域名（含或不含 https:// 皆可），留空用 <bucket>.<endpoint>
}

type ImageHostingFile struct {
	FileID    int    `json:"file_id"`
	Filename  string `json:"filename"`
	Storename string `json:"storename"`
	URL       string `json:"url"`
	Delete    string `json:"delete"`
	Hash      string `json:"hash"`
	Path      string `json:"path"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Size      int    `json:"size"`
	CreatedAt int    `json:"created_at"`
	Page      string `json:"page"`
}

type ImageHostingListResponse struct {
	Code    int                `json:"code"`
	Data    []ImageHostingFile `json:"data"`
	Message string             `json:"message"`
	Success bool               `json:"success"`
}

type ImageHostingUploadResponse struct {
	Code    int              `json:"code"`
	Data    ImageHostingFile `json:"data"`
	Message string           `json:"message"`
	Success bool             `json:"success"`
}

type ImageHostingDeleteResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

type ImageHostingRepository interface {
	GetSetting() (*ImageHostingSetting, error)
	SaveSetting(setting *ImageHostingSetting) error
}
