package assessment

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AcademicPeriod struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string    `gorm:"size:50;not null"`
	IsActive  bool      `gorm:"default:false;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (AcademicPeriod) TableName() string {
	return "academic_periods"
}

type AssessmentTemplate struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DepartmentID    uuid.UUID      `gorm:"type:uuid;not null;index"`
	PeriodID        uuid.UUID      `gorm:"type:uuid;not null;index"`
	Title           string         `gorm:"size:255;not null"`
	WorkAspects     datatypes.JSON `gorm:"type:jsonb;not null"`
	BehaviorAspects datatypes.JSON `gorm:"type:jsonb;not null"`
	CreatedBy       *uuid.UUID     `gorm:"type:uuid"`
	UpdatedBy       *uuid.UUID     `gorm:"type:uuid"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	// Associations
	Department *Department     `gorm:"foreignKey:DepartmentID"`
	Period     *AcademicPeriod `gorm:"foreignKey:PeriodID"`
}

func (AssessmentTemplate) TableName() string {
	return "assessment_templates"
}

type StudentAssessment struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StudentID       uuid.UUID      `gorm:"type:uuid;not null;index"`
	DepartmentID    uuid.UUID      `gorm:"type:uuid;not null;index"`
	PeriodID        uuid.UUID      `gorm:"type:uuid;not null;index"`
	TemplateID      *uuid.UUID     `gorm:"type:uuid"`
	CompanyID       *uuid.UUID     `gorm:"type:uuid;index"`
	StudentName     string         `gorm:"size:255;not null"`
	StudentNIS      string         `gorm:"column:student_nis;size:50"`
	StudentClass    string         `gorm:"size:50"`
	SchoolName      string         `gorm:"size:255;not null;default:'SMK Negeri 6 Malang'"`
	SchoolAddress   string         `gorm:"size:255;not null;default:'Jl. Ki Ageng Gribig 28 Malang'"`
	SchoolPhone     string         `gorm:"size:50;not null;default:'(0341) 722216'"`
	SchoolEmail     string         `gorm:"size:100;not null;default:'humas@smkn6malang.sch.id'"`
	DepartmentName  string         `gorm:"size:100;not null"`
	CompanyName     string         `gorm:"size:255;not null"`
	CompanyAddress  string         `gorm:"type:text"`
	SupervisorName  string         `gorm:"size:255;not null"`
	SupervisorNIP   string         `gorm:"column:supervisor_nip;size:100"`
	AssessmentCity  string         `gorm:"size:100;not null;default:'Malang'"`
	AssessmentDate  *time.Time     `gorm:"type:date"`
	WorkAspects     datatypes.JSON `gorm:"type:jsonb;not null"`
	BehaviorAspects datatypes.JSON `gorm:"type:jsonb;not null"`
	WorkAverage     float64        `gorm:"type:numeric(5,2);not null;default:0"`
	BehaviorAverage float64        `gorm:"type:numeric(5,2);not null;default:0"`
	OverallAverage  float64        `gorm:"type:numeric(5,2);not null;default:0"`
	Status          string         `gorm:"size:20;not null;default:'draft'"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	// Associations
	Student    *User               `gorm:"foreignKey:StudentID"`
	Department *Department         `gorm:"foreignKey:DepartmentID"`
	Period     *AcademicPeriod     `gorm:"foreignKey:PeriodID"`
	Template   *AssessmentTemplate `gorm:"foreignKey:TemplateID"`
}

func (StudentAssessment) TableName() string {
	return "student_assessments"
}

// Light relation models for eager loading
type Department struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Name      string         `gorm:"size:100;not null"`
	Code      string         `gorm:"size:20;not null"`
	Color     string         `gorm:"size:7;not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Department) TableName() string {
	return "departments"
}

type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Name         string         `gorm:"size:255;not null"`
	Email        string         `gorm:"size:255;not null"`
	DepartmentID *uuid.UUID     `gorm:"type:uuid"`
	CompanyID    *uuid.UUID     `gorm:"type:uuid"`
	ParentID     *uuid.UUID     `gorm:"type:uuid"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (User) TableName() string {
	return "users"
}
