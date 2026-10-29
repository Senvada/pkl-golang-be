package assessment

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// Helper to determine letter grade and predicate based on standard scale:
// A = 86 - 100 = Sangat Baik
// B = 71 - 85  = Baik
// C = 56 - 70  = Cukup
// D = 0 - 55   = Kurang
func CalculateGrade(score float64) (string, string) {
	rounded := math.Round(score*100) / 100
	if rounded >= 86 {
		return "A", "Sangat Baik"
	} else if rounded >= 71 {
		return "B", "Baik"
	} else if rounded >= 56 {
		return "C", "Cukup"
	}
	return "D", "Kurang"
}

type AspectTemplateItemDTO struct {
	ID       string `json:"id"`
	Category string `json:"category,omitempty"`
	Title    string `json:"title"`
}

type WorkAspectItemDTO struct {
	ID             string  `json:"id"`
	Category       string  `json:"category,omitempty"`
	Title          string  `json:"title"`
	IsCustom       bool    `json:"is_custom"`
	ScoreNumeric   float64 `json:"score_numeric"`
	ScoreLetter    string  `json:"score_letter"`
	ScorePredicate string  `json:"score_predicate"`
}

type BehaviorAspectItemDTO struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	ScoreNumeric   float64 `json:"score_numeric"`
	ScoreLetter    string  `json:"score_letter"`
	ScorePredicate string  `json:"score_predicate"`
}

// Academic Period DTOs
type CreateAcademicPeriodRequest struct {
	Name       string     `json:"name" validate:"required,min=4,max=50"`
	IsActive   bool       `json:"is_active"`
	CopyFromID *uuid.UUID `json:"copy_from_id"` // Optional: copy all department templates from this period
}

type AcademicPeriodResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Assessment Template DTOs
type CreateAssessmentTemplateRequest struct {
	DepartmentID    uuid.UUID               `json:"department_id" validate:"required"`
	PeriodID        uuid.UUID               `json:"period_id" validate:"required"`
	Title           string                  `json:"title" validate:"required,min=3,max=255"`
	WorkAspects     []AspectTemplateItemDTO `json:"work_aspects"`
	BehaviorAspects []AspectTemplateItemDTO `json:"behavior_aspects"`
}

type UpdateAssessmentTemplateRequest struct {
	Title           string                  `json:"title" validate:"required,min=3,max=255"`
	WorkAspects     []AspectTemplateItemDTO `json:"work_aspects"`
	BehaviorAspects []AspectTemplateItemDTO `json:"behavior_aspects"`
}

type AssessmentTemplateResponse struct {
	ID              uuid.UUID               `json:"id"`
	DepartmentID    uuid.UUID               `json:"department_id"`
	DepartmentName  string                  `json:"department_name"`
	DepartmentCode  string                  `json:"department_code"`
	PeriodID        uuid.UUID               `json:"period_id"`
	PeriodName      string                  `json:"period_name"`
	Title           string                  `json:"title"`
	WorkAspects     []AspectTemplateItemDTO `json:"work_aspects"`
	BehaviorAspects []AspectTemplateItemDTO `json:"behavior_aspects"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

// Student Assessment DTOs
type UpsertStudentAssessmentRequest struct {
	PeriodID        *uuid.UUID              `json:"period_id"` // Optional: defaults to active period
	StudentNIS      string                  `json:"student_nis"`
	StudentClass    string                  `json:"student_class"`
	CompanyID       *uuid.UUID              `json:"company_id"`
	CompanyName     string                  `json:"company_name" validate:"required"`
	CompanyAddress  string                  `json:"company_address"`
	SupervisorName  string                  `json:"supervisor_name" validate:"required"`
	SupervisorNIP   string                  `json:"supervisor_nip"`
	AssessmentCity  string                  `json:"assessment_city"`
	AssessmentDate  *string                 `json:"assessment_date"` // YYYY-MM-DD
	WorkAspects     []WorkAspectItemDTO     `json:"work_aspects"`
	BehaviorAspects []BehaviorAspectItemDTO `json:"behavior_aspects"`
	Status          string                  `json:"status" validate:"omitempty,oneof=draft submitted verified"`
}

type StudentAssessmentResponse struct {
	ID              uuid.UUID               `json:"id"`
	StudentID       uuid.UUID               `json:"student_id"`
	StudentName     string                  `json:"student_name"`
	StudentEmail    string                  `json:"student_email"`
	StudentNIS      string                  `json:"student_nis"`
	StudentClass    string                  `json:"student_class"`
	SchoolName      string                  `json:"school_name"`
	SchoolAddress   string                  `json:"school_address"`
	SchoolPhone     string                  `json:"school_phone"`
	SchoolEmail     string                  `json:"school_email"`
	DepartmentID    uuid.UUID               `json:"department_id"`
	DepartmentName  string                  `json:"department_name"`
	DepartmentCode  string                  `json:"department_code"`
	PeriodID        uuid.UUID               `json:"period_id"`
	PeriodName      string                  `json:"period_name"`
	TemplateID      *uuid.UUID              `json:"template_id"`
	CompanyID       *uuid.UUID              `json:"company_id"`
	CompanyName     string                  `json:"company_name"`
	CompanyAddress  string                  `json:"company_address"`
	SupervisorName  string                  `json:"supervisor_name"`
	SupervisorNIP   string                  `json:"supervisor_nip"`
	AssessmentCity  string                  `json:"assessment_city"`
	AssessmentDate  *string                 `json:"assessment_date"`
	WorkAspects     []WorkAspectItemDTO     `json:"work_aspects"`
	BehaviorAspects []BehaviorAspectItemDTO `json:"behavior_aspects"`
	WorkAverage     float64                 `json:"work_average"`
	BehaviorAverage float64                 `json:"behavior_average"`
	OverallAverage  float64                 `json:"overall_average"`
	Status          string                  `json:"status"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}
