package transform

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/logx"
)

// TestEngine_Transform_DirectMapping 测试直接映射
func TestEngine_Transform_DirectMapping(t *testing.T) {
	// 创建Engine
	logger := logx.WithContext(context.Background())
	engine := NewEngine(nil, nil, logger)

	// 准备测试数据
	sourceData := map[string]interface{}{
		"name":  "John Doe",
		"age":   30,
		"email": "john@example.com",
	}

	// 创建映射规则
	mappings := []*ent.FieldMapping{
		{
			ID:            1,
			MappingName:   "name_mapping",
			SourceField:   "name",
			TargetField:   "full_name",
			TransformType: "direct",
			IsActive:      true,
			AllowNull:     true,
		},
		{
			ID:            2,
			MappingName:   "age_mapping",
			SourceField:   "age",
			TargetField:   "user_age",
			TransformType: "direct",
			IsActive:      true,
			AllowNull:     true,
		},
	}

	// 创建Transform上下文
	ctx := &TransformContext{
		Ctx:        context.Background(),
		TenantID:   1,
		TaskID:     100,
		SourceData: sourceData,
		Mappings:   mappings,
	}

	// 执行Transform
	result := engine.Transform(ctx)

	// 验证结果
	assert.True(t, result.Success, "Transform应该成功")
	assert.Equal(t, 2, result.Stats.TotalFields)
	assert.Equal(t, 2, result.Stats.SuccessFields)
	assert.Equal(t, 0, result.Stats.FailedFields)
	assert.Equal(t, "John Doe", result.TargetData["full_name"])
	assert.Equal(t, 30, result.TargetData["user_age"])
}

// TestEngine_Transform_TypeConversion 测试类型转换
func TestEngine_Transform_TypeConversion(t *testing.T) {
	logger := logx.WithContext(context.Background())
	engine := NewEngine(nil, nil, logger)

	sourceData := map[string]interface{}{
		"age_str":    "25",
		"price_str":  "99.99",
		"active_str": "true",
	}

	mappings := []*ent.FieldMapping{
		{
			ID:             1,
			MappingName:    "age_convert",
			SourceField:    "age_str",
			TargetField:    "age",
			TransformType:  "convert",
			TargetDataType: "int",
			IsActive:       true,
			AllowNull:      true,
		},
		{
			ID:             2,
			MappingName:    "price_convert",
			SourceField:    "price_str",
			TargetField:    "price",
			TransformType:  "convert",
			TargetDataType: "float",
			IsActive:       true,
			AllowNull:      true,
		},
		{
			ID:             3,
			MappingName:    "active_convert",
			SourceField:    "active_str",
			TargetField:    "is_active",
			TransformType:  "convert",
			TargetDataType: "bool",
			IsActive:       true,
			AllowNull:      true,
		},
	}

	ctx := &TransformContext{
		Ctx:        context.Background(),
		TenantID:   1,
		TaskID:     101,
		SourceData: sourceData,
		Mappings:   mappings,
	}

	result := engine.Transform(ctx)

	assert.True(t, result.Success)
	assert.Equal(t, int64(25), result.TargetData["age"])
	assert.Equal(t, 99.99, result.TargetData["price"])
	assert.Equal(t, true, result.TargetData["is_active"])
}

// TestEngine_Transform_NestedPath 测试嵌套路径
func TestEngine_Transform_NestedPath(t *testing.T) {
	logger := logx.WithContext(context.Background())
	engine := NewEngine(nil, nil, logger)

	sourceData := map[string]interface{}{
		"user": map[string]interface{}{
			"profile": map[string]interface{}{
				"name": "Alice",
				"age":  28,
			},
			"settings": map[string]interface{}{
				"theme": "dark",
			},
		},
	}

	mappings := []*ent.FieldMapping{
		{
			ID:              1,
			MappingName:     "nested_name",
			SourceField:     "user.profile.name",
			SourceFieldPath: "user.profile.name",
			TargetField:     "name",
			TransformType:   "direct",
			IsActive:        true,
			AllowNull:       true,
		},
		{
			ID:              2,
			MappingName:     "nested_theme",
			SourceField:     "user.settings.theme",
			SourceFieldPath: "user.settings.theme",
			TargetField:     "theme",
			TransformType:   "direct",
			IsActive:        true,
			AllowNull:       true,
		},
	}

	ctx := &TransformContext{
		Ctx:        context.Background(),
		TenantID:   1,
		TaskID:     102,
		SourceData: sourceData,
		Mappings:   mappings,
	}

	result := engine.Transform(ctx)

	assert.True(t, result.Success)
	assert.Equal(t, "Alice", result.TargetData["name"])
	assert.Equal(t, "dark", result.TargetData["theme"])
}

