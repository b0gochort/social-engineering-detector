-- Remove access control fields from incidents table
DROP INDEX IF EXISTS idx_incidents_access_granted;
ALTER TABLE incidents DROP COLUMN IF EXISTS current_access_request_id;
ALTER TABLE incidents DROP COLUMN IF EXISTS access_granted;
