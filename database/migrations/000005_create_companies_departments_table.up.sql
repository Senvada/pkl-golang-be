CREATE TABLE companies_departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX idx_companies_departments_unique ON companies_departments(company_id, department_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_companies_departments_company ON companies_departments(company_id);
CREATE INDEX idx_companies_departments_department ON companies_departments(department_id);
