package transform

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/zeromicro/go-zero/core/logx"
)

// ScriptExecutor JavaScript 脚本执行器
// 职责:
// 1. 执行 JavaScript 转换脚本
// 2. 沙箱隔离（禁止网络、文件访问）
// 3. 超时控制
// 4. 提供内置函数库
type ScriptExecutor struct {
	logger  logx.Logger
	timeout time.Duration // 默认 5 秒
}

// ScriptExecutorConfig 脚本执行器配置
type ScriptExecutorConfig struct {
	Timeout          time.Duration // 执行超时时间
	MaxMemoryMB      int64         // 最大内存限制（MB）
	EnableBuiltins   bool          // 启用内置函数
	AllowedFunctions []string      // 允许的函数列表
}

// ScriptResult 脚本执行结果
type ScriptResult struct {
	Value        interface{}   // 返回值
	ExecutionMS  int64         // 执行时间（毫秒）
	Success      bool          // 是否成功
	ErrorMessage string        // 错误信息
}

// NewScriptExecutor 创建脚本执行器
func NewScriptExecutor(logger logx.Logger) *ScriptExecutor {
	return &ScriptExecutor{
		logger:  logger,
		timeout: 5 * time.Second, // 默认 5 秒超时
	}
}

// NewScriptExecutorWithConfig 使用配置创建脚本执行器
func NewScriptExecutorWithConfig(logger logx.Logger, config *ScriptExecutorConfig) *ScriptExecutor {
	executor := &ScriptExecutor{
		logger:  logger,
		timeout: config.Timeout,
	}

	if executor.timeout == 0 {
		executor.timeout = 5 * time.Second
	}

	return executor
}

// Execute 执行 JavaScript 脚本
// 参数:
//   - ctx: 上下文（用于超时控制）
//   - script: JavaScript 代码
//   - value: 当前字段值
//   - record: 完整记录（可选）
// 返回:
//   - 转换后的值
//   - 错误信息
func (e *ScriptExecutor) Execute(ctx context.Context, script string, value interface{}, record map[string]interface{}) (interface{}, error) {
	startTime := time.Now()

	// 创建超时上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// 创建结果通道
	resultChan := make(chan *ScriptResult, 1)
	errorChan := make(chan error, 1)

	// 在goroutine中执行脚本
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorChan <- fmt.Errorf("script panic: %v", r)
			}
		}()

		result, err := e.executeScript(script, value, record)
		if err != nil {
			errorChan <- err
			return
		}

		resultChan <- &ScriptResult{
			Value:       result,
			ExecutionMS: time.Since(startTime).Milliseconds(),
			Success:     true,
		}
	}()

	// 等待执行完成或超时
	select {
	case result := <-resultChan:
		e.logger.Debugw("Script executed successfully",
			logx.Field("execution_ms", result.ExecutionMS))
		return result.Value, nil

	case err := <-errorChan:
		e.logger.Errorw("Script execution failed",
			logx.Field("error", err.Error()))
		return nil, err

	case <-timeoutCtx.Done():
		e.logger.Errorw("Script execution timeout",
			logx.Field("timeout", e.timeout))
		return nil, fmt.Errorf("script execution timeout after %v", e.timeout)
	}
}

// executeScript 实际执行脚本（内部方法）
func (e *ScriptExecutor) executeScript(script string, value interface{}, record map[string]interface{}) (interface{}, error) {
	// 创建 JavaScript VM
	vm := goja.New()

	// 设置全局变量
	if err := vm.Set("value", value); err != nil {
		return nil, fmt.Errorf("failed to set value: %w", err)
	}

	// 确保 record 始终存在（即使为空对象）
	if record == nil {
		record = make(map[string]interface{})
	}
	if err := vm.Set("record", record); err != nil {
		return nil, fmt.Errorf("failed to set record: %w", err)
	}

	// 注入内置函数
	e.injectBuiltinFunctions(vm)

	// 包装脚本：如果脚本定义了 transform 函数，调用它
	wrappedScript := e.wrapScript(script)

	// 执行脚本
	result, err := vm.RunString(wrappedScript)
	if err != nil {
		return nil, fmt.Errorf("script execution error: %w", err)
	}

	// 转换结果为 Go 值
	return e.exportValue(result), nil
}

// wrapScript 包装脚本
func (e *ScriptExecutor) wrapScript(script string) string {
	// 检查脚本是否定义了 transform 函数
	if strings.Contains(script, "function transform") {
		// 如果定义了 transform 函数，调用它
		return script + "\ntransform(value, record);"
	}

	// 否则直接执行脚本（脚本应该返回结果）
	return script
}

