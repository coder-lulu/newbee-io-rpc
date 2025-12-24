package provider

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

type FileImportProvider struct{}

func NewFileImportProvider() *FileImportProvider {
	return &FileImportProvider{}
}

func (p *FileImportProvider) GetMetadata() *ProviderMetadata {
	return &ProviderMetadata{
		ID:          "file_import",
		Name:        "文件导入",
		Description: "支持Excel、CSV、JSON文件导入",
		Version:     "1.0.0",
		Type:        "file",
		IconURL:     "/static/icons/file-import.svg",
		SupportedModes: []string{
			"excel",
			"csv",
			"json",
		},
		Tags: []string{"file", "import", "data"},
	}
}

func (p *FileImportProvider) GetParameterSchema() []ParameterDefinition {
	return []ParameterDefinition{
		{
			Name:        "file_path",
			DisplayName: "文件路径",
			Type:        "string",
			Required:    true,
			Description: "要导入的文件路径",
			Example:     "/path/to/data.xlsx",
		},
		{
			Name:        "file_type",
			DisplayName: "文件类型",
			Type:        "select",
			Required:    false,
			Description: "文件类型，留空则自动检测",
			DefaultValue: "auto",
			Options: []SelectOption{
				{Value: "auto", Label: "自动检测", Description: "根据文件扩展名自动检测"},
				{Value: "excel", Label: "Excel文件", Description: ".xlsx, .xls文件"},
				{Value: "csv", Label: "CSV文件", Description: ".csv文件"},
				{Value: "json", Label: "JSON文件", Description: ".json文件"},
			},
		},
		{
			Name:        "sheet_name",
			DisplayName: "工作表名称",
			Type:        "string",
			Required:    false,
			Description: "Excel工作表名称，留空则使用第一个工作表",
			Example:     "Sheet1",
		},
		{
			Name:        "header_row",
			DisplayName: "标题行",
			Type:        "integer",
			Required:    false,
			Description: "标题行位置，从1开始",
			DefaultValue: 1,
			Example:     "1",
		},
		{
			Name:        "encoding",
			DisplayName: "文件编码",
			Type:        "select",
			Required:    false,
			Description: "文件编码格式",
			DefaultValue: "utf-8",
			Options: []SelectOption{
				{Value: "utf-8", Label: "UTF-8", Description: "Unicode编码"},
				{Value: "gbk", Label: "GBK", Description: "中文编码"},
				{Value: "gb2312", Label: "GB2312", Description: "简体中文编码"},
			},
		},
	}
}

func (p *FileImportProvider) GetFieldSchema() []FieldDefinition {
	return []FieldDefinition{
		{
			Name:        "row_number",
			DisplayName: "行号",
			Type:        "integer",
			Description: "数据行号",
			Required:    false,
		},
		{
			Name:        "data",
			DisplayName: "数据内容",
			Type:        "json",
			Description: "解析后的数据内容",
			Required:    true,
		},
		{
			Name:        "source_file",
			DisplayName: "源文件",
			Type:        "string",
			Description: "源文件路径",
			Required:    false,
		},
		{
			Name:        "file_type",
			DisplayName: "文件类型",
			Type:        "string",
			Description: "文件类型",
			Required:    false,
		},
	}
}

func (p *FileImportProvider) ValidateConfig(config map[string]interface{}) error {
	filePath, ok := config["file_path"].(string)
	if !ok || filePath == "" {
		return fmt.Errorf("file_path is required")
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	return nil
}

func (p *FileImportProvider) TestConnection(config map[string]interface{}) (*TestResult, error) {
	filePath, ok := config["file_path"].(string)
	if !ok {
		return &TestResult{
			Success: false,
			Message: "配置错误",
			Error:   "file_path参数类型错误或缺失",
		}, fmt.Errorf("file_path must be a string")
	}
	
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("无法访问文件: %v", err),
		}, nil
	}

	fileType := p.detectFileType(filePath, config)
	
	switch fileType {
	case "excel":
		f, err := excelize.OpenFile(filePath)
		if err != nil {
			return &TestResult{
				Success: false,
				Message: fmt.Sprintf("无法打开Excel文件: %v", err),
			}, nil
		}
		f.Close()
	case "csv":
		file, err := os.Open(filePath)
		if err != nil {
			return &TestResult{
				Success: false,
				Message: fmt.Sprintf("无法打开CSV文件: %v", err),
			}, nil
		}
		file.Close()
	case "json":
		file, err := os.Open(filePath)
		if err != nil {
			return &TestResult{
				Success: false,
				Message: fmt.Sprintf("无法打开JSON文件: %v", err),
			}, nil
		}
		
		var testData interface{}
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&testData); err != nil {
			file.Close()
			return &TestResult{
				Success: false,
				Message: fmt.Sprintf("JSON格式错误: %v", err),
			}, nil
		}
		file.Close()
	default:
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("不支持的文件类型: %s", fileType),
		}, nil
	}

	return &TestResult{
		Success: true,
		Message: "文件访问成功",
		Details: map[string]interface{}{
			"file_path": filePath,
			"file_type": fileType,
			"file_size": fileInfo.Size(),
		},
	}, nil
}

