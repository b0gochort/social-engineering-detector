-- Remove dual model category columns from incidents table
ALTER TABLE incidents
DROP COLUMN IF EXISTS v2_category_id,
DROP COLUMN IF EXISTS v4_category_id,
DROP COLUMN IF EXISTS models_agree;
