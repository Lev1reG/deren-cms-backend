-- Projects table
CREATE TABLE IF NOT EXISTS projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  href TEXT,
  technologies TEXT[] NOT NULL DEFAULT '{}',
  display_order INTEGER DEFAULT 0,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

-- Hero table (single row for hero content)
CREATE TABLE IF NOT EXISTS hero (
  id UUID PRIMARY KEY DEFAULT '00000000-0000-0000-0000-000000000001'::UUID,
  phrases TEXT[] NOT NULL DEFAULT '{}',
  description TEXT NOT NULL,
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Work experience table
CREATE TABLE IF NOT EXISTS work_experience (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company TEXT NOT NULL,
  position TEXT NOT NULL,
  date TEXT NOT NULL,
  description TEXT NOT NULL,
  href TEXT,
  type TEXT CHECK (type IN ('Freelance', 'Internship', 'Contract', 'Part-Time', 'Full-Time')),
  display_order INTEGER DEFAULT 0,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

-- Indexes for ordering
CREATE INDEX IF NOT EXISTS idx_projects_order ON projects(display_order) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_work_order ON work_experience(display_order) WHERE deleted_at IS NULL;

-- Insert default hero row
INSERT INTO hero (id, phrases, description)
VALUES (
  '00000000-0000-0000-0000-000000000001'::UUID,
  ARRAY['Software Engineer', 'Full Stack Developer'],
  'Welcome to my personal website.'
) ON CONFLICT (id) DO NOTHING;

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers
CREATE TRIGGER update_projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_work_experience_updated_at
    BEFORE UPDATE ON work_experience
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
