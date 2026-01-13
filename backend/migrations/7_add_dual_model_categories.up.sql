-- Add v2 and v4 category columns to incidents table
ALTER TABLE incidents
ADD COLUMN v2_category_id INTEGER,
ADD COLUMN v4_category_id INTEGER,
ADD COLUMN models_agree BOOLEAN;

-- Add comments for clarity
COMMENT ON COLUMN incidents.v2_category_id IS 'Category ID from v2 model (1-9, accuracy 67.96%)';
COMMENT ON COLUMN incidents.v4_category_id IS 'Category ID from v4 model (1-9, accuracy 64.00%)';
COMMENT ON COLUMN incidents.models_agree IS 'Whether both models predicted the same category';

-- threat_type will continue to hold the primary category (from v2 model)
COMMENT ON COLUMN incidents.threat_type IS 'Primary threat category (from v2 model)';
