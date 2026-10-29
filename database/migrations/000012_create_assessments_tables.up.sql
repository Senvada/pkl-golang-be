CREATE TABLE academic_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX idx_academic_periods_name_active ON academic_periods(name) WHERE deleted_at IS NULL;

CREATE TABLE assessment_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    period_id UUID NOT NULL REFERENCES academic_periods(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    work_aspects JSONB NOT NULL DEFAULT '[]',
    behavior_aspects JSONB NOT NULL DEFAULT '[]',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX idx_assessment_templates_dept_period ON assessment_templates(department_id, period_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_assessment_templates_dept ON assessment_templates(department_id);
CREATE INDEX idx_assessment_templates_period ON assessment_templates(period_id);

CREATE TABLE student_assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE RESTRICT,
    period_id UUID NOT NULL REFERENCES academic_periods(id) ON DELETE RESTRICT,
    template_id UUID REFERENCES assessment_templates(id) ON DELETE SET NULL,
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    student_name VARCHAR(255) NOT NULL,
    student_nis VARCHAR(50),
    student_class VARCHAR(50),
    school_name VARCHAR(255) NOT NULL DEFAULT 'SMK Negeri 6 Malang',
    school_address VARCHAR(255) NOT NULL DEFAULT 'Jl. Ki Ageng Gribig 28 Malang',
    school_phone VARCHAR(50) NOT NULL DEFAULT '(0341) 722216',
    school_email VARCHAR(100) NOT NULL DEFAULT 'humas@smkn6malang.sch.id',
    department_name VARCHAR(100) NOT NULL,
    company_name VARCHAR(255) NOT NULL,
    company_address TEXT,
    supervisor_name VARCHAR(255) NOT NULL,
    supervisor_nip VARCHAR(100),
    assessment_city VARCHAR(100) NOT NULL DEFAULT 'Malang',
    assessment_date DATE,
    work_aspects JSONB NOT NULL DEFAULT '[]',
    behavior_aspects JSONB NOT NULL DEFAULT '[]',
    work_average NUMERIC(5, 2) NOT NULL DEFAULT 0,
    behavior_average NUMERIC(5, 2) NOT NULL DEFAULT 0,
    overall_average NUMERIC(5, 2) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'verified')),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX idx_student_assessments_student_period ON student_assessments(student_id, period_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_student_assessments_student ON student_assessments(student_id);
CREATE INDEX idx_student_assessments_dept ON student_assessments(department_id);
CREATE INDEX idx_student_assessments_period ON student_assessments(period_id);
CREATE INDEX idx_student_assessments_company ON student_assessments(company_id);
CREATE INDEX idx_student_assessments_status ON student_assessments(status);
