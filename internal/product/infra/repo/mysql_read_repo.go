package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"go-baseline-skeleton/internal/product/domain"
)

type MySQLReadRepository struct {
	db *sql.DB
}

func NewMySQLReadRepository(db *sql.DB) *MySQLReadRepository {
	return &MySQLReadRepository{db: db}
}

func (r *MySQLReadRepository) ListCategories(ctx context.Context, q domain.CategoryQuery) ([]domain.Category, error) {
	if r == nil || r.db == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "product read db is not initialized", nil)
	}

	var sb strings.Builder
	sb.WriteString("SELECT id, type, name, sort, status FROM category WHERE 1=1")
	args := make([]any, 0, 2)
	if q.Type != nil {
		sb.WriteString(" AND type = ?")
		args = append(args, *q.Type)
	}
	if q.Status != nil {
		sb.WriteString(" AND status = ?")
		args = append(args, *q.Status)
	}
	// Align with Java CategoryMapper.list/pageQuery ordering.
	sb.WriteString(" ORDER BY sort ASC, create_time DESC")

	rows, err := r.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list categories failed: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Category, 0)
	for rows.Next() {
		var item domain.Category
		if err := rows.Scan(&item.ID, &item.Type, &item.Name, &item.Sort, &item.Status); err != nil {
			return nil, fmt.Errorf("scan category failed: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories failed: %w", err)
	}
	return out, nil
}

func (r *MySQLReadRepository) ListDishes(ctx context.Context, q domain.DishQuery) ([]domain.Dish, error) {
	if r == nil || r.db == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "product read db is not initialized", nil)
	}

	var sb strings.Builder
	sb.WriteString("SELECT id, category_id, name, price, image, description, status, sort FROM dish WHERE 1=1")
	args := make([]any, 0, 3)
	if strings.TrimSpace(q.Name) != "" {
		sb.WriteString(" AND name LIKE CONCAT('%', ?, '%')")
		args = append(args, strings.TrimSpace(q.Name))
	}
	if q.CategoryID != nil {
		sb.WriteString(" AND category_id = ?")
		args = append(args, *q.CategoryID)
	}
	if q.Status != nil {
		sb.WriteString(" AND status = ?")
		args = append(args, *q.Status)
	}
	// Align with Java DishMapper.list ordering.
	sb.WriteString(" ORDER BY create_time DESC")

	rows, err := r.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list dishes failed: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Dish, 0)
	for rows.Next() {
		var item domain.Dish
		if err := rows.Scan(&item.ID, &item.CategoryID, &item.Name, &item.Price, &item.Image, &item.Description, &item.Status, &item.Sort); err != nil {
			return nil, fmt.Errorf("scan dish failed: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dishes failed: %w", err)
	}
	return out, nil
}

func (r *MySQLReadRepository) ListDishFlavorsByDishIDs(ctx context.Context, dishIDs []int64) (map[int64][]domain.DishFlavor, error) {
	if r == nil || r.db == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "product read db is not initialized", nil)
	}
	out := make(map[int64][]domain.DishFlavor)
	if len(dishIDs) == 0 {
		return out, nil
	}

	inClause, args := buildInClauseInt64(dishIDs)
	query := "SELECT id, dish_id, name, value FROM dish_flavor WHERE dish_id IN (" + inClause + ") ORDER BY dish_id ASC, id ASC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list dish flavors failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var f domain.DishFlavor
		if err := rows.Scan(&f.ID, &f.DishID, &f.Name, &f.Value); err != nil {
			return nil, fmt.Errorf("scan dish flavor failed: %w", err)
		}
		out[f.DishID] = append(out[f.DishID], f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dish flavors failed: %w", err)
	}
	return out, nil
}

func (r *MySQLReadRepository) GetDishByID(ctx context.Context, id int64) (*domain.Dish, error) {
	if r == nil || r.db == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "product read db is not initialized", nil)
	}

	const query = "SELECT id, category_id, name, price, image, description, status, sort FROM dish WHERE id = ? AND is_deleted = 0 LIMIT 1"
	var item domain.Dish
	err := r.db.QueryRowContext(ctx, query, id).Scan(&item.ID, &item.CategoryID, &item.Name, &item.Price, &item.Image, &item.Description, &item.Status, &item.Sort)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get dish by id failed: %w", err)
	}
	return &item, nil
}

func (r *MySQLReadRepository) ListSetmeals(ctx context.Context, q domain.SetmealQuery) ([]domain.Setmeal, error) {
	if r == nil || r.db == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "product read db is not initialized", nil)
	}

	var sb strings.Builder
	// Align with Java SetmealMapper.list (no ORDER BY in this SQL).
	sb.WriteString("SELECT id, category_id, name, price, image, description, status FROM setmeal WHERE 1=1")
	args := make([]any, 0, 3)
	if strings.TrimSpace(q.Name) != "" {
		sb.WriteString(" AND name LIKE CONCAT('%', ?, '%')")
		args = append(args, strings.TrimSpace(q.Name))
	}
	if q.CategoryID != nil {
		sb.WriteString(" AND category_id = ?")
		args = append(args, *q.CategoryID)
	}
	if q.Status != nil {
		sb.WriteString(" AND status = ?")
		args = append(args, *q.Status)
	}

	rows, err := r.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list setmeals failed: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Setmeal, 0)
	for rows.Next() {
		var item domain.Setmeal
		if err := rows.Scan(&item.ID, &item.CategoryID, &item.Name, &item.Price, &item.Image, &item.Description, &item.Status); err != nil {
			return nil, fmt.Errorf("scan setmeal failed: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate setmeals failed: %w", err)
	}
	return out, nil
}

func (r *MySQLReadRepository) ListSetmealDishesBySetmealIDs(ctx context.Context, setmealIDs []int64) (map[int64][]domain.SetmealDish, error) {
	if r == nil || r.db == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "product read db is not initialized", nil)
	}
	out := make(map[int64][]domain.SetmealDish)
	if len(setmealIDs) == 0 {
		return out, nil
	}

	inClause, args := buildInClauseInt64(setmealIDs)
	query := "SELECT id, setmeal_id, dish_id, name, copies FROM setmeal_dish WHERE setmeal_id IN (" + inClause + ")"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list setmeal dishes failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.SetmealDish
		if err := rows.Scan(&item.ID, &item.SetmealID, &item.DishID, &item.Name, &item.Copies); err != nil {
			return nil, fmt.Errorf("scan setmeal dish failed: %w", err)
		}
		out[item.SetmealID] = append(out[item.SetmealID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate setmeal dishes failed: %w", err)
	}
	return out, nil
}

func (r *MySQLReadRepository) GetSetmealByID(ctx context.Context, id int64) (*domain.Setmeal, error) {
	if r == nil || r.db == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "product read db is not initialized", nil)
	}

	const query = "SELECT id, category_id, name, price, image, description, status FROM setmeal WHERE id = ? AND is_deleted = 0 LIMIT 1"
	var item domain.Setmeal
	err := r.db.QueryRowContext(ctx, query, id).Scan(&item.ID, &item.CategoryID, &item.Name, &item.Price, &item.Image, &item.Description, &item.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get setmeal by id failed: %w", err)
	}
	return &item, nil
}

func buildInClauseInt64(ids []int64) (string, []any) {
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	return strings.Join(placeholders, ","), args
}
