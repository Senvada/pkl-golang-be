package assessment

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/shared/apperror"
	"pkl-golang-be/internal/shared/authutil"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Service interface {
	// Academic Periods
	GetAllAcademicPeriods(ctx context.Context) ([]AcademicPeriodResponse, error)
	GetActiveAcademicPeriod(ctx context.Context) (*AcademicPeriodResponse, error)
	CreateAcademicPeriod(ctx context.Context, req CreateAcademicPeriodRequest) (*AcademicPeriodResponse, error)
	SetActiveAcademicPeriod(ctx context.Context, periodID uuid.UUID) (*AcademicPeriodResponse, error)

	// Assessment Templates
	GetAssessmentTemplates(ctx context.Context, userID, roleID uuid.UUID, roleName string, periodID *uuid.UUID, deptID *uuid.UUID) ([]AssessmentTemplateResponse, error)
	GetAssessmentTemplateByID(ctx context.Context, id uuid.UUID) (*AssessmentTemplateResponse, error)
	GetAssessmentTemplateByDeptAndPeriod(ctx context.Context, deptID, periodID uuid.UUID) (*AssessmentTemplateResponse, error)
	CreateAssessmentTemplate(ctx context.Context, userID, roleID uuid.UUID, roleName string, req CreateAssessmentTemplateRequest) (*AssessmentTemplateResponse, error)
	UpdateAssessmentTemplate(ctx context.Context, userID, roleID uuid.UUID, roleName string, id uuid.UUID, req UpdateAssessmentTemplateRequest) (*AssessmentTemplateResponse, error)

	// Student Assessments
	GetMyStudentAssessment(ctx context.Context, studentID uuid.UUID, periodID *uuid.UUID) (*StudentAssessmentResponse, error)
	UpsertStudentAssessment(ctx context.Context, studentID uuid.UUID, req UpsertStudentAssessmentRequest) (*StudentAssessmentResponse, error)
	GetStudentAssessmentByID(ctx context.Context, id, userID, roleID uuid.UUID, roleName string) (*StudentAssessmentResponse, error)
	ListStudentAssessments(ctx context.Context, userID, roleID uuid.UUID, roleName string, periodID *uuid.UUID, deptID *uuid.UUID, search string) ([]StudentAssessmentResponse, error)
}

type service struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewService(db *gorm.DB, cfg *config.Config) Service {
	return &service{
		db:  db,
		cfg: cfg,
	}
}

// -----------------------------------------------------------------------------
// Academic Periods
// -----------------------------------------------------------------------------

func (s *service) GetAllAcademicPeriods(ctx context.Context) ([]AcademicPeriodResponse, error) {
	var periods []AcademicPeriod
	if err := s.db.WithContext(ctx).
		Order("created_at desc").
		Find(&periods).Error; err != nil {
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	responses := make([]AcademicPeriodResponse, len(periods))
	for i, p := range periods {
		responses[i] = AcademicPeriodResponse{
			ID:        p.ID,
			Name:      p.Name,
			IsActive:  p.IsActive,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		}
	}
	return responses, nil
}

func (s *service) GetActiveAcademicPeriod(ctx context.Context) (*AcademicPeriodResponse, error) {
	var period AcademicPeriod
	err := s.db.WithContext(ctx).
		Where("is_active = ?", true).
		First(&period).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Fallback to latest period
		if err := s.db.WithContext(ctx).Order("created_at desc").First(&period).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperror.NotFound("No academic period found")
			}
			return nil, apperror.Internal("Terjadi kesalahan internal", err)
		}
	} else if err != nil {
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	return &AcademicPeriodResponse{
		ID:        period.ID,
		Name:      period.Name,
		IsActive:  period.IsActive,
		CreatedAt: period.CreatedAt,
		UpdatedAt: period.UpdatedAt,
	}, nil
}

