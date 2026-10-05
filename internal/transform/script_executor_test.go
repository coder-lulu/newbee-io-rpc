package transform

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"
)

// ============ ScriptExecutor Basic Tests ============

func TestScriptExecutor_SimpleScript(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "value * 2"
	result, err := executor.Execute(context.Background(), script, 10, nil)

	require.NoError(t, err)
	assert.Equal(t, int64(20), result)
}

func TestScriptExecutor_StringConcatenation(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "value + '-suffix'"
	result, err := executor.Execute(context.Background(), script, "test", nil)

	require.NoError(t, err)
	assert.Equal(t, "test-suffix", result)
}

func TestScriptExecutor_TransformFunction(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := `
		function transform(value, record) {
			return value * 1024; // Convert to bytes
		}
	`

	result, err := executor.Execute(context.Background(), script, 10, nil)

	require.NoError(t, err)
	assert.Equal(t, int64(10*1024), result)
}

func TestScriptExecutor_AccessRecord(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := `
		function transform(value, record) {
			return value + ' ' + record.suffix;
		}
	`

	record := map[string]interface{}{
		"suffix": "GB",
	}

	result, err := executor.Execute(context.Background(), script, "10", record)

	require.NoError(t, err)
	assert.Equal(t, "10 GB", result)
}

// ============ ScriptExecutor Builtin Functions Tests ============

func TestScriptExecutor_BuiltinTrim(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "trim(value)"
	result, err := executor.Execute(context.Background(), script, "  test  ", nil)

	require.NoError(t, err)
	assert.Equal(t, "test", result)
}

func TestScriptExecutor_BuiltinUpper(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "upper(value)"
	result, err := executor.Execute(context.Background(), script, "test", nil)

	require.NoError(t, err)
	assert.Equal(t, "TEST", result)
}

func TestScriptExecutor_BuiltinLower(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "lower(value)"
	result, err := executor.Execute(context.Background(), script, "TEST", nil)

	require.NoError(t, err)
	assert.Equal(t, "test", result)
}

func TestScriptExecutor_BuiltinReplace(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "replace(value, 'old', 'new')"
	result, err := executor.Execute(context.Background(), script, "old_value_old", nil)

	require.NoError(t, err)
	assert.Equal(t, "new_value_new", result)
}

func TestScriptExecutor_BuiltinSplit(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "split(value, ',')"
	result, err := executor.Execute(context.Background(), script, "a,b,c", nil)

	require.NoError(t, err)
	// split 返回的可能是 []string 或 []interface{}，都要处理
	switch v := result.(type) {
	case []interface{}:
		assert.Len(t, v, 3)
	case []string:
		assert.Len(t, v, 3)
	default:
		t.Fatalf("unexpected type: %T", result)
	}
}

func TestScriptExecutor_BuiltinJoin(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "join(['a', 'b', 'c'], '-')"
	result, err := executor.Execute(context.Background(), script, nil, nil)

	require.NoError(t, err)
	assert.Equal(t, "a-b-c", result)
}

func TestScriptExecutor_BuiltinParseInt(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "parseInt(value)"
	result, err := executor.Execute(context.Background(), script, "123", nil)

	require.NoError(t, err)
	assert.Equal(t, int64(123), result)
}

func TestScriptExecutor_BuiltinParseFloat(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "parseFloat(value)"
	result, err := executor.Execute(context.Background(), script, "123.45", nil)

	require.NoError(t, err)
	assert.InDelta(t, 123.45, result, 0.01)
}

func TestScriptExecutor_BuiltinRound(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "round(value)"
	result, err := executor.Execute(context.Background(), script, 123.56, nil)

	require.NoError(t, err)
	assert.Equal(t, int64(124), result)
}

func TestScriptExecutor_BuiltinParseJSON(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := `parseJSON(value).name`
	jsonStr := `{"name":"test","value":123}`
	result, err := executor.Execute(context.Background(), script, jsonStr, nil)

	require.NoError(t, err)
	assert.Equal(t, "test", result)
}

func TestScriptExecutor_BuiltinToJSON(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := `toJSON({name: "test", value: 123})`
	result, err := executor.Execute(context.Background(), script, nil, nil)

	require.NoError(t, err)
	resultStr, ok := result.(string)
	require.True(t, ok)
	assert.Contains(t, resultStr, "test")
	assert.Contains(t, resultStr, "123")
}

// ============ ScriptExecutor Type Check Functions Tests ============

func TestScriptExecutor_IsString(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "isString(value)"
	result, err := executor.Execute(context.Background(), script, "test", nil)

	require.NoError(t, err)
	assert.Equal(t, true, result)
}

func TestScriptExecutor_IsNumber(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "isNumber(value)"
	result, err := executor.Execute(context.Background(), script, 123, nil)

	require.NoError(t, err)
	assert.Equal(t, true, result)
}

// ============ ScriptExecutor Complex Scenarios ============

func TestScriptExecutor_MemoryConversion(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	// Convert MB to GB
	script := `
		function transform(value, record) {
			var memoryMB = parseInt(value);
			return round(memoryMB / 1024);
		}
	`

	result, err := executor.Execute(context.Background(), script, "2048", nil)

	require.NoError(t, err)
	assert.Equal(t, int64(2), result)
}

func TestScriptExecutor_IPAddressExtraction(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	// Extract IP from string
	script := `
		function transform(value, record) {
			var parts = split(value, ' ');
			return parts[0];
		}
	`

	result, err := executor.Execute(context.Background(), script, "192.168.1.10 eth0", nil)

	require.NoError(t, err)
	assert.Equal(t, "192.168.1.10", result)
}

