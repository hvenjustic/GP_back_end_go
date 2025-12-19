package mysql

import (
	"errors"
	"time"

	"image-api/models/mysql"
	"image-api/pkg/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CrawlTargetDAO struct{}

func NewCrawlTargetDAO() *CrawlTargetDAO {
	return &CrawlTargetDAO{}
}

func (dao *CrawlTargetDAO) UpsertForSubmission(url string, siteName string) (*mysql.CrawlTarget, error) {
	if url == "" {
		return nil, errors.New("url empty")
	}

	now := time.Now()
	var siteNamePtr *string
	if siteName != "" {
		siteNamePtr = &siteName
	}
	create := mysql.CrawlTarget{
		URL:            url,
		SiteName:       siteNamePtr,
		IsCrawled:      false,
		CrawledAt:      nil,
		LLMProcessedAt: nil,
		PageCount:      0,
		ChunkCount:     0,
		ResultMD:       nil,
		GraphJSON:      nil,
		UpdatedAt:      now,
	}

	updates := map[string]any{
		"site_name":        siteNamePtr,
		"is_crawled":       false,
		"crawled_at":       nil,
		"llm_processed_at": nil,
		"page_count":       0,
		"chunk_count":      0,
		"result_md":        nil,
		"graph_json":       nil,
		"updated_at":       now,
	}

	// MySQL 不支持通用 RETURNING，采用 upsert 后再查询。
	if err := db.DB.MysqlDB.DB().
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "url"}},
			DoUpdates: clause.Assignments(updates),
		}).
		Create(&create).Error; err != nil {
		return nil, err
	}

	var out mysql.CrawlTarget
	if err := db.DB.MysqlDB.DB().Where("url = ?", url).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (dao *CrawlTargetDAO) GetDetailByID(id uint64) (*mysql.CrawlTarget, error) {
	var out mysql.CrawlTarget
	if err := db.DB.MysqlDB.DB().Where("id = ?", id).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (dao *CrawlTargetDAO) ApplyResultByIDOrURL(id *uint64, url string, patch map[string]any) (*mysql.CrawlTarget, error) {
	if id == nil && url == "" {
		return nil, errors.New("id/url empty")
	}

	tx := db.DB.MysqlDB.DB().Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var target mysql.CrawlTarget
	var err error
	if id != nil && *id > 0 {
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", *id).First(&target).Error
	} else {
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("url = ?", url).First(&target).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return nil, err
		}
		tx.Rollback()
		return nil, err
	}

	if err := tx.Model(&mysql.CrawlTarget{}).Where("id = ?", target.ID).Updates(patch).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	if err := db.DB.MysqlDB.DB().Where("id = ?", target.ID).First(&target).Error; err != nil {
		return nil, err
	}
	return &target, nil
}

func (dao *CrawlTargetDAO) List(page int, pageSize int) ([]mysql.CrawlTarget, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := db.DB.MysqlDB.DB().Model(&mysql.CrawlTarget{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []mysql.CrawlTarget
	err := db.DB.MysqlDB.DB().
		Model(&mysql.CrawlTarget{}).
		Select("id, site_name, url, crawled_at, llm_processed_at, is_crawled, crawl_count, page_count, chunk_count").
		Order("updated_at desc").
		Limit(pageSize).
		Offset(offset).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
