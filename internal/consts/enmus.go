package consts

// ValueType 数据类型枚举
const (
	ValueTypeInt       = "int"       // 整数
	ValueTypeFloat     = "float"     // 浮点数
	ValueTypeShortText = "text"      // 短文本（文本长度 <= 128）
	ValueTypeLongText  = "longtext"  // 长文本（文本长度 > 128）
	ValueTypeDateTime  = "datetime"  // 日期时间（yyyy-MM-dd HH:mm:ss）
	ValueTypeDate      = "date"      // 日期（yyyy-MM-dd）
	ValueTypeTime      = "time"      // 时间（HH:mm:ss）
	ValueTypeJSON      = "json"      // JSON
	ValueTypePassword  = "password"  // 密码
	ValueTypeLink      = "link"      // 链接
	ValueTypeReference = "reference" // 引用
	ValueTypeBool      = "boolean"   // 布尔
	ValueTypeImage     = "image"     // 图片
)

// ValueTypeLabelMap 用于前端展示的label和描述
var ValueTypeLabelMap = map[string]struct {
	Label       string
	Description string
}{
	ValueTypeInt:       {Label: "整数", Description: "整数"},
	ValueTypeFloat:     {Label: "浮点数", Description: "浮点数"},
	ValueTypeShortText: {Label: "短文本", Description: "文本长度 <= 128"},
	ValueTypeLongText:  {Label: "长文本", Description: "文本长度 > 128"},
	ValueTypeDateTime:  {Label: "日期时间", Description: "yyyy-MM-dd HH:mm:ss"},
	ValueTypeDate:      {Label: "日期", Description: "yyyy-MM-dd"},
	ValueTypeTime:      {Label: "时间", Description: "HH:mm:ss"},
	ValueTypeJSON:      {Label: "JSON", Description: "JSON"},
	ValueTypePassword:  {Label: "密码", Description: "密码"},
	ValueTypeLink:      {Label: "链接", Description: "链接"},
	ValueTypeReference: {Label: "引用", Description: "引用"},
	ValueTypeBool:      {Label: "布尔", Description: "布尔"},
	ValueTypeImage:     {Label: "图片", Description: "图片"},
}
