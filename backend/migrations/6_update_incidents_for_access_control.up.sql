-- Add access control fields to incidents table
ALTER TABLE incidents
    ADD COLUMN access_granted BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN current_access_request_id INTEGER REFERENCES access_requests(id) ON DELETE SET NULL;

-- Create index for faster access checks
CREATE INDEX idx_incidents_access_granted ON incidents(access_granted);