func TestScriptExecutor_ConditionalTransform(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := `
		function transform(value, record) {
			var cores = parseInt(value);
			if (cores >= 16) {
				return "high";
			} else if (cores >= 8) {
				return "medium";
			} else {
				return "low";
			}
		}
	`

	testCases := []struct {
		input    int
		expected string
	}{
		{32, "high"},
		{12, "medium"},
		{4, "low"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result, err := executor.Execute(context.Background(), script, tc.input, nil)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// ============ ScriptExecutor Error Handling Tests ============

func TestScriptExecutor_SyntaxError(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "value ++ +" // Invalid syntax
	_, err := executor.Execute(context.Background(), script, 10, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "script execution error")
}

func TestScriptExecutor_ReferenceError(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "nonexistent_variable * 2"
	_, err := executor.Execute(context.Background(), script, 10, nil)

	assert.Error(t, err)
}

func TestScriptExecutor_Timeout(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))
	executor.SetTimeout(100 * time.Millisecond)

	// Infinite loop
	script := "while(true) {}"
	_, err := executor.Execute(context.Background(), script, nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

// ============ ScriptExecutor Validation Tests ============

func TestScriptExecutor_ValidateScript_Valid(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := `
		function transform(value, record) {
			return value * 2;
		}
	`

	err := executor.ValidateScript(script)
	assert.NoError(t, err)
}

func TestScriptExecutor_ValidateScript_Invalid(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "function test( {" // Invalid syntax
	err := executor.ValidateScript(script)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "syntax error")
}

// ============ ScriptExecutor Configuration Tests ============

func TestScriptExecutor_WithConfig(t *testing.T) {
	config := &ScriptExecutorConfig{
		Timeout:        2 * time.Second,
		EnableBuiltins: true,
	}

	executor := NewScriptExecutorWithConfig(logx.WithContext(context.Background()), config)
	assert.Equal(t, 2*time.Second, executor.GetTimeout())

	script := "value * 2"
	result, err := executor.Execute(context.Background(), script, 10, nil)

	require.NoError(t, err)
	assert.Equal(t, int64(20), result)
}

func TestScriptExecutor_CustomTimeout(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	originalTimeout := executor.GetTimeout()
	assert.Equal(t, 5*time.Second, originalTimeout)

	executor.SetTimeout(10 * time.Second)
	assert.Equal(t, 10*time.Second, executor.GetTimeout())
}

// ============ ScriptExecutor Integration Tests ============

func TestScriptExecutor_CompleteWorkflow(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	// Scenario: Process server memory data
	script := `
		function transform(value, record) {
			// value: "8192 MB"
			// Expected output: "8 GB"

			var parts = split(trim(value), ' ');
			var memoryMB = parseInt(parts[0]);
			var memoryGB = round(memoryMB / 1024);

			return memoryGB + ' GB';
		}
	`

	result, err := executor.Execute(context.Background(), script, "8192 MB", nil)

	require.NoError(t, err)
	assert.Equal(t, "8 GB", result)

	t.Log("✅ Complete workflow test passed")
}

func TestScriptExecutor_MultipleExecutions(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	script := "value * 2"

	// Execute multiple times with different inputs
	inputs := []interface{}{10, 20, 30, 40, 50}
	expected := []int64{20, 40, 60, 80, 100}

	for i, input := range inputs {
		result, err := executor.Execute(context.Background(), script, input, nil)
		require.NoError(t, err)
		assert.Equal(t, expected[i], result)
	}

	t.Log("✅ Multiple executions test passed")
}

// ============ Performance Benchmarks ============

func BenchmarkScriptExecutor_SimpleScript(b *testing.B) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))
	script := "value * 2"
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.Execute(ctx, script, 10, nil)
	}
}

func BenchmarkScriptExecutor_ComplexScript(b *testing.B) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))
	script := `
		function transform(value, record) {
			var parts = split(trim(value), ' ');
			var memoryMB = parseInt(parts[0]);
			return round(memoryMB / 1024);
		}
	`
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.Execute(ctx, script, "8192 MB", nil)
	}
}

func BenchmarkScriptExecutor_BuiltinFunctions(b *testing.B) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))
	script := "upper(trim(replace(value, 'old', 'new')))"
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.Execute(ctx, script, "  old_value_old  ", nil)
	}
}

// ============ Script Examples for Documentation ============

// These are example scripts demonstrating common usage patterns

func TestScriptExecutor_Example_SimpleConversion(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	// Convert MB to GB
	script := `
		function transform(value, record) {
			return value / 1024;
		}
	`

	result, err := executor.Execute(context.Background(), script, 2048, nil)
	require.NoError(t, err)
	// JavaScript division may return int64 for whole numbers
	assert.Equal(t, int64(2), result)

	t.Log("✅ Simple conversion example passed")
}

func TestScriptExecutor_Example_StringManipulation(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	// Format hostname
	script := `
		function transform(value, record) {
			return upper(trim(value)) + '-' + record.env;
		}
	`

	record := map[string]interface{}{
		"env": "prod",
	}

	result, err := executor.Execute(context.Background(), script, "  server01  ", record)
	require.NoError(t, err)
	assert.Equal(t, "SERVER01-prod", result)

	t.Log("✅ String manipulation example passed")
}

func TestScriptExecutor_Example_ConditionalLogic(t *testing.T) {
	executor := NewScriptExecutor(logx.WithContext(context.Background()))

	// Classify server by CPU cores
	script := `
		function transform(value, record) {
			var cores = parseInt(value);

			if (cores >= 32) {
				return "ultra";
			} else if (cores >= 16) {
				return "high";
			} else if (cores >= 8) {
				return "medium";
			} else {
				return "low";
			}
		}
	`

	result, err := executor.Execute(context.Background(), script, "16", nil)
	require.NoError(t, err)
	assert.Equal(t, "high", result)

	t.Log("✅ Conditional logic example passed")
}
