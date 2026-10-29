package assessment

import (
	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, h *Handler, cfg *config.Config) {
	authMiddleware := middleware.NewAuthMiddleware(cfg)

	// Academic Periods
	periodsGroup := router.Group("/academic-periods")
	periodsGroup.Get("", h.GetAcademicPeriods)
	periodsGroup.Get("/active", h.GetActiveAcademicPeriod)
	periodsGroup.Post("", authMiddleware, h.CreateAcademicPeriod)
	periodsGroup.Put("/:id/activate", authMiddleware, h.SetActiveAcademicPeriod)

	// Assessment Templates
	templatesGroup := router.Group("/assessment-templates")
	templatesGroup.Get("", authMiddleware, h.GetAssessmentTemplates)
	templatesGroup.Get("/:id", authMiddleware, h.GetAssessmentTemplateByID)
	templatesGroup.Post("", authMiddleware, h.CreateAssessmentTemplate)
	templatesGroup.Put("/:id", authMiddleware, h.UpdateAssessmentTemplate)

	// Student Assessments
	studentGroup := router.Group("/student-assessments")
	studentGroup.Get("/my", authMiddleware, h.GetMyStudentAssessment)
	studentGroup.Post("", authMiddleware, h.UpsertStudentAssessment)
	studentGroup.Get("", authMiddleware, h.ListStudentAssessments)
	studentGroup.Get("/:id", authMiddleware, h.GetStudentAssessmentByID)
}
