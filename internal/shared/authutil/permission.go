package authutil

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type RoleRecord struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Name        string         `gorm:"size:50;not null"`
	Color       string         `gorm:"size:7;not null"`
	Permissions datatypes.JSON `gorm:"type:jsonb"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (RoleRecord) TableName() string {
	return "roles"
}

// GetUserDepartmentIDs returns all department IDs bound to a user,
// checking both the users_departments join table and users.department_id.
func GetUserDepartmentIDs(ctx context.Context, db *gorm.DB, userID uuid.UUID) ([]uuid.UUID, error) {
	if userID == uuid.Nil {
		return []uuid.UUID{}, nil
	}

	deptIDSet := make(map[uuid.UUID]struct{})

	// 1. Check users_departments join table
	type deptJoinRow struct {
		DepartmentID uuid.UUID `gorm:"column:department_id"`
	}
	var joinRows []deptJoinRow
	if err := db.WithContext(ctx).
		Table("users_departments").
		Select("department_id").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Scan(&joinRows).Error; err == nil {
		for _, row := range joinRows {
			if row.DepartmentID != uuid.Nil {
				deptIDSet[row.DepartmentID] = struct{}{}
			}
		}
	}

	// 2. Check primary users.department_id
	var userRow struct {
		DepartmentID *uuid.UUID `gorm:"column:department_id"`
	}
	if err := db.WithContext(ctx).
		Table("users").
		Select("department_id").
		Where("id = ? AND deleted_at IS NULL", userID).
		Scan(&userRow).Error; err == nil && userRow.DepartmentID != nil {
		if *userRow.DepartmentID != uuid.Nil {
			deptIDSet[*userRow.DepartmentID] = struct{}{}
		}
	}

	res := make([]uuid.UUID, 0, len(deptIDSet))
	for id := range deptIDSet {
		res = append(res, id)
	}
	return res, nil
}

// CheckRolePermission verifies if a given role has the required permission on a module.
// It recognizes Admin universal bypass and Siswa as base role.
func CheckRolePermission(ctx context.Context, db *gorm.DB, roleID uuid.UUID, roleName string, module string, action string) (bool, error) {
	// 1. Admin universal bypass
	if strings.EqualFold(roleName, "Admin") {
		return true, nil
	}

	// 2. Siswa base role logic
	if strings.EqualFold(roleName, "Siswa") {
		switch strings.ToLower(module) {
		case "article":
			// Siswa cannot publish, and cannot edit/delete others' articles
			if strings.EqualFold(action, "publish") || strings.EqualFold(action, "update") || strings.EqualFold(action, "delete") {
				return false, nil
			}
			// Siswa can read, write own, update own, delete own
			return true, nil
		case "asset":
			// Siswa can access own gallery (read, write, delete own)
			if strings.EqualFold(action, "update") || strings.EqualFold(action, "delete") {
				return false, nil
			}
			return true, nil
		default:
			// For other modules (company, users, role, department), check explicit permissions
		}
	}

	// 3. Query role from database
	var role RoleRecord
	if err := db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", roleID).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	if strings.EqualFold(role.Name, "Admin") {
		return true, nil
	}

	if len(role.Permissions) == 0 {
		return false, nil
	}

	var permMap map[string]interface{}
	if err := json.Unmarshal(role.Permissions, &permMap); err != nil {
		return false, nil
	}

	// Check admin wildcard flag in permissions JSON
	if adminVal, ok := permMap["admin"]; ok {
		if isAdmin, ok := adminVal.(bool); ok && isAdmin {
			return true, nil
		}
	}

	modVal, ok := permMap[module]
	if !ok {
		return false, nil
	}

	actionsMap, ok := modVal.(map[string]interface{})
	if !ok {
		return false, nil
	}

	isTrue := func(key string) bool {
		val, exists := actionsMap[key]
		if !exists {
			return false
		}
		b, ok := val.(bool)
		return ok && b
	}

	switch strings.ToLower(action) {
	case "read", "view", "access":
		// Has access/read to module if any action is enabled
		for _, v := range actionsMap {
			if b, ok := v.(bool); ok && b {
				return true, nil
			}
		}
		return false, nil
	case "create", "write":
		return isTrue("write") || isTrue("create") || isTrue("update") || isTrue("delete"), nil
	case "update", "edit":
		return isTrue("update"), nil
	case "delete":
		return isTrue("delete"), nil
	case "publish":
		return isTrue("publish"), nil
	default:
		return isTrue(action), nil
	}
}

// GetRolePermissions parses the role's permissions map.
func GetRolePermissions(ctx context.Context, db *gorm.DB, roleID uuid.UUID) (map[string]map[string]bool, bool, error) {
	var role RoleRecord
	if err := db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", roleID).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return map[string]map[string]bool{}, false, nil
		}
		return nil, false, err
	}

	isAdmin := strings.EqualFold(role.Name, "Admin")
	isSiswa := strings.EqualFold(role.Name, "Siswa")

	var permMap map[string]interface{}
	if len(role.Permissions) > 0 {
		_ = json.Unmarshal(role.Permissions, &permMap)
	}
	if permMap == nil {
		permMap = make(map[string]interface{})
	}

	if adminVal, ok := permMap["admin"]; ok {
		if val, ok := adminVal.(bool); ok && val {
			isAdmin = true
		}
	}

	allModules := []string{"users", "article", "company", "asset", "department", "role"}
	res := make(map[string]map[string]bool)

	for _, mod := range allModules {
		modMap := make(map[string]bool)
		if isAdmin {
			modMap["write"] = true
			modMap["update"] = true
			modMap["delete"] = true
			if mod == "article" {
				modMap["publish"] = true
			}
		} else if isSiswa && (mod == "article" || mod == "asset") {
			// Siswa base role has own article and gallery access
			modMap["write"] = true
			modMap["update"] = false
			modMap["delete"] = false
			if mod == "article" {
				modMap["publish"] = false
			}
		} else if rawMod, ok := permMap[mod].(map[string]interface{}); ok {
			for _, act := range []string{"write", "update", "delete", "publish", "create", "read"} {
				if val, exists := rawMod[act]; exists {
					if b, ok := val.(bool); ok {
						// Normalize "create" -> "write" if write is not set
						if act == "create" {
							if !modMap["write"] {
								modMap["write"] = b
							}
						} else {
							modMap[act] = b
						}
					}
				}
			}
		}
		res[mod] = modMap
	}

	return res, isAdmin, nil
}