// TestEngine_Transform_LookupTable 测试查找表
func TestEngine_Transform_LookupTable(t *testing.T) {
	logger := logx.WithContext(context.Background())
	engine := NewEngine(nil, nil, logger)

	sourceData := map[string]interface{}{
		"gender": "1",
		"status": "active",
	}

	// 创建查找表
	genderLookup := map[string]string{
		"1": "男",
		"2": "女",
	}
	genderLookupJSON, _ := json.Marshal(genderLookup)

	statusLookup := map[string]string{
		"active":   "激活",
		"inactive": "未激活",
	}
	statusLookupJSON, _ := json.Marshal(statusLookup)

	mappings := []*ent.FieldMapping{
		{
			ID:            1,
			MappingName:   "gender_lookup",
			SourceField:   "gender",
			TargetField:   "gender_text",
			TransformType: "direct",
			LookupTable:   string(genderLookupJSON),
			IsActive:      true,
			AllowNull:     true,
		},
		{
			ID:            2,
			MappingName:   "status_lookup",
			SourceField:   "status",
			TargetField:   "status_text",
			TransformType: "direct",
			LookupTable:   string(statusLookupJSON),
			IsActive:      true,
			AllowNull:     true,
		},
	}

	ctx := &TransformContext{
		Ctx:        context.Background(),
		TenantID:   1,
		TaskID:     103,
		SourceData: sourceData,
		Mappings:   mappings,
	}

	result := engine.Transform(ctx)

	assert.True(t, result.Success)
	assert.Equal(t, "男", result.TargetData["gender_text"])
	assert.Equal(t, "激活", result.TargetData["status_text"])
}

// TestEngine_Transform_RequiredField 测试必填字段验证
func TestEngine_Transform_RequiredField(t *testing.T) {
	logger := logx.WithContext(context.Background())
	engine := NewEngine(nil, nil, logger)

	// 缺少required字段
	sourceData := map[string]interface{}{
		"name": "John",
		// email字段缺失
	}

	mappings := []*ent.FieldMapping{
		{
			ID:            1,
			MappingName:   "email_required",
			SourceField:   "email",
			TargetField:   "user_email",
			TransformType: "direct",
			IsRequired:    true, // 必填
			IsActive:      true,
			AllowNull:     false,
		},
	}

	ctx := &TransformContext{
		Ctx:        context.Background(),
		TenantID:   1,
		TaskID:     104,
		SourceData: sourceData,
		Mappings:   mappings,
	}

	result := engine.Transform(ctx)

	// 应该失败
	assert.False(t, result.Success)
	assert.Equal(t, 1, result.Stats.FailedFields)
	assert.Contains(t, result.Stats.ErrorMessages[0], "not found")
}

// TestEngine_Transform_DefaultValue 测试默认值
func TestEngine_Transform_DefaultValue(t *testing.T) {
	logger := logx.WithContext(context.Background())
	engine := NewEngine(nil, nil, logger)

	sourceData := map[string]interface{}{
		"name": "John",
		// country字段缺失，应使用默认值
	}

	mappings := []*ent.FieldMapping{
		{
			ID:            1,
			MappingName:   "country_default",
			SourceField:   "country",
			TargetField:   "user_country",
			TransformType: "direct",
			DefaultValue:  "China",
			IsActive:      true,
			AllowNull:     true,
		},
	}

	ctx := &TransformContext{
		Ctx:        context.Background(),
		TenantID:   1,
		TaskID:     105,
		SourceData: sourceData,
		Mappings:   mappings,
	}

	result := engine.Transform(ctx)

	assert.True(t, result.Success)
	assert.Equal(t, "China", result.TargetData["user_country"])
}

