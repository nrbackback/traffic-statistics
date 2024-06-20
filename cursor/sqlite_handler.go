package cursor

import (
	"os"
	"sync"

	"github.com/mitchellh/mapstructure"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"traffic-statistics/pkg/log"
)

const sqliteType = "sqlite"

type sqliteConfig struct {
	DBFile string `mapstructure:"db_file"`
}

type sqliteHandler struct {
	db   *gorm.DB
	lock sync.Mutex
}

type uploadProgress struct {
	File          string
	Completed     bool
	FinishedCount int64
}

func (*uploadProgress) tableName() string {
	return "upload_progress"
}

func newSqliteHandler(config map[string]interface{}) *sqliteHandler {
	var c sqliteConfig
	if err := mapstructure.Decode(config, &c); err != nil {
		log.Fatalw("decode sqlite config failed", "error", err)
	}
	f, err := os.OpenFile(c.DBFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		log.Fatalw("open db file failed", "file", c.DBFile, "error", err)
	}
	f.Close()
	db, err := gorm.Open(sqlite.Open(c.DBFile), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalw("gorm open db file failed", "file", c.DBFile, "error", err)
	}
	p := &uploadProgress{}
	db.Table(p.tableName()).AutoMigrate(&p)
	return &sqliteHandler{
		db: db,
	}
}

func (h *sqliteHandler) AddOneUploadedRecord(filename string) error {
	h.lock.Lock()
	defer h.lock.Unlock()
	r := &uploadProgress{}
	results := h.db.Table(r.tableName()).Where("file = ?", filename).First(r)
	if results.Error != nil {
		if results.Error == gorm.ErrRecordNotFound {
			results := h.db.Table(r.tableName()).Create(&uploadProgress{
				File:          filename,
				Completed:     false,
				FinishedCount: 1,
			})
			if results.Error != nil {
				return results.Error
			}
		}
	} else {
		if err := h.db.Model(&r).Where("file = ?", filename).
			UpdateColumn("finished_count", gorm.Expr("finished_count + ?", 1)).Error; err != nil {
			return err
		}
	}
	return nil
}

func (h *sqliteHandler) MarkFileAsUploaded(filename string) error {
	h.lock.Lock()
	defer h.lock.Unlock()
	if err := h.db.Model(&uploadProgress{}).Where("file = ?", filename).
		UpdateColumn("completed", true).Error; err != nil {
		return err
	}
	return nil
}

func (h *sqliteHandler) IsFileUploaded(filename string) (bool, error) {
	h.lock.Lock()
	defer h.lock.Unlock()
	var exists bool
	if err := h.db.Model(&uploadProgress{}).
		Select("count(*) > 0").
		Where("file = ?", filename).
		Find(&exists).
		Error; err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	var completed bool
	if err := h.db.Model(&uploadProgress{}).Where("file = ?", filename).
		Select("completed").Scan(&completed).Error; err != nil {
		return false, err
	}
	return completed, nil
}

func (h *sqliteHandler) UploadedRecordCount(filename string) (int64, error) {
	h.lock.Lock()
	defer h.lock.Unlock()
	var exists bool
	if err := h.db.Model(&uploadProgress{}).
		Select("count(*) > 0").
		Where("file = ?", filename).
		Find(&exists).
		Error; err != nil {
		return 0, err
	}
	if !exists {
		return 0, nil
	}
	var count int64
	if err := h.db.Model(&uploadProgress{}).Where("file = ?", filename).
		Select("finished_count").Scan(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (h *sqliteHandler) Stop() {
}
