package uuid

import "github.com/gofrs/uuid/v5"

// SafeUUIDToStringPtr 安全地将UUID转换为字符串指针
func SafeUUIDToStringPtr(id *uuid.UUID) *string {
	if id == nil || *id == uuid.Nil {
		return nil
	}
	str := id.String()
	return &str
}
