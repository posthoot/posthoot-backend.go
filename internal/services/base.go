package services

import (
	"context"

	"gorm.io/gorm"
)

// BaseService interface defines common CRUD operations
type BaseService[T any] interface {
	Create(ctx context.Context, entity *T) error
	Get(ctx context.Context, id string) (*T, error)
	List(ctx context.Context, page, limit int, filters map[string]interface{}) ([]T, int64, error)
	Update(ctx context.Context, id string, entity *T) error
	Delete(ctx context.Context, id string) error
}

// BaseServiceImpl implements BaseService
type BaseServiceImpl[T any] struct {
	db        *gorm.DB
	modelType T
}

// NewBaseService creates a new base service
func NewBaseService[T any](db *gorm.DB, modelType T) BaseService[T] {
	return &BaseServiceImpl[T]{
		db:        db,
		modelType: modelType,
	}
}

func (s *BaseServiceImpl[T]) Create(ctx context.Context, entity *T) error {
	return s.db.WithContext(ctx).Create(entity).Error
}

func (s *BaseServiceImpl[T]) Get(ctx context.Context, id string) (*T, error) {
	var entity T
	if err := s.db.WithContext(ctx).First(&entity, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

func (s *BaseServiceImpl[T]) List(ctx context.Context, page, limit int, filters map[string]interface{}) ([]T, int64, error) {
	var entities []T
	var total int64

	query := s.db.WithContext(ctx).Model(s.modelType)

	// Apply filters
	for key, value := range filters {
		query = query.Where(key+" = ?", value)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if page > 0 && limit > 0 {
		offset := (page - 1) * limit
		query = query.Offset(offset).Limit(limit)
	}

	// Execute query
	if err := query.Find(&entities).Error; err != nil {
		return nil, 0, err
	}

	return entities, total, nil
}

func (s *BaseServiceImpl[T]) Update(ctx context.Context, id string, entity *T) error {
	return s.db.WithContext(ctx).Model(entity).Where("id = ?", id).Updates(entity).Error
}

func (s *BaseServiceImpl[T]) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(s.modelType, "id = ?", id).Error
}