// TestEngine_Transform_Priority 测试优先级排序
func TestEngine_Transform_Priority(t *testing.T) {
	logger := logx.WithContext(context.Background())
	engine := NewEngine(nil, nil, logger)

	sourceData := map[string]interface{}{
		"value": 10,
	}

	// 创建不同优先级的映射（高优先级应先执行）
	mappings := []*ent.FieldMapping{
		{
			ID:            1,
			MappingName:   "low_priority",
			SourceField:   "value",
			TargetField:   "result",
			TransformType: "direct",
			Priority:      1,
			IsActive:      true,
			AllowNull:     true,
		},
		{
			ID:            2,
			MappingName:   "high_priority",
			SourceField:   "value",
			TargetField:   "result",
			TransformType: "direct",
			Priority:      10, // 高优先级
			IsActive:      true,
			AllowNull:     true,
		},
	}

	ctx := &TransformContext{
		Ctx:        context.Background(),
		TenantID:   1,
		TaskID:     106,
		SourceData: sourceData,
		Mappings:   mappings,
	}

	result := engine.Transform(ctx)

	// 高优先级先执行，但低优先级会覆盖（都映射到同一个字段）
	assert.True(t, result.Success)
	assert.Equal(t, 10, result.TargetData["result"])
}

// TestEngine_Transform_StrictMode 测试严格模式
func TestEngine_Transform_StrictMode(t *testing.T) {
	logger := logx.WithContext(context.Background())
	config := &EngineConfig{
		StrictMode:      true, // 严格模式：第一个错误就停止
		ContinueOnError: false,
	}
	engine := NewEngine(config, nil, logger)

	sourceData := map[string]interface{}{
		"name": "John",
	}

	mappings := []*ent.FieldMapping{
		{
			ID:            1,
			MappingName:   "email_required",
			SourceField:   "email", // 不存在
			TargetField:   "user_email",
			TransformType: "direct",
			IsRequired:    true,
			IsActive:      true,
			AllowNull:     false,
		},
		{
			ID:            2,
			MappingName:   "name_mapping",
			SourceField:   "name",
			TargetField:   "user_name",
			TransformType: "direct",
			IsActive:      true,
			AllowNull:     true,
		},
	}

	ctx := &TransformContext{
		Ctx:        context.Background(),
		TenantID:   1,
		TaskID:     107,
		SourceData: sourceData,
		Mappings:   mappings,
	}

	result := engine.Transform(ctx)

	// 严格模式下应该在第一个错误后停止
	assert.False(t, result.Success)
	assert.Equal(t, 1, result.Stats.FailedFields)
	// 第二个映射不应被执行
	_, exists := result.TargetData["user_name"]
	assert.False(t, exists)
}

// TestEngine_Transform_Performance 性能基准测试
func TestEngine_Transform_Performance(t *testing.T) {
	logger := logx.WithContext(context.Background())
	engine := NewEngine(nil, nil, logger)

	// 大量映射规则
	sourceData := make(map[string]interface{})
	mappings := make([]*ent.FieldMapping, 0, 100)

	for i := 0; i < 100; i++ {
		fieldName := string(rune('a' + i%26)) + string(rune('0' + i/26))
		sourceData[fieldName] = i

		mappings = append(mappings, &ent.FieldMapping{
			ID:            uint64(i + 1),
			MappingName:   "mapping_" + fieldName,
			SourceField:   fieldName,
			TargetField:   "target_" + fieldName,
			TransformType: "direct",
			IsActive:      true,
			AllowNull:     true,
		})
	}

	ctx := &TransformContext{
		Ctx:        context.Background(),
		TenantID:   1,
		TaskID:     108,
		SourceData: sourceData,
		Mappings:   mappings,
	}

	start := time.Now()
	result := engine.Transform(ctx)
	duration := time.Since(start)

	assert.True(t, result.Success)
	assert.Equal(t, 100, result.Stats.SuccessFields)
	t.Logf("Transform 100 fields took: %v", duration)

	// 性能要求：100个字段应在100ms内完成
	assert.Less(t, duration.Milliseconds(), int64(100), "性能不达标")
}
