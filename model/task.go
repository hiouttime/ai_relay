package model

import (
	"time"

	"gorm.io/gorm"
)

type Task struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Title       string         `json:"title" gorm:"type:varchar(200);not null"`
	Description string         `json:"description" gorm:"type:text"`
	Status      string         `json:"status" gorm:"type:varchar(20);default:pending"` // pending, running, completed, failed
	Priority    int            `json:"priority" gorm:"default:1"`                      // 1:低 2:中 3:高
	UserID      uint           `json:"user_id" gorm:"not null"`
	ScheduleAt  Time           `json:"schedule_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP"`
	CompletedAt *Time          `json:"completed_at" gorm:"type:datetime"`
	CreatedAt   Time           `json:"created_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP"`
	UpdatedAt   Time           `json:"updated_at" gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	User User `json:"user" gorm:"foreignKey:UserID"`
}

func (t *Task) TableName() string {
	return "tasks"
}

func CreateTask(task *Task) error {
	task.ID = 0
	return DB.Create(task).Error
}

func GetTaskById(id uint) (*Task, error) {
	var task Task
	err := DB.Preload("User").First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func UpdateTask(task *Task) error {
	return DB.Save(task).Error
}

func DeleteTask(id uint) error {
	return DB.Delete(&Task{}, id).Error
}

func GetTasks(page, limit int, userID uint) ([]Task, int64, error) {
	var tasks []Task
	var total int64

	query := DB.Model(&Task{}).Preload("User")
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err = query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&tasks).Error
	if err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

func GetTasksByStatus(status string) ([]Task, error) {
	var tasks []Task
	err := DB.Where("status = ?", status).Find(&tasks).Error
	return tasks, err
}

func UpdateTaskStatus(id uint, status string) error {
	updates := map[string]any{
		"status": status,
	}
	if status == "completed" {
		now := time.Now()
		updates["completed_at"] = &now
	}
	return DB.Model(&Task{}).Where("id = ?", id).Updates(updates).Error
}
