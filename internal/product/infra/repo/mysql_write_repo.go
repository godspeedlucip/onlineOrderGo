package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go-baseline-skeleton/internal/product/domain"
	producttx "go-baseline-skeleton/internal/product/infra/tx"
)

type sqlExecQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type MySQLWriteRepository struct {
	db *sql.DB
}

func NewMySQLWriteRepository(db *sql.DB) *MySQLWriteRepository {
	return &MySQLWriteRepository{db: db}
}

func (r *MySQLWriteRepository) CreateCategory(ctx context.Context, c domain.Category) (int64, error) {
	if err := r.ensureDB(); err != nil {
		return 0, err
	}
	exec := r.execer(ctx)
	exists, err := r.existsCategoryName(ctx, exec, c.ID, c.Type, c.Name)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, domain.NewBizError(domain.CodeConflict, "category name exists", nil)
	}

	now := time.Now()
	if c.Status != domain.StatusEnabled && c.Status != domain.StatusDisabled {
		c.Status = domain.StatusEnabled
	}
	res, err := exec.ExecContext(ctx,
		"INSERT INTO category (type, name, sort, status, create_time, update_time, create_user, update_user) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		c.Type, strings.TrimSpace(c.Name), c.Sort, c.Status, now, now, int64(0), int64(0),
	)
	if err != nil {
		return 0, fmt.Errorf("create category failed: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get category id failed: %w", err)
	}
	return id, nil
}

func (r *MySQLWriteRepository) UpdateCategory(ctx context.Context, c domain.Category, expectedVersion int64) (bool, error) {
	_ = expectedVersion // Java mapper has no version field; keep optimistic hook in app layer.
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	exec := r.execer(ctx)
	exists, err := r.existsCategoryName(ctx, exec, c.ID, c.Type, c.Name)
	if err != nil {
		return false, err
	}
	if exists {
		return false, domain.NewBizError(domain.CodeConflict, "category name exists", nil)
	}
	res, err := exec.ExecContext(ctx,
		"UPDATE category SET type = ?, name = ?, sort = ?, update_time = ?, update_user = ? WHERE id = ?",
		c.Type, strings.TrimSpace(c.Name), c.Sort, time.Now(), int64(0), c.ID,
	)
	if err != nil {
		return false, fmt.Errorf("update category failed: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("update category affected rows failed: %w", err)
	}
	return affected > 0, nil
}

func (r *MySQLWriteRepository) UpdateCategoryStatus(ctx context.Context, id int64, status int) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	res, err := r.execer(ctx).ExecContext(ctx,
		"UPDATE category SET status = ?, update_time = ?, update_user = ? WHERE id = ?",
		status, time.Now(), int64(0), id,
	)
	if err != nil {
		return false, fmt.Errorf("update category status failed: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("update category status affected rows failed: %w", err)
	}
	return affected > 0, nil
}

func (r *MySQLWriteRepository) DeleteCategory(ctx context.Context, id int64) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	res, err := r.execer(ctx).ExecContext(ctx, "DELETE FROM category WHERE id = ?", id)
	if err != nil {
		return false, fmt.Errorf("delete category failed: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete category affected rows failed: %w", err)
	}
	return affected > 0, nil
}