func (s *service) CreateAcademicPeriod(ctx context.Context, req CreateAcademicPeriodRequest) (*AcademicPeriodResponse, error) {
	var existing AcademicPeriod
	err := s.db.WithContext(ctx).Where("name = ?", req.Name).First(&existing).Error
	if err == nil {
		return nil, apperror.Conflict("Academic period with this name already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	newPeriod := AcademicPeriod{
		ID:       uuid.New(),
		Name:     req.Name,
		IsActive: req.IsActive,
	}

	tx := s.db.WithContext(ctx).Begin()
	if req.IsActive {
		if err := tx.Model(&AcademicPeriod{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
			tx.Rollback()
			return nil, apperror.Internal("Terjadi kesalahan internal", err)
		}
	}

	if err := tx.Create(&newPeriod).Error; err != nil {
		tx.Rollback()
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	// Copy templates from previous period if requested
	if req.CopyFromID != nil {
		var srcTemplates []AssessmentTemplate
		if err := tx.Where("period_id = ?", *req.CopyFromID).Find(&srcTemplates).Error; err == nil {
			for _, src := range srcTemplates {
				copiedTpl := AssessmentTemplate{
					ID:              uuid.New(),
					DepartmentID:    src.DepartmentID,
					PeriodID:        newPeriod.ID,
					Title:           src.Title,
					WorkAspects:     src.WorkAspects,
					BehaviorAspects: src.BehaviorAspects,
				}
				_ = tx.Create(&copiedTpl).Error
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	return &AcademicPeriodResponse{
		ID:        newPeriod.ID,
		Name:      newPeriod.Name,
		IsActive:  newPeriod.IsActive,
		CreatedAt: newPeriod.CreatedAt,
		UpdatedAt: newPeriod.UpdatedAt,
	}, nil
}

func (s *service) SetActiveAcademicPeriod(ctx context.Context, periodID uuid.UUID) (*AcademicPeriodResponse, error) {
	var period AcademicPeriod
	if err := s.db.WithContext(ctx).First(&period, "id = ?", periodID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Academic period not found")
		}
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	tx := s.db.WithContext(ctx).Begin()
	if err := tx.Model(&AcademicPeriod{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		tx.Rollback()
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	period.IsActive = true
	if err := tx.Save(&period).Error; err != nil {
		tx.Rollback()
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	return &AcademicPeriodResponse{
		ID:        period.ID,
		Name:      period.Name,
		IsActive:  period.IsActive,
		CreatedAt: period.CreatedAt,
		UpdatedAt: period.UpdatedAt,
	}, nil
}

// -----------------------------------------------------------------------------
// Assessment Templates
// -----------------------------------------------------------------------------

func (s *service) checkUserDepartmentAccess(ctx context.Context, userID, roleID uuid.UUID, roleName string, targetDeptID uuid.UUID) error {
	if strings.EqualFold(roleName, "Admin") {
		return nil
	}

	allowedDeptIDs, err := authutil.GetUserDepartmentIDs(ctx, s.db, userID)
	if err != nil {
		return apperror.Internal("Terjadi kesalahan internal", err)
	}

	for _, dID := range allowedDeptIDs {
		if dID == targetDeptID {
			return nil
		}
	}

	return apperror.Forbidden("You do not have permission to manage assessment templates for this department")
}

func (s *service) GetAssessmentTemplates(ctx context.Context, userID, roleID uuid.UUID, roleName string, periodID *uuid.UUID, deptID *uuid.UUID) ([]AssessmentTemplateResponse, error) {
	query := s.db.WithContext(ctx).
		Preload("Department").
		Preload("Period").
		Order("created_at asc")

	// Scope by user's departments if not admin
	if !strings.EqualFold(roleName, "Admin") {
		allowedDeptIDs, err := authutil.GetUserDepartmentIDs(ctx, s.db, userID)
		if err != nil {
			return nil, apperror.Internal("Terjadi kesalahan internal", err)
		}
		if len(allowedDeptIDs) == 0 {
			return []AssessmentTemplateResponse{}, nil
		}
		query = query.Where("department_id IN (?)", allowedDeptIDs)
	}

	if deptID != nil && *deptID != uuid.Nil {
		query = query.Where("department_id = ?", *deptID)
	}

	if periodID != nil && *periodID != uuid.Nil {
		query = query.Where("period_id = ?", *periodID)
	} else {
		// Default to active period
		var activePeriod AcademicPeriod
		if err := s.db.WithContext(ctx).Where("is_active = ?", true).First(&activePeriod).Error; err == nil {
			query = query.Where("period_id = ?", activePeriod.ID)
		}
	}

	var templates []AssessmentTemplate
	if err := query.Find(&templates).Error; err != nil {
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	responses := make([]AssessmentTemplateResponse, len(templates))
	for i, t := range templates {
		responses[i] = s.toTemplateResponse(&t)
	}
	return responses, nil
}

func (s *service) GetAssessmentTemplateByID(ctx context.Context, id uuid.UUID) (*AssessmentTemplateResponse, error) {
	var tpl AssessmentTemplate
	if err := s.db.WithContext(ctx).
		Preload("Department").
		Preload("Period").
		First(&tpl, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Assessment template not found")
		}
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	resp := s.toTemplateResponse(&tpl)
	return &resp, nil
}

func (s *service) GetAssessmentTemplateByDeptAndPeriod(ctx context.Context, deptID, periodID uuid.UUID) (*AssessmentTemplateResponse, error) {
	var tpl AssessmentTemplate
	if err := s.db.WithContext(ctx).
		Preload("Department").
		Preload("Period").
		Where("department_id = ? AND period_id = ?", deptID, periodID).
		First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Assessment template not found for this department and period")
		}
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	resp := s.toTemplateResponse(&tpl)
	return &resp, nil
}

func (s *service) CreateAssessmentTemplate(ctx context.Context, userID, roleID uuid.UUID, roleName string, req CreateAssessmentTemplateRequest) (*AssessmentTemplateResponse, error) {
	if err := s.checkUserDepartmentAccess(ctx, userID, roleID, roleName, req.DepartmentID); err != nil {
		return nil, err
	}

	var count int64
	s.db.WithContext(ctx).Model(&AssessmentTemplate{}).
		Where("department_id = ? AND period_id = ?", req.DepartmentID, req.PeriodID).
		Count(&count)
	if count > 0 {
		return nil, apperror.Conflict("Assessment template for this department and academic period already exists")
	}

	// Ensure item IDs are present
	for i := range req.WorkAspects {
		if req.WorkAspects[i].ID == "" {
			req.WorkAspects[i].ID = uuid.NewString()
		}
	}
	for i := range req.BehaviorAspects {
		if req.BehaviorAspects[i].ID == "" {
			req.BehaviorAspects[i].ID = uuid.NewString()
		}
	}

	workJSON, _ := json.Marshal(req.WorkAspects)
	behaviorJSON, _ := json.Marshal(req.BehaviorAspects)

	tpl := AssessmentTemplate{
		ID:              uuid.New(),
		DepartmentID:    req.DepartmentID,
		PeriodID:        req.PeriodID,
		Title:           req.Title,
		WorkAspects:     datatypes.JSON(workJSON),
		BehaviorAspects: datatypes.JSON(behaviorJSON),
		CreatedBy:       &userID,
		UpdatedBy:       &userID,
	}

	if err := s.db.WithContext(ctx).Create(&tpl).Error; err != nil {
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	return s.GetAssessmentTemplateByID(ctx, tpl.ID)
}

func (s *service) UpdateAssessmentTemplate(ctx context.Context, userID, roleID uuid.UUID, roleName string, id uuid.UUID, req UpdateAssessmentTemplateRequest) (*AssessmentTemplateResponse, error) {
	var tpl AssessmentTemplate
	if err := s.db.WithContext(ctx).First(&tpl, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Assessment template not found")
		}
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	if err := s.checkUserDepartmentAccess(ctx, userID, roleID, roleName, tpl.DepartmentID); err != nil {
		return nil, err
	}

	for i := range req.WorkAspects {
		if req.WorkAspects[i].ID == "" {
			req.WorkAspects[i].ID = uuid.NewString()
		}
	}
	for i := range req.BehaviorAspects {
		if req.BehaviorAspects[i].ID == "" {
			req.BehaviorAspects[i].ID = uuid.NewString()
		}
	}

	workJSON, _ := json.Marshal(req.WorkAspects)
	behaviorJSON, _ := json.Marshal(req.BehaviorAspects)

	tpl.Title = req.Title
	tpl.WorkAspects = datatypes.JSON(workJSON)
	tpl.BehaviorAspects = datatypes.JSON(behaviorJSON)
	tpl.UpdatedBy = &userID

	if err := s.db.WithContext(ctx).Save(&tpl).Error; err != nil {
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	return s.GetAssessmentTemplateByID(ctx, tpl.ID)
}

func (s *service) toTemplateResponse(t *AssessmentTemplate) AssessmentTemplateResponse {
	resp := AssessmentTemplateResponse{
		ID:              t.ID,
		DepartmentID:    t.DepartmentID,
		PeriodID:        t.PeriodID,
		Title:           t.Title,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
		WorkAspects:     []AspectTemplateItemDTO{},
		BehaviorAspects: []AspectTemplateItemDTO{},
	}

	if t.Department != nil {
		resp.DepartmentName = t.Department.Name
		resp.DepartmentCode = t.Department.Code
	}
	if t.Period != nil {
		resp.PeriodName = t.Period.Name
	}

	if len(t.WorkAspects) > 0 {
		_ = json.Unmarshal(t.WorkAspects, &resp.WorkAspects)
	}
	if len(t.BehaviorAspects) > 0 {
		_ = json.Unmarshal(t.BehaviorAspects, &resp.BehaviorAspects)
	}

	return resp
}

// -----------------------------------------------------------------------------
// Student Assessments
// -----------------------------------------------------------------------------

func (s *service) GetMyStudentAssessment(ctx context.Context, studentID uuid.UUID, periodID *uuid.UUID) (*StudentAssessmentResponse, error) {
	var student User
	if err := s.db.WithContext(ctx).First(&student, "id = ?", studentID).Error; err != nil {
		return nil, apperror.NotFound("Student user not found")
	}

	if student.DepartmentID == nil {
		return nil, apperror.BadRequest("Student is not assigned to any department", nil)
	}

	// Resolve academic period
	var targetPeriod AcademicPeriod
	if periodID != nil && *periodID != uuid.Nil {
		if err := s.db.WithContext(ctx).First(&targetPeriod, "id = ?", *periodID).Error; err != nil {
			return nil, apperror.NotFound("Academic period not found")
		}
	} else {
		if err := s.db.WithContext(ctx).Where("is_active = ?", true).First(&targetPeriod).Error; err != nil {
			if err := s.db.WithContext(ctx).Order("created_at desc").First(&targetPeriod).Error; err != nil {
				return nil, apperror.NotFound("No active academic period found")
			}
		}
	}

	// Check if student assessment already exists
	var sa StudentAssessment
	err := s.db.WithContext(ctx).
		Preload("Student").
		Preload("Department").
		Preload("Period").
		Where("student_id = ? AND period_id = ?", studentID, targetPeriod.ID).
		First(&sa).Error

	if err == nil {
		resp := s.toStudentAssessmentResponse(&sa)
		return &resp, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	// Not found yet -> construct draft from department template
	var tpl AssessmentTemplate
	err = s.db.WithContext(ctx).
		Preload("Department").
		Where("department_id = ? AND period_id = ?", *student.DepartmentID, targetPeriod.ID).
		First(&tpl).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	var dept Department
	_ = s.db.WithContext(ctx).First(&dept, "id = ?", *student.DepartmentID).Error

	companyName := ""
	companyAddress := ""
	if student.CompanyID != nil {
		type compLight struct {
			Name    string
			Address string
		}
		var comp compLight
		if err := s.db.WithContext(ctx).Table("companies").Select("name, address").Where("id = ?", *student.CompanyID).Scan(&comp).Error; err == nil {
			companyName = comp.Name
			companyAddress = comp.Address
		}
	}

	workItems := make([]WorkAspectItemDTO, 0)
	behaviorItems := make([]BehaviorAspectItemDTO, 0)

	if len(tpl.WorkAspects) > 0 {
		var aspectTemplates []AspectTemplateItemDTO
		if err := json.Unmarshal(tpl.WorkAspects, &aspectTemplates); err == nil {
			for _, item := range aspectTemplates {
				workItems = append(workItems, WorkAspectItemDTO{
					ID:             item.ID,
					Category:       item.Category,
					Title:          item.Title,
					IsCustom:       false,
					ScoreNumeric:   0,
					ScoreLetter:    "-",
					ScorePredicate: "-",
				})
			}
		}
	}

	if len(tpl.BehaviorAspects) > 0 {
		var behTemplates []AspectTemplateItemDTO
		if err := json.Unmarshal(tpl.BehaviorAspects, &behTemplates); err == nil {
			for _, item := range behTemplates {
				behaviorItems = append(behaviorItems, BehaviorAspectItemDTO{
					ID:             item.ID,
					Title:          item.Title,
					ScoreNumeric:   0,
					ScoreLetter:    "-",
					ScorePredicate: "-",
				})
			}
		}
	}

	resp := StudentAssessmentResponse{
		ID:              uuid.Nil,
		StudentID:       studentID,
		StudentName:     student.Name,
		StudentEmail:    student.Email,
		DepartmentID:    *student.DepartmentID,
		DepartmentName:  dept.Name,
		DepartmentCode:  dept.Code,
		PeriodID:        targetPeriod.ID,
		PeriodName:      targetPeriod.Name,
		TemplateID:      &tpl.ID,
		CompanyID:       student.CompanyID,
		CompanyName:     companyName,
		CompanyAddress:  companyAddress,
		SchoolName:      "SMK Negeri 6 Malang",
		SchoolAddress:   "Jl. Ki Ageng Gribig 28 Malang",
		SchoolPhone:     "(0341) 722216",
		SchoolEmail:     "humas@smkn6malang.sch.id",
		AssessmentCity:  "Malang",
		WorkAspects:     workItems,
		BehaviorAspects: behaviorItems,
		WorkAverage:     0,
		BehaviorAverage: 0,
		OverallAverage:  0,
		Status:          "draft",
	}

	return &resp, nil
}

func (s *service) UpsertStudentAssessment(ctx context.Context, studentID uuid.UUID, req UpsertStudentAssessmentRequest) (*StudentAssessmentResponse, error) {
	var student User
	if err := s.db.WithContext(ctx).First(&student, "id = ?", studentID).Error; err != nil {
		return nil, apperror.NotFound("Student user not found")
	}

	if student.DepartmentID == nil {
		return nil, apperror.BadRequest("Student is not assigned to any department", nil)
	}

	// Resolve academic period
	var targetPeriod AcademicPeriod
	if req.PeriodID != nil && *req.PeriodID != uuid.Nil {
		if err := s.db.WithContext(ctx).First(&targetPeriod, "id = ?", *req.PeriodID).Error; err != nil {
			return nil, apperror.NotFound("Academic period not found")
		}
	} else {
		if err := s.db.WithContext(ctx).Where("is_active = ?", true).First(&targetPeriod).Error; err != nil {
			if err := s.db.WithContext(ctx).Order("created_at desc").First(&targetPeriod).Error; err != nil {
				return nil, apperror.NotFound("No active academic period found")
			}
		}
	}

	// Fetch template for this department and period
	var tpl AssessmentTemplate
	var templateID *uuid.UUID
	if err := s.db.WithContext(ctx).Where("department_id = ? AND period_id = ?", *student.DepartmentID, targetPeriod.ID).First(&tpl).Error; err == nil {
		templateID = &tpl.ID
	}

	// Validate aspects and compute grades
	var totalWorkScore float64
	var validWorkCount int
	for i := range req.WorkAspects {
		item := &req.WorkAspects[i]
		if item.ID == "" {
			item.ID = uuid.NewString()
			item.IsCustom = true
		}
		if item.ScoreNumeric > 0 {
			letter, pred := CalculateGrade(item.ScoreNumeric)
			item.ScoreLetter = letter
			item.ScorePredicate = pred
			totalWorkScore += item.ScoreNumeric
			validWorkCount++
		} else {
			item.ScoreLetter = "-"
			item.ScorePredicate = "-"
		}
	}

	var totalBehScore float64
	var validBehCount int
	for i := range req.BehaviorAspects {
		item := &req.BehaviorAspects[i]
		if item.ID == "" {
			item.ID = uuid.NewString()
		}
		if item.ScoreNumeric > 0 {
			letter, pred := CalculateGrade(item.ScoreNumeric)
			item.ScoreLetter = letter
			item.ScorePredicate = pred
			totalBehScore += item.ScoreNumeric
			validBehCount++
		} else {
			item.ScoreLetter = "-"
			item.ScorePredicate = "-"
		}
	}

	var workAvg float64
	if validWorkCount > 0 {
		workAvg = math.Round((totalWorkScore/float64(validWorkCount))*100) / 100
	}

	var behAvg float64
	if validBehCount > 0 {
		behAvg = math.Round((totalBehScore/float64(validBehCount))*100) / 100
	}

	var overallAvg float64
	if validWorkCount > 0 && validBehCount > 0 {
		overallAvg = math.Round(((workAvg+behAvg)/2)*100) / 100
	} else if validWorkCount > 0 {
		overallAvg = workAvg
	} else if validBehCount > 0 {
		overallAvg = behAvg
	}

	status := "draft"
	if req.Status != "" {
		status = req.Status
	}

	var parsedDate *time.Time
	if req.AssessmentDate != nil && *req.AssessmentDate != "" {
		if t, err := time.Parse("2006-01-02", *req.AssessmentDate); err == nil {
			parsedDate = &t
		}
	}

	var dept Department
	_ = s.db.WithContext(ctx).First(&dept, "id = ?", *student.DepartmentID).Error

	workJSON, _ := json.Marshal(req.WorkAspects)
	behJSON, _ := json.Marshal(req.BehaviorAspects)

	var sa StudentAssessment
	err := s.db.WithContext(ctx).
		Where("student_id = ? AND period_id = ?", studentID, targetPeriod.ID).
		First(&sa).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		sa = StudentAssessment{
			ID:              uuid.New(),
			StudentID:       studentID,
			DepartmentID:    *student.DepartmentID,
			PeriodID:        targetPeriod.ID,
			TemplateID:      templateID,
			CompanyID:       req.CompanyID,
			StudentName:     student.Name,
			StudentNIS:      req.StudentNIS,
			StudentClass:    req.StudentClass,
			SchoolName:      "SMK Negeri 6 Malang",
			SchoolAddress:   "Jl. Ki Ageng Gribig 28 Malang",
			SchoolPhone:     "(0341) 722216",
			SchoolEmail:     "humas@smkn6malang.sch.id",
			DepartmentName:  dept.Name,
			CompanyName:     req.CompanyName,
			CompanyAddress:  req.CompanyAddress,
			SupervisorName:  req.SupervisorName,
			SupervisorNIP:   req.SupervisorNIP,
			AssessmentCity:  req.AssessmentCity,
			AssessmentDate:  parsedDate,
			WorkAspects:     datatypes.JSON(workJSON),
			BehaviorAspects: datatypes.JSON(behJSON),
			WorkAverage:     workAvg,
			BehaviorAverage: behAvg,
			OverallAverage:  overallAvg,
			Status:          status,
		}
		if sa.AssessmentCity == "" {
			sa.AssessmentCity = "Malang"
		}
		if err := s.db.WithContext(ctx).Create(&sa).Error; err != nil {
			return nil, apperror.Internal("Terjadi kesalahan internal", err)
		}
	} else if err == nil {
		if sa.Status == "verified" {
			return nil, apperror.BadRequest("Assessment has already been verified and cannot be edited", nil)
		}

		sa.CompanyID = req.CompanyID
		sa.StudentNIS = req.StudentNIS
		sa.StudentClass = req.StudentClass
		sa.CompanyName = req.CompanyName
		sa.CompanyAddress = req.CompanyAddress
		sa.SupervisorName = req.SupervisorName
		sa.SupervisorNIP = req.SupervisorNIP
		if req.AssessmentCity != "" {
			sa.AssessmentCity = req.AssessmentCity
		}
		sa.AssessmentDate = parsedDate
		sa.WorkAspects = datatypes.JSON(workJSON)
		sa.BehaviorAspects = datatypes.JSON(behJSON)
		sa.WorkAverage = workAvg
		sa.BehaviorAverage = behAvg
		sa.OverallAverage = overallAvg
		sa.Status = status
		if templateID != nil {
			sa.TemplateID = templateID
		}

		if err := s.db.WithContext(ctx).Save(&sa).Error; err != nil {
			return nil, apperror.Internal("Terjadi kesalahan internal", err)
		}
	} else {
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	return s.GetStudentAssessmentByID(ctx, sa.ID, studentID, uuid.Nil, "Siswa")
}

func (s *service) GetStudentAssessmentByID(ctx context.Context, id, userID, roleID uuid.UUID, roleName string) (*StudentAssessmentResponse, error) {
	var sa StudentAssessment
	if err := s.db.WithContext(ctx).
		Preload("Student").
		Preload("Department").
		Preload("Period").
		First(&sa, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Student assessment not found")
		}
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	// Authorization check
	if !strings.EqualFold(roleName, "Admin") {
		if sa.StudentID != userID {
			// Check if user is a supervisor for this student's department
			allowedDeptIDs, err := authutil.GetUserDepartmentIDs(ctx, s.db, userID)
			if err != nil {
				return nil, apperror.Internal("Terjadi kesalahan internal", err)
			}
			allowed := false
			for _, dID := range allowedDeptIDs {
				if dID == sa.DepartmentID {
					allowed = true
					break
				}
			}
			if !allowed {
				return nil, apperror.Forbidden("You do not have access to view this student's assessment")
			}
		}
	}

	resp := s.toStudentAssessmentResponse(&sa)
	return &resp, nil
}

func (s *service) ListStudentAssessments(ctx context.Context, userID, roleID uuid.UUID, roleName string, periodID *uuid.UUID, deptID *uuid.UUID, search string) ([]StudentAssessmentResponse, error) {
	query := s.db.WithContext(ctx).
		Preload("Student").
		Preload("Department").
		Preload("Period").
		Order("updated_at desc")

	if !strings.EqualFold(roleName, "Admin") {
		allowedDeptIDs, err := authutil.GetUserDepartmentIDs(ctx, s.db, userID)
		if err != nil {
			return nil, apperror.Internal("Terjadi kesalahan internal", err)
		}
		if len(allowedDeptIDs) == 0 {
			return []StudentAssessmentResponse{}, nil
		}
		query = query.Where("department_id IN (?)", allowedDeptIDs)
	}

	if deptID != nil && *deptID != uuid.Nil {
		query = query.Where("department_id = ?", *deptID)
	}

	if periodID != nil && *periodID != uuid.Nil {
		query = query.Where("period_id = ?", *periodID)
	}

	if strings.TrimSpace(search) != "" {
		term := "%" + strings.TrimSpace(search) + "%"
		query = query.Where("student_name ILIKE ? OR company_name ILIKE ? OR student_nis ILIKE ?", term, term, term)
	}

	var assessments []StudentAssessment
	if err := query.Find(&assessments).Error; err != nil {
		return nil, apperror.Internal("Terjadi kesalahan internal", err)
	}

	responses := make([]StudentAssessmentResponse, len(assessments))
	for i, sa := range assessments {
		responses[i] = s.toStudentAssessmentResponse(&sa)
	}
	return responses, nil
}

func (s *service) toStudentAssessmentResponse(sa *StudentAssessment) StudentAssessmentResponse {
	resp := StudentAssessmentResponse{
		ID:              sa.ID,
		StudentID:       sa.StudentID,
		StudentName:     sa.StudentName,
		StudentNIS:      sa.StudentNIS,
		StudentClass:    sa.StudentClass,
		SchoolName:      sa.SchoolName,
		SchoolAddress:   sa.SchoolAddress,
		SchoolPhone:     sa.SchoolPhone,
		SchoolEmail:     sa.SchoolEmail,
		DepartmentID:    sa.DepartmentID,
		DepartmentName:  sa.DepartmentName,
		PeriodID:        sa.PeriodID,
		TemplateID:      sa.TemplateID,
		CompanyID:       sa.CompanyID,
		CompanyName:     sa.CompanyName,
		CompanyAddress:  sa.CompanyAddress,
		SupervisorName:  sa.SupervisorName,
		SupervisorNIP:   sa.SupervisorNIP,
		AssessmentCity:  sa.AssessmentCity,
		WorkAverage:     sa.WorkAverage,
		BehaviorAverage: sa.BehaviorAverage,
		OverallAverage:  sa.OverallAverage,
		Status:          sa.Status,
		CreatedAt:       sa.CreatedAt,
		UpdatedAt:       sa.UpdatedAt,
		WorkAspects:     []WorkAspectItemDTO{},
		BehaviorAspects: []BehaviorAspectItemDTO{},
	}

	if sa.Student != nil {
		resp.StudentEmail = sa.Student.Email
	}
	if sa.Department != nil {
		resp.DepartmentCode = sa.Department.Code
		if resp.DepartmentName == "" {
			resp.DepartmentName = sa.Department.Name
		}
	}
	if sa.Period != nil {
		resp.PeriodName = sa.Period.Name
	}
	if sa.AssessmentDate != nil {
		dStr := sa.AssessmentDate.Format("2006-01-02")
		resp.AssessmentDate = &dStr
	}

	if len(sa.WorkAspects) > 0 {
		_ = json.Unmarshal(sa.WorkAspects, &resp.WorkAspects)
	}
	if len(sa.BehaviorAspects) > 0 {
		_ = json.Unmarshal(sa.BehaviorAspects, &resp.BehaviorAspects)
	}

	return resp
}
