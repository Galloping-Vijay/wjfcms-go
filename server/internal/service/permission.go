package service

import (
	"strings"

	"wjfcm-go/internal/model"

	"gorm.io/gorm"
)

func IsSuperAdmin(adminID uint64) bool {
	return adminID == 1
}

func AdminPermissionIDs(db *gorm.DB, adminID uint64, includeAncestors bool) ([]uint, error) {
	var roleIDs []uint
	if err := db.Model(&model.ModelHasRole{}).
		Joins("JOIN "+db.NamingStrategy.TableName("roles")+" ON "+db.NamingStrategy.TableName("roles")+".id = "+db.NamingStrategy.TableName("model_has_roles")+".role_id").
		Where(db.NamingStrategy.TableName("model_has_roles")+".model_type = ? AND "+db.NamingStrategy.TableName("model_has_roles")+".model_id = ?", "App\\Models\\Admin", adminID).
		Where(db.NamingStrategy.TableName("roles")+".guard_name = ? AND "+db.NamingStrategy.TableName("roles")+".status = ?", "admin", 1).
		Pluck(db.NamingStrategy.TableName("model_has_roles")+".role_id", &roleIDs).Error; err != nil {
		return nil, err
	}
	if len(roleIDs) == 0 {
		return []uint{}, nil
	}

	var permissionIDs []uint
	if err := db.Model(&model.RoleHasPermission{}).
		Where("role_id IN ?", roleIDs).
		Pluck("permission_id", &permissionIDs).Error; err != nil {
		return nil, err
	}
	if len(permissionIDs) == 0 {
		return []uint{}, nil
	}
	if !includeAncestors {
		return uniqueUint(permissionIDs), nil
	}
	return PermissionIDsWithAncestors(db, permissionIDs)
}

func AdminPermissionURLs(db *gorm.DB, adminID uint64) ([]string, error) {
	query := db.Model(&model.Permission{}).Where("guard_name = ?", "admin")
	if !IsSuperAdmin(adminID) {
		ids, err := AdminPermissionIDs(db, adminID, false)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return []string{}, nil
		}
		query = query.Where("id IN ?", ids)
	}

	var urls []string
	if err := query.Pluck("url", &urls).Error; err != nil {
		return nil, err
	}
	result := make([]string, 0, len(urls))
	for _, url := range urls {
		url = strings.TrimRight(strings.TrimSpace(url), "/")
		if url != "" {
			result = append(result, url)
		}
	}
	return result, nil
}

func PermissionIDsWithAncestors(db *gorm.DB, ids []uint) ([]uint, error) {
	seen := make(map[uint]bool, len(ids))
	for _, id := range ids {
		seen[id] = true
	}

	current := uniqueUint(ids)
	for len(current) > 0 {
		var parents []uint
		if err := db.Model(&model.Permission{}).
			Where("id IN ? AND parent_id > 0", current).
			Pluck("parent_id", &parents).Error; err != nil {
			return nil, err
		}

		next := make([]uint, 0)
		for _, parentID := range parents {
			if !seen[parentID] {
				seen[parentID] = true
				next = append(next, parentID)
			}
		}
		current = next
	}

	result := make([]uint, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	return result, nil
}

func uniqueUint(items []uint) []uint {
	seen := make(map[uint]bool, len(items))
	result := make([]uint, 0, len(items))
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