func (r *MySQLWriteRepository) CreateDishWithFlavors(ctx context.Context, d domain.Dish, flavors []domain.DishFlavor) (int64, error) {
	if err := r.ensureDB(); err != nil {
		return 0, err
	}
	exec := r.execer(ctx)
	exists, err := r.existsDishName(ctx, exec, d.ID, d.CategoryID, d.Name)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, domain.NewBizError(domain.CodeConflict, "dish name exists", nil)
	}
	now := time.Now()
	res, err := exec.ExecContext(ctx,
		"INSERT INTO dish (name, category_id, price, image, description, status, create_time, create_user, update_time, update_user) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		strings.TrimSpace(d.Name), d.CategoryID, d.Price, strings.TrimSpace(d.Image), strings.TrimSpace(d.Description), d.Status, now, int64(0), now, int64(0),
	)
	if err != nil {
		return 0, fmt.Errorf("create dish failed: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get dish id failed: %w", err)
	}
	if err := insertDishFlavors(ctx, exec, id, flavors); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *MySQLWriteRepository) UpdateDishWithFlavors(ctx context.Context, d domain.Dish, flavors []domain.DishFlavor, expectedVersion int64) (bool, error) {
	_ = expectedVersion
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	exec := r.execer(ctx)
	exists, err := r.existsDishName(ctx, exec, d.ID, d.CategoryID, d.Name)
	if err != nil {
		return false, err
	}
	if exists {
		return false, domain.NewBizError(domain.CodeConflict, "dish name exists", nil)
	}
	res, err := exec.ExecContext(ctx,
		"UPDATE dish SET name = ?, category_id = ?, price = ?, image = ?, description = ?, status = ?, update_time = ?, update_user = ? WHERE id = ?",
		strings.TrimSpace(d.Name), d.CategoryID, d.Price, strings.TrimSpace(d.Image), strings.TrimSpace(d.Description), d.Status, time.Now(), int64(0), d.ID,
	)
	if err != nil {
		return false, fmt.Errorf("update dish failed: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("update dish affected rows failed: %w", err)
	}
	if affected == 0 {
		return false, nil
	}
	if _, err := exec.ExecContext(ctx, "DELETE FROM dish_flavor WHERE dish_id = ?", d.ID); err != nil {
		return false, fmt.Errorf("delete dish flavors failed: %w", err)
	}
	if err := insertDishFlavors(ctx, exec, d.ID, flavors); err != nil {
		return false, err
	}
	return true, nil
}

func (r *MySQLWriteRepository) UpdateDishStatus(ctx context.Context, id int64, status int) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	res, err := r.execer(ctx).ExecContext(ctx,
		"UPDATE dish SET status = ?, update_time = ?, update_user = ? WHERE id = ?",
		status, time.Now(), int64(0), id,
	)
	if err != nil {
		return false, fmt.Errorf("update dish status failed: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("update dish status affected rows failed: %w", err)
	}
	return affected > 0, nil
}

func (r *MySQLWriteRepository) DeleteDish(ctx context.Context, id int64) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	exec := r.execer(ctx)
	res, err := exec.ExecContext(ctx, "DELETE FROM dish WHERE id = ?", id)
	if err != nil {
		return false, fmt.Errorf("delete dish failed: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete dish affected rows failed: %w", err)
	}
	if affected == 0 {
		return false, nil
	}
	if _, err := exec.ExecContext(ctx, "DELETE FROM dish_flavor WHERE dish_id = ?", id); err != nil {
		return false, fmt.Errorf("delete dish flavor failed: %w", err)
	}
	return true, nil
}

func (r *MySQLWriteRepository) CreateSetmealWithItems(ctx context.Context, s domain.Setmeal, items []domain.SetmealDish) (int64, error) {
	if err := r.ensureDB(); err != nil {
		return 0, err
	}
	exec := r.execer(ctx)
	exists, err := r.existsSetmealName(ctx, exec, s.ID, s.CategoryID, s.Name)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, domain.NewBizError(domain.CodeConflict, "setmeal name exists", nil)
	}
	now := time.Now()
	res, err := exec.ExecContext(ctx,
		"INSERT INTO setmeal (category_id, name, price, status, description, image, create_time, update_time, create_user, update_user) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		s.CategoryID, strings.TrimSpace(s.Name), s.Price, s.Status, strings.TrimSpace(s.Description), strings.TrimSpace(s.Image), now, now, int64(0), int64(0),
	)
	if err != nil {
		return 0, fmt.Errorf("create setmeal failed: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get setmeal id failed: %w", err)
	}
	if err := insertSetmealItems(ctx, exec, id, items); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *MySQLWriteRepository) UpdateSetmealWithItems(ctx context.Context, s domain.Setmeal, items []domain.SetmealDish, expectedVersion int64) (bool, error) {
	_ = expectedVersion
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	exec := r.execer(ctx)
	exists, err := r.existsSetmealName(ctx, exec, s.ID, s.CategoryID, s.Name)
	if err != nil {
		return false, err
	}
	if exists {
		return false, domain.NewBizError(domain.CodeConflict, "setmeal name exists", nil)
	}
	res, err := exec.ExecContext(ctx,
		"UPDATE setmeal SET category_id = ?, name = ?, price = ?, status = ?, description = ?, image = ?, update_time = ?, update_user = ? WHERE id = ?",
		s.CategoryID, strings.TrimSpace(s.Name), s.Price, s.Status, strings.TrimSpace(s.Description), strings.TrimSpace(s.Image), time.Now(), int64(0), s.ID,
	)
	if err != nil {
		return false, fmt.Errorf("update setmeal failed: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("update setmeal affected rows failed: %w", err)
	}
	if affected == 0 {
		return false, nil
	}
	if _, err := exec.ExecContext(ctx, "DELETE FROM setmeal_dish WHERE setmeal_id = ?", s.ID); err != nil {
		return false, fmt.Errorf("delete setmeal items failed: %w", err)
	}
	if err := insertSetmealItems(ctx, exec, s.ID, items); err != nil {
		return false, err
	}
	return true, nil
}

func (r *MySQLWriteRepository) UpdateSetmealStatus(ctx context.Context, id int64, status int) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	res, err := r.execer(ctx).ExecContext(ctx,
		"UPDATE setmeal SET status = ?, update_time = ?, update_user = ? WHERE id = ?",
		status, time.Now(), int64(0), id,
	)
	if err != nil {
		return false, fmt.Errorf("update setmeal status failed: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("update setmeal status affected rows failed: %w", err)
	}
	return affected > 0, nil
}

func (r *MySQLWriteRepository) DeleteSetmeal(ctx context.Context, id int64) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	exec := r.execer(ctx)
	res, err := exec.ExecContext(ctx, "DELETE FROM setmeal WHERE id = ?", id)
	if err != nil {
		return false, fmt.Errorf("delete setmeal failed: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete setmeal affected rows failed: %w", err)
	}
	if affected == 0 {
		return false, nil
	}
	if _, err := exec.ExecContext(ctx, "DELETE FROM setmeal_dish WHERE setmeal_id = ?", id); err != nil {
		return false, fmt.Errorf("delete setmeal items failed: %w", err)
	}
	return true, nil
}