func (p *FileImportProvider) Discover(config map[string]interface{}) (*DiscoveryResult, error) {
	filePath, ok := config["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path must be a string")
	}
	fileType := p.detectFileType(filePath, config)
	
	var records []map[string]interface{}
	var err error

	switch fileType {
	case "excel":
		records, err = p.parseExcel(filePath, config)
	case "csv":
		records, err = p.parseCSV(filePath, config)
	case "json":
		records, err = p.parseJSON(filePath, config)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}

	if err != nil {
		return nil, err
	}

	return &DiscoveryResult{
		Success:      true,
		TotalRecords: int64(len(records)),
		Records:      records,
		Metadata: map[string]interface{}{
			"file_path": filePath,
			"file_type": fileType,
		},
	}, nil
}

func (p *FileImportProvider) detectFileType(filePath string, config map[string]interface{}) string {
	if fileType, exists := config["file_type"]; exists {
		if fileTypeStr, ok := fileType.(string); ok && fileTypeStr != "auto" {
			return fileTypeStr
		}
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".xlsx", ".xls":
		return "excel"
	case ".csv":
		return "csv"
	case ".json":
		return "json"
	default:
		return "unknown"
	}
}

func (p *FileImportProvider) parseExcel(filePath string, config map[string]interface{}) ([]map[string]interface{}, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %v", err)
	}
	defer f.Close()

	sheetName := ""
	if sheet, exists := config["sheet_name"]; exists {
		if sheetStr, ok := sheet.(string); ok {
			sheetName = sheetStr
		}
	}

	if sheetName == "" {
		sheetList := f.GetSheetList()
		if len(sheetList) == 0 {
			return nil, fmt.Errorf("no sheets found in Excel file")
		}
		sheetName = sheetList[0]
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet: %v", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("sheet is empty")
	}

	headerRow := 1
	if hr, exists := config["header_row"]; exists {
		if hrFloat, ok := hr.(float64); ok {
			headerRow = int(hrFloat)
		}
	}

	if headerRow > len(rows) {
		return nil, fmt.Errorf("header row %d exceeds total rows %d", headerRow, len(rows))
	}

	headers := rows[headerRow-1]
	var records []map[string]interface{}

	for i, row := range rows[headerRow:] {
		record := map[string]interface{}{
			"row_number":  i + headerRow + 1,
			"source_file": filePath,
			"file_type":   "excel",
		}

		data := make(map[string]interface{})
		for j, cell := range row {
			header := fmt.Sprintf("column_%d", j+1)
			if j < len(headers) && headers[j] != "" {
				header = headers[j]
			}
			data[header] = cell
		}
		record["data"] = data

		records = append(records, record)
	}

	return records, nil
}

func (p *FileImportProvider) parseCSV(filePath string, config map[string]interface{}) ([]map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %v", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	headerRow := 1
	if hr, exists := config["header_row"]; exists {
		if hrFloat, ok := hr.(float64); ok {
			headerRow = int(hrFloat)
		}
	}

	if headerRow > len(rows) {
		return nil, fmt.Errorf("header row %d exceeds total rows %d", headerRow, len(rows))
	}

	headers := rows[headerRow-1]
	var records []map[string]interface{}

	for i, row := range rows[headerRow:] {
		record := map[string]interface{}{
			"row_number":  i + headerRow + 1,
			"source_file": filePath,
			"file_type":   "csv",
		}

		data := make(map[string]interface{})
		for j, cell := range row {
			header := fmt.Sprintf("column_%d", j+1)
			if j < len(headers) && headers[j] != "" {
				header = headers[j]
			}
			data[header] = cell
		}
		record["data"] = data

		records = append(records, record)
	}

	return records, nil
}

func (p *FileImportProvider) parseJSON(filePath string, config map[string]interface{}) ([]map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open JSON file: %v", err)
	}
	defer file.Close()

	var jsonData interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	var records []map[string]interface{}

	switch data := jsonData.(type) {
	case []interface{}:
		for i, item := range data {
			record := map[string]interface{}{
				"row_number":  i + 1,
				"source_file": filePath,
				"file_type":   "json",
				"data":        item,
			}
			records = append(records, record)
		}
	case map[string]interface{}:
		record := map[string]interface{}{
			"row_number":  1,
			"source_file": filePath,
			"file_type":   "json",
			"data":        data,
		}
		records = append(records, record)
	default:
		record := map[string]interface{}{
			"row_number":  1,
			"source_file": filePath,
			"file_type":   "json",
			"data":        data,
		}
		records = append(records, record)
	}

	return records, nil
}

func (p *FileImportProvider) GetFieldMapping(targetSchema string) (*FieldMappingConfig, error) {
	mappings := make(map[string][]FieldMapping)

	commonMappings := []FieldMapping{
		{
			SourceField: "data",
			TargetField: "raw_data",
			Transform:   "direct",
		},
		{
			SourceField: "source_file",
			TargetField: "source",
			Transform:   "direct",
		},
		{
			SourceField: "row_number",
			TargetField: "line_number",
			Transform:   "direct",
		},
	}

	mappings["default"] = commonMappings

	return &FieldMappingConfig{
		Version:     "1.0",
		Mappings:    mappings,
		Transforms: map[string]TransformDefinition{
			"direct": {
				Name:        "direct",
				Description: "直接映射，不做任何转换",
				Parameters:  []string{},
			},
		},
		Description: "文件导入结果字段映射配置",
	}, nil
}