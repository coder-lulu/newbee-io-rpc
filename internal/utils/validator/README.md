# 通用数据验证器

基于 [go-playground/validator](https://github.com/go-playground/validator) 和 go-zero 框架实现的通用数据验证工具，支持结构体验证、单个字段验证、自定义验证规则和国际化错误消息。

## 特性

- 基于 go-playground/validator/v10 实现
- 与 go-zero 框架无缝集成
- 支持自定义验证规则
- 友好的错误提示（支持中文错误消息）
- 可扩展的设计，易于添加新的验证规则
- 提供 HTTP 和 gRPC 验证中间件
- 使用 go-zero 的 errorx 统一错误处理

## 安装依赖

```bash
go get github.com/go-playground/validator/v10
```

## 基本用法

### 结构体验证

```go
type User struct {
    ID        string `json:"id" validate:"required,uuid"`
    Name      string `json:"name" validate:"required,min=2,max=50"`
    Age       int    `json:"age" validate:"required,gte=0,lte=120"`
    Email     string `json:"email" validate:"required,email"`
    Mobile    string `json:"mobile" validate:"required,mobile"`
}

func CreateUser(user User) error {
    // 验证整个结构体
    if err := validator.Validate(user); err != nil {
        return err
    }
    
    // 验证通过，继续处理...
    return nil
}
```

### 单个字段验证

```go
func ValidateEmail(email string) error {
    // 验证单个字段
    if err := validator.ValidateVar(email, "required,email"); err != nil {
        return err
    }
    
    // 验证通过，继续处理...
    return nil
}
```

### 在Go-Zero的RPC服务中使用

```go
func main() {
    // ... 初始化代码 ...
    
    s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
        // 注册服务
        pb.RegisterUserServer(grpcServer, server.NewUserServer(ctx))
    })
    
    // 添加验证器拦截器
    s.AddUnaryInterceptors(validator.UnaryServerInterceptor())
    
    // ... 启动服务 ...
}
```

### 在Go-Zero的API服务中使用

```go
// 在API Handler中使用
func (h *UserHandler) CreateUser(r *http.Request) (resp *types.Response, err error) {
    var req types.CreateUserReq
    
    // 解析并验证请求
    if err := validator.HTTPValidator(r, &req); err != nil {
        return nil, err
    }
    
    // 验证通过，继续处理...
    return &types.Response{
        Code: 200,
        Message: "创建成功",
    }, nil
}
```

## 自定义验证规则

```go
// 注册自定义验证规则
func init() {
    // 注册一个"even"验证器，验证数字是否为偶数
    _ = validator.RegisterValidation("even", func(fl validator.FieldLevel) bool {
        return fl.Field().Int() % 2 == 0
    }, "必须是偶数")
}

// 使用自定义验证规则
type Product struct {
    ID      string `json:"id" validate:"required,uuid"`
    Count   int    `json:"count" validate:"required,gte=0,even"`
}
```

## 内置自定义验证器

| 验证标签 | 说明 |
|---------|------|
| mobile | 手机号验证（中国大陆手机号格式） |
| uuid | UUID格式验证 |
| alphanumdash | 字母数字下划线验证 |

## 错误处理

验证器返回的错误是 `errorx.InvalidArgumentError` 类型，可以直接返回给客户端：

```go
func (l *SomeLogic) SomeMethod(req *pb.Request) (*pb.Response, error) {
    if err := validator.Validate(req); err != nil {
        // 错误已经是 errorx.InvalidArgumentError 类型，可以直接返回
        return nil, err
    }
    
    // 验证通过，继续处理...
}
``` 