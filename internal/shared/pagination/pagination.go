package pagination

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Params holds parsed query parameters for pagination, searching, and sorting.
type Params struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Search string `json:"search,omitempty"`
	SortBy string `json:"sort_by,omitempty"`
	Order  string `json:"order,omitempty"`
}

// Meta holds pagination metadata returned in the API response.
type Meta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalRows  int64 `json:"total_rows"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// FromContext extracts pagination parameters from a Fiber request context with safe defaults.
func FromContext(c *fiber.Ctx) Params {
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "10")
	search := strings.TrimSpace(c.Query("search", ""))
	sortBy := strings.TrimSpace(c.Query("sort_by", "created_at"))
	order := strings.ToLower(strings.TrimSpace(c.Query("order", "desc")))

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	} else if limit > 100 {
		limit = 100 // Cap max items per page to prevent memory exhaustion
	}

	if order != "asc" && order != "desc" {
		order = "desc"
	}

	return Params{
		Page:   page,
		Limit:  limit,
		Search: search,
		SortBy: sortBy,
		Order:  order,
	}
}

// Offset calculates SQL offset.
func (p Params) Offset() int {
	return (p.Page - 1) * p.Limit
}

// SearchPattern returns a wildcard pattern for SQL LIKE/ILIKE queries (e.g. "%keyword%").
func (p Params) SearchPattern() string {
	if p.Search == "" {
		return ""
	}
	return fmt.Sprintf("%%%s%%", p.Search)
}

// BuildMeta computes pagination metadata given the total rows count.
func (p Params) BuildMeta(totalRows int64) Meta {
	totalPages := 1
	if p.Limit > 0 && totalRows > 0 {
		totalPages = int(math.Ceil(float64(totalRows) / float64(p.Limit)))
	}

	return Meta{
		Page:       p.Page,
		Limit:      p.Limit,
		TotalRows:  totalRows,
		TotalPages: totalPages,
		HasNext:    p.Page < totalPages,
		HasPrev:    p.Page > 1,
	}
}

// Scope returns a reusable GORM scope function that automatically applies Offset, Limit, and safe Order.
func (p Params) Scope(allowedSortFields ...string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		sortField := "created_at"
		if len(allowedSortFields) > 0 && p.SortBy != "" {
			for _, allowed := range allowedSortFields {
				if strings.EqualFold(p.SortBy, allowed) {
					sortField = allowed
					break
				}
			}
		} else if p.SortBy != "" {
			// Basic sanitization: only allow alphanumeric and underscore
			isSafe := true
			for _, ch := range p.SortBy {
				if !(ch >= 'a' && ch <= 'z') && !(ch >= 'A' && ch <= 'Z') && !(ch >= '0' && ch <= '9') && ch != '_' {
					isSafe = false
					break
				}
			}
			if isSafe {
				sortField = p.SortBy
			}
		}

		orderClause := fmt.Sprintf("%s %s", sortField, p.Order)

		return db.Offset(p.Offset()).Limit(p.Limit).Order(orderClause)
	}
}