func (r *MySQLWriteRepository) ExistsDishUsedBySetmeal(ctx context.Context, dishID int64) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	var count int64
	if err := r.execer(ctx).QueryRowContext(ctx, "SELECT COUNT(1) FROM setmeal_dish WHERE dish_id = ?", dishID).Scan(&count); err != nil {
		return false, fmt.Errorf("query dish used by setmeal failed: %w", err)
	}
	return count > 0, nil
}

func (r *MySQLWriteRepository) ExistsCategoryUsedByDish(ctx context.Context, categoryID int64) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	var count int64
	if err := r.execer(ctx).QueryRowContext(ctx, "SELECT COUNT(1) FROM dish WHERE category_id = ?", categoryID).Scan(&count); err != nil {
		return false, fmt.Errorf("query category used by dish failed: %w", err)
	}
	return count > 0, nil
}

func (r *MySQLWriteRepository) ExistsCategoryUsedBySetmeal(ctx context.Context, categoryID int64) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	var count int64
	if err := r.execer(ctx).QueryRowContext(ctx, "SELECT COUNT(1) FROM setmeal WHERE category_id = ?", categoryID).Scan(&count); err != nil {
		return false, fmt.Errorf("query category used by setmeal failed: %w", err)
	}
	return count > 0, nil
}

func (r *MySQLWriteRepository) ensureDB() error {
	if r == nil || r.db == nil {
		return domain.NewBizError(domain.CodeInternal, "product write db is not initialized", nil)
	}
	return nil
}

func (r *MySQLWriteRepository) execer(ctx context.Context) sqlExecQuerier {
	if tx, ok := producttx.TxFromContext(ctx); ok && tx != nil {
		return tx
	}
	return r.db
}

func (r *MySQLWriteRepository) existsCategoryName(ctx context.Context, exec sqlExecQuerier, id int64, categoryType int, name string) (bool, error) {
	var count int64
	err := exec.QueryRowContext(ctx,
		"SELECT COUNT(1) FROM category WHERE type = ? AND name = ? AND id <> ?",
		categoryType, strings.TrimSpace(name), id,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("query category duplicate failed: %w", err)
	}
	return count > 0, nil
}

func (r *MySQLWriteRepository) existsDishName(ctx context.Context, exec sqlExecQuerier, id, categoryID int64, name string) (bool, error) {
	var count int64
	err := exec.QueryRowContext(ctx,
		"SELECT COUNT(1) FROM dish WHERE category_id = ? AND name = ? AND id <> ?",
		categoryID, strings.TrimSpace(name), id,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("query dish duplicate failed: %w", err)
	}
	return count > 0, nil
}

func (r *MySQLWriteRepository) existsSetmealName(ctx context.Context, exec sqlExecQuerier, id, categoryID int64, name string) (bool, error) {
	var count int64
	err := exec.QueryRowContext(ctx,
		"SELECT COUNT(1) FROM setmeal WHERE category_id = ? AND name = ? AND id <> ?",
		categoryID, strings.TrimSpace(name), id,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("query setmeal duplicate failed: %w", err)
	}
	return count > 0, nil
}

func insertDishFlavors(ctx context.Context, exec sqlExecQuerier, dishID int64, flavors []domain.DishFlavor) error {
	for _, flavor := range flavors {
		if _, err := exec.ExecContext(ctx,
			"INSERT INTO dish_flavor (dish_id, name, value) VALUES (?, ?, ?)",
			dishID, strings.TrimSpace(flavor.Name), strings.TrimSpace(flavor.Value),
		); err != nil {
			return fmt.Errorf("insert dish flavor failed: %w", err)
		}
	}
	return nil
}

func insertSetmealItems(ctx context.Context, exec sqlExecQuerier, setmealID int64, items []domain.SetmealDish) error {
	for _, item := range items {
		if _, err := exec.ExecContext(ctx,
			"INSERT INTO setmeal_dish (setmeal_id, dish_id, name, price, copies) VALUES (?, ?, ?, ?, ?)",
			setmealID, item.DishID, strings.TrimSpace(item.Name), int64(0), item.Copies,
		); err != nil {
			return fmt.Errorf("insert setmeal item failed: %w", err)
		}
	}
	return nil
}