// injectBuiltinFunctions 注入内置函数
func (e *ScriptExecutor) injectBuiltinFunctions(vm *goja.Runtime) {
	// 日志函数
	vm.Set("log", func(args ...interface{}) {
		e.logger.Infow("Script log", logx.Field("args", args))
	})

	// 字符串操作
	vm.Set("trim", func(s string) string {
		return strings.TrimSpace(s)
	})

	vm.Set("upper", func(s string) string {
		return strings.ToUpper(s)
	})

	vm.Set("lower", func(s string) string {
		return strings.ToLower(s)
	})

	vm.Set("replace", func(s, old, new string) string {
		return strings.ReplaceAll(s, old, new)
	})

	vm.Set("split", func(s, sep string) []string {
		return strings.Split(s, sep)
	})

	vm.Set("join", func(arr []string, sep string) string {
		return strings.Join(arr, sep)
	})

	// 数值操作
	vm.Set("parseInt", func(s string) int64 {
		var result int64
		fmt.Sscanf(s, "%d", &result)
		return result
	})

	vm.Set("parseFloat", func(s string) float64 {
		var result float64
		fmt.Sscanf(s, "%f", &result)
		return result
	})

	vm.Set("round", func(n float64) int64 {
		if n >= 0 {
			return int64(n + 0.5)
		}
		return int64(n - 0.5)
	})

	// JSON 操作
	vm.Set("parseJSON", func(s string) interface{} {
		var result interface{}
		if err := json.Unmarshal([]byte(s), &result); err != nil {
			e.logger.Errorw("Failed to parse JSON", logx.Field("error", err))
			return nil
		}
		return result
	})

	vm.Set("toJSON", func(v interface{}) string {
		bytes, err := json.Marshal(v)
		if err != nil {
			e.logger.Errorw("Failed to convert to JSON", logx.Field("error", err))
			return ""
		}
		return string(bytes)
	})

	// 日期时间操作
	vm.Set("now", func() int64 {
		return time.Now().Unix()
	})

	vm.Set("formatDate", func(timestamp int64, layout string) string {
		t := time.Unix(timestamp, 0)
		// 简化的日期格式化
		switch layout {
		case "YYYY-MM-DD":
			return t.Format("2006-01-02")
		case "YYYY-MM-DD HH:mm:ss":
			return t.Format("2006-01-02 15:04:05")
		default:
			return t.Format(time.RFC3339)
		}
	})

	// 类型检查
	vm.Set("isString", func(v interface{}) bool {
		_, ok := v.(string)
		return ok
	})

	vm.Set("isNumber", func(v interface{}) bool {
		switch v.(type) {
		case int, int32, int64, float32, float64:
			return true
		default:
			return false
		}
	})

	vm.Set("isArray", func(v interface{}) bool {
		_, ok := v.([]interface{})
		return ok
	})

	vm.Set("isObject", func(v interface{}) bool {
		_, ok := v.(map[string]interface{})
		return ok
	})

	// 数组操作
	vm.Set("arrayLength", func(arr []interface{}) int {
		return len(arr)
	})

	vm.Set("arrayJoin", func(arr []interface{}, sep string) string {
		parts := make([]string, len(arr))
		for i, v := range arr {
			parts[i] = fmt.Sprintf("%v", v)
		}
		return strings.Join(parts, sep)
	})
}

// exportValue 导出 Goja 值为 Go 值
func (e *ScriptExecutor) exportValue(value goja.Value) interface{} {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil
	}

	return value.Export()
}

// ExecuteWithConfig 使用配置执行脚本
func (e *ScriptExecutor) ExecuteWithConfig(ctx context.Context, script string, value interface{}, record map[string]interface{}, config *ScriptExecutorConfig) (interface{}, error) {
	// 临时修改超时时间
	originalTimeout := e.timeout
	if config != nil && config.Timeout > 0 {
		e.timeout = config.Timeout
	}
	defer func() {
		e.timeout = originalTimeout
	}()

	return e.Execute(ctx, script, value, record)
}

// ValidateScript 验证脚本语法
func (e *ScriptExecutor) ValidateScript(script string) error {
	// 尝试编译脚本
	_, err := goja.Compile("validation", script, false)
	if err != nil {
		return fmt.Errorf("script syntax error: %w", err)
	}

	return nil
}

// SetTimeout 设置超时时间
func (e *ScriptExecutor) SetTimeout(timeout time.Duration) {
	e.timeout = timeout
}

// GetTimeout 获取超时时间
func (e *ScriptExecutor) GetTimeout() time.Duration {
	return e.timeout
}
